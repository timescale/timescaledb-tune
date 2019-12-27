package pgtune

import (
	"fmt"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// Keys in the conf file that are tuned related to the WAL
const (
	WALBuffersKey = "wal_buffers"
	MinWALKey     = "min_wal_size"
	MaxWALKey     = "max_wal_size"

	walBuffersThreshold = 2 * parse.Gigabyte
	walBuffersDefault   = 16 * parse.Megabyte
	defaultMaxWALBytes  = 1 * parse.Gigabyte
)

// WALLabel is the label used to refer to the WAL settings group
const WALLabel = "WAL"

// WALKeys is an array of keys that are tunable for the WAL
var WALKeys = []string{
	WALBuffersKey,
	MinWALKey,
	MaxWALKey,
}

// WALRecommender gives recommendations for WALKeys based on system resources
type WALRecommender struct {
	totalMemory uint64
	walDiskSize uint64
}

// NewWALRecommender returns a WALRecommender that recommends based on the given
// totalMemory bytes.
func NewWALRecommender(totalMemory, walDiskSize uint64) *WALRecommender {
	return &WALRecommender{
		totalMemory: totalMemory,
		walDiskSize: walDiskSize,
	}
}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *WALRecommender) IsAvailable() bool {
	return true
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *WALRecommender) Recommend(key string) string {
	var val string
	if key == WALBuffersKey {
		if r.totalMemory < walBuffersThreshold {
			temp := (float64(r.totalMemory) / float64(parse.Gigabyte)) * (7864.0 * float64(parse.Kilobyte))
			val = parse.BytesToPGFormat(uint64(temp))
		} else {
			val = parse.BytesToPGFormat(walBuffersDefault)
		}
	} else if key == MinWALKey {
		temp := r.calcMaxWALBytes() / 2
		val = parse.BytesToPGFormat(temp)
	} else if key == MaxWALKey {
		temp := r.calcMaxWALBytes()
		val = parse.BytesToPGFormat(temp)
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}

func (r *WALRecommender) calcMaxWALBytes() uint64 {
	// If disk size is not given, just use default
	if r.walDiskSize == 0 {
		return defaultMaxWALBytes
	}

	// With size given, we want to take up at most 80% of it, to give
	// additional room for safety.
	max := uint64(r.walDiskSize*80) / 100

	// WAL segments are 16MB, so it doesn't make sense not to round
	// up to the nearest 16MB boundary.
	if max%(16*parse.Megabyte) != 0 {
		max = (max/(16*parse.Megabyte) + 1) * 16 * parse.Megabyte
	}
	return max
}

// WALSettingsGroup is the SettingsGroup to represent settings that affect WAL usage.
type WALSettingsGroup struct {
	totalMemory uint64
	walDiskSize uint64
}

// Label should always return the value WALLabel.
func (sg *WALSettingsGroup) Label() string { return WALLabel }

// Keys should always return the WALKeys slice.
func (sg *WALSettingsGroup) Keys() []string { return WALKeys }

// GetRecommender should return a new WALRecommender.
func (sg *WALSettingsGroup) GetRecommender() Recommender {
	return NewWALRecommender(sg.totalMemory, sg.walDiskSize)
}
