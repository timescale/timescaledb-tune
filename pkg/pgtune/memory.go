package pgtune

import (
	"math"
	"runtime"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// Keys in the conf file that are tuned related to memory
const (
	SharedBuffersKey      = "shared_buffers"
	EffectiveCacheKey     = "effective_cache_size"
	MaintenanceWorkMemKey = "maintenance_work_mem"
	WorkMemKey            = "work_mem"

	// the limit is 2GB on Unix, but 2047MB on Windows, so using 2047MB is easier all around
	maintenanceWorkMemLimit     = 2047 * parse.Megabyte
	sharedBuffersWindows        = 512 * parse.Megabyte
	baseConns                   = 20
	workMemMin                  = 64 * parse.Kilobyte
	workMemPerGigPerConn        = 6.4 * baseConns     // derived from pgtune results
	workMemPerGigPerConnWindows = 8.53336 * baseConns // derived from pgtune results
)

// MemoryLabel is the label used to refer to the memory settings group
const MemoryLabel = "memory"

// MemoryKeys is an array of keys that are tunable for memory
var MemoryKeys = []string{
	SharedBuffersKey,
	EffectiveCacheKey,
	MaintenanceWorkMemKey,
	WorkMemKey,
}

// MemoryRecommender gives recommendations for ParallelKeys based on system resources
type MemoryRecommender struct {
	totalMemory uint64
	cpus        int
	conns       uint64
}

// NewMemoryRecommender returns a MemoryRecommender that recommends based on the given
// number of cpus and system memory
func NewMemoryRecommender(totalMemory uint64, cpus int, maxConns uint64) *MemoryRecommender {
	conns := maxConns
	if conns == 0 {
		conns = getMaxConns(totalMemory)
	}
	return &MemoryRecommender{totalMemory, cpus, conns}
}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *MemoryRecommender) IsAvailable() bool {
	return true
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *MemoryRecommender) Recommend(key string) string {
	var val string
	if key == SharedBuffersKey {
		if runtime.GOOS == osWindows {
			val = parse.BytesToPGFormat(sharedBuffersWindows)
		} else {
			val = parse.BytesToPGFormat(r.totalMemory / 4)
		}
	} else if key == EffectiveCacheKey {
		val = parse.BytesToPGFormat((r.totalMemory * 3) / 4)
	} else if key == MaintenanceWorkMemKey {
		temp := (float64(r.totalMemory) / float64(parse.Gigabyte)) * (128.0 * float64(parse.Megabyte))
		if temp > maintenanceWorkMemLimit {
			temp = maintenanceWorkMemLimit
		}
		val = parse.BytesToPGFormat(uint64(temp))
	} else if key == WorkMemKey {
		if runtime.GOOS == osWindows {
			val = r.recommendWindows()
		} else {
			cpuFactor := math.Round(float64(r.cpus) / 2.0)
			gigs := float64(r.totalMemory) / float64(parse.Gigabyte)
			temp := uint64(gigs * (workMemPerGigPerConn * float64(parse.Megabyte) / float64(r.conns)) / cpuFactor)
			if temp < workMemMin {
				temp = workMemMin
			}
			val = parse.BytesToPGFormat(temp)
		}

	} else {
		val = NoRecommendation
	}
	return val
}

func (r *MemoryRecommender) recommendWindows() string {
	cpuFactor := math.Round(float64(r.cpus) / 2.0)
	var temp uint64

	if r.totalMemory <= 2*parse.Gigabyte {
		gigs := float64(r.totalMemory) / float64(parse.Gigabyte)
		temp = uint64(gigs * (workMemPerGigPerConn * float64(parse.Megabyte) / float64(r.conns)) / cpuFactor)
	} else {
		base := 2.0 * workMemPerGigPerConn * float64(parse.Megabyte)
		gigs := float64(r.totalMemory)/float64(parse.Gigabyte) - 2.0
		temp = uint64(((gigs*(workMemPerGigPerConnWindows*float64(parse.Megabyte)) + base) / float64(r.conns)) / cpuFactor)
	}
	if temp < workMemMin {
		temp = workMemMin
	}
	return parse.BytesToPGFormat(temp)
}

// PromscaleMemoryRecommender gives recommendations for ParallelKeys based on system resources
type PromscaleMemoryRecommender struct {
	*MemoryRecommender
}

// NewPromscaleMemoryRecommender returns a PromscaleMemoryRecommender that recommends based on the given
// number of cpus and system memory
func NewPromscaleMemoryRecommender(totalMemory uint64, cpus int, maxConns uint64) *PromscaleMemoryRecommender {
	return &PromscaleMemoryRecommender{
		MemoryRecommender: NewMemoryRecommender(totalMemory, cpus, maxConns),
	}
}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *PromscaleMemoryRecommender) IsAvailable() bool {
	return true
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *PromscaleMemoryRecommender) Recommend(key string) string {
	var val string
	switch key {
	case SharedBuffersKey:
		if runtime.GOOS == osWindows {
			val = parse.BytesToPGFormat(sharedBuffersWindows)
		} else {
			val = parse.BytesToPGFormat(r.totalMemory / 2)
		}
	default:
		val = r.MemoryRecommender.Recommend(key)
	}
	return val
}

// MemorySettingsGroup is the SettingsGroup to represent settings that affect memory usage.
type MemorySettingsGroup struct {
	totalMemory uint64
	cpus        int
	maxConns    uint64
}

// Label should always return the value MemoryLabel.
func (sg *MemorySettingsGroup) Label() string { return MemoryLabel }

// Keys should always return the MemoryKeys slice.
func (sg *MemorySettingsGroup) Keys() []string { return MemoryKeys }

// GetRecommender should return a new MemoryRecommender.
func (sg *MemorySettingsGroup) GetRecommender(profile Profile) Recommender {
	switch profile {
	case PromscaleProfile:
		return NewPromscaleMemoryRecommender(sg.totalMemory, sg.cpus, sg.maxConns)
	default:
		return NewMemoryRecommender(sg.totalMemory, sg.cpus, sg.maxConns)
	}
}
