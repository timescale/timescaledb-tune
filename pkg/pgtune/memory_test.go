package pgtune

import (
	"math"
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// defaultMemoryToBaseVals provides a memory from test memory levels to expected "base"
// memory settings. These "base" values are the values if there is only 1 CPU
// and 20 max connections to the database. Most settings are actually
// unaffected by number of CPUs and max connections; the exception is work_mem,
// so the adjustment is done in the init function
var defaultMemoryToBaseVals = map[uint64]map[string]uint64{
	10 * parse.Gigabyte: {
		SharedBuffersKey:      2560 * parse.Megabyte,
		EffectiveCacheKey:     7680 * parse.Megabyte,
		MaintenanceWorkMemKey: 1280 * parse.Megabyte,
		WorkMemKey:            200 * parse.Megabyte,
	},
	12 * parse.Gigabyte: {
		SharedBuffersKey:      3 * parse.Gigabyte,
		EffectiveCacheKey:     9 * parse.Gigabyte,
		MaintenanceWorkMemKey: 1536 * parse.Megabyte,
		WorkMemKey:            240 * parse.Megabyte,
	},
	32 * parse.Gigabyte: {
		SharedBuffersKey:      8 * parse.Gigabyte,
		EffectiveCacheKey:     24 * parse.Gigabyte,
		MaintenanceWorkMemKey: maintenanceWorkMemLimit,
		WorkMemKey:            640 * parse.Megabyte,
	},
}

// promscaleMemoryToBaseVals provides a memory from test memory levels to expected "base"
// memory settings. These "base" values are the values if there is only 1 CPU
// and 20 max connections to the database. Most settings are actually
// unaffected by number of CPUs and max connections; the exception is work_mem,
// so the adjustment is done in the init function
var promscaleMemoryToBaseVals = map[uint64]map[string]uint64{
	10 * parse.Gigabyte: {
		SharedBuffersKey:      5120 * parse.Megabyte,
		EffectiveCacheKey:     7680 * parse.Megabyte,
		MaintenanceWorkMemKey: 1280 * parse.Megabyte,
		WorkMemKey:            200 * parse.Megabyte,
	},
	12 * parse.Gigabyte: {
		SharedBuffersKey:      6 * parse.Gigabyte,
		EffectiveCacheKey:     9 * parse.Gigabyte,
		MaintenanceWorkMemKey: 1536 * parse.Megabyte,
		WorkMemKey:            240 * parse.Megabyte,
	},
	32 * parse.Gigabyte: {
		SharedBuffersKey:      16 * parse.Gigabyte,
		EffectiveCacheKey:     24 * parse.Gigabyte,
		MaintenanceWorkMemKey: maintenanceWorkMemLimit,
		WorkMemKey:            640 * parse.Megabyte,
	},
}

// highCPUs is the number of CPUs that is high enough that work_mem would normally
// fall below the minimum (64KB) using the standard formula
const highCPUs = 9000

var (
	// cpuVals is the different amounts of CPUs to test
	cpuVals = []int{1, 4, 5, highCPUs}
	// connVals is the different number of conns to test
	connVals = []uint64{0, 19, 20, 50}
	// defaultMemorySettingsMatrix stores the test cases for MemoryRecommend along with
	// the expected values
	defaultMemorySettingsMatrix = map[uint64]map[int]map[uint64]map[string]string{}
	// promscaleMemorySettingsMatrix stores the test cases for PromscaleMemoryRecommend along with
	// the expected values
	promscaleMemorySettingsMatrix = map[uint64]map[int]map[uint64]map[string]string{}
)

func init() {
	for mem, baseMatrix := range defaultMemoryToBaseVals {
		defaultMemorySettingsMatrix[mem] = make(map[int]map[uint64]map[string]string)
		for _, cpus := range cpuVals {
			defaultMemorySettingsMatrix[mem][cpus] = make(map[uint64]map[string]string)
			for _, conns := range connVals {
				defaultMemorySettingsMatrix[mem][cpus][conns] = make(map[string]string)

				defaultMemorySettingsMatrix[mem][cpus][conns][SharedBuffersKey] = parse.BytesToPGFormat(baseMatrix[SharedBuffersKey])
				defaultMemorySettingsMatrix[mem][cpus][conns][EffectiveCacheKey] = parse.BytesToPGFormat(baseMatrix[EffectiveCacheKey])
				defaultMemorySettingsMatrix[mem][cpus][conns][MaintenanceWorkMemKey] = parse.BytesToPGFormat(baseMatrix[MaintenanceWorkMemKey])

				// CPU only affects work_mem in groups of 2 (i.e. 2 and 3 CPUs are treated as the same)
				cpuFactor := math.Round(float64(cpus) / 2.0)
				// Our work_mem values are derivied by a certain amount of memory lost/gained when
				// moving away from baseConns
				connFactor := float64(MaxConnectionsDefault) / float64(baseConns)
				if conns != 0 {
					connFactor = float64(conns) / float64(baseConns)
				}

				wm := uint64(float64(baseMatrix[WorkMemKey]) / connFactor / cpuFactor)
				if wm < workMemMin {
					wm = workMemMin
				}
				defaultMemorySettingsMatrix[mem][cpus][conns][WorkMemKey] = parse.BytesToPGFormat(wm)
			}
		}
	}

	for mem, baseMatrix := range promscaleMemoryToBaseVals {
		promscaleMemorySettingsMatrix[mem] = make(map[int]map[uint64]map[string]string)
		for _, cpus := range cpuVals {
			promscaleMemorySettingsMatrix[mem][cpus] = make(map[uint64]map[string]string)
			for _, conns := range connVals {
				promscaleMemorySettingsMatrix[mem][cpus][conns] = make(map[string]string)

				promscaleMemorySettingsMatrix[mem][cpus][conns][SharedBuffersKey] = parse.BytesToPGFormat(baseMatrix[SharedBuffersKey])
				promscaleMemorySettingsMatrix[mem][cpus][conns][EffectiveCacheKey] = parse.BytesToPGFormat(baseMatrix[EffectiveCacheKey])
				promscaleMemorySettingsMatrix[mem][cpus][conns][MaintenanceWorkMemKey] = parse.BytesToPGFormat(baseMatrix[MaintenanceWorkMemKey])

				// CPU only affects work_mem in groups of 2 (i.e. 2 and 3 CPUs are treated as the same)
				cpuFactor := math.Round(float64(cpus) / 2.0)
				// Our work_mem values are derivied by a certain amount of memory lost/gained when
				// moving away from baseConns
				connFactor := float64(MaxConnectionsDefault) / float64(baseConns)
				if conns != 0 {
					connFactor = float64(conns) / float64(baseConns)
				}

				wm := uint64(float64(baseMatrix[WorkMemKey]) / connFactor / cpuFactor)
				if wm < workMemMin {
					wm = workMemMin
				}
				promscaleMemorySettingsMatrix[mem][cpus][conns][WorkMemKey] = parse.BytesToPGFormat(wm)
			}
		}
	}
}

func TestNewMemoryRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		cpus := rand.Intn(128)
		r := NewMemoryRecommender(mem, cpus, MaxConnectionsDefault)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}
		if got := r.cpus; got != cpus {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestNewPromscaleMemoryRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		cpus := rand.Intn(128)
		r := NewPromscaleMemoryRecommender(mem, cpus, MaxConnectionsDefault)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}
		if got := r.cpus; got != cpus {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestMemoryRecommenderRecommend(t *testing.T) {
	for totalMemory, cpuMatrix := range defaultMemorySettingsMatrix {
		for cpus, connMatrix := range cpuMatrix {
			for conns, cases := range connMatrix {
				mr := NewMemoryRecommender(totalMemory, cpus, conns)
				testRecommender(t, mr, MemoryKeys, cases)
			}
		}
	}
}

func TestPromscaleMemoryRecommenderRecommend(t *testing.T) {
	for totalMemory, cpuMatrix := range promscaleMemorySettingsMatrix {
		for cpus, connMatrix := range cpuMatrix {
			for conns, cases := range connMatrix {
				mr := NewPromscaleMemoryRecommender(totalMemory, cpus, conns)
				testRecommender(t, mr, MemoryKeys, cases)
			}
		}
	}
}

func TestMemoryRecommenderNoRecommendation(t *testing.T) {
	r := NewMemoryRecommender(1, 1, 1)
	if r.Recommend("foo") != NoRecommendation {
		t.Error("Recommendation was provided when there should have been none")
	}
}

func TestPromscaleMemoryRecommenderNoRecommendation(t *testing.T) {
	r := NewPromscaleMemoryRecommender(1, 1, 1)
	if r.Recommend("foo") != NoRecommendation {
		t.Error("Recommendation was provided when there should have been none")
	}
}

func TestMemorySettingsGroup(t *testing.T) {
	for totalMemory, cpuMatrix := range defaultMemorySettingsMatrix {
		for cpus, connMatrix := range cpuMatrix {
			for conns, matrix := range connMatrix {
				config := getDefaultTestSystemConfig(t)
				config.CPUs = cpus
				config.Memory = totalMemory
				config.maxConns = conns

				sg := GetSettingsGroup(MemoryLabel, config)
				testSettingGroup(t, sg, DefaultProfile, matrix, MemoryLabel, MemoryKeys)
			}
		}
	}
}

func TestPromscaleMemorySettingsGroup(t *testing.T) {
	for totalMemory, cpuMatrix := range promscaleMemorySettingsMatrix {
		for cpus, connMatrix := range cpuMatrix {
			for conns, matrix := range connMatrix {
				config := getDefaultTestSystemConfig(t)
				config.CPUs = cpus
				config.Memory = totalMemory
				config.maxConns = conns

				sg := GetSettingsGroup(MemoryLabel, config)
				testSettingGroup(t, sg, PromscaleProfile, matrix, MemoryLabel, MemoryKeys)
			}
		}
	}
}
