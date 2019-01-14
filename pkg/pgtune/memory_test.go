package pgtune

import (
	"math"
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// memoryToBaseVals provides a memory from test memory levels to expected "base"
// memory settings. These "base" values are the values if there is only 1 CPU.
// Most settings are unaffected by number of CPUs; the exception is work_mem,
// so the adjustment is done in the init function
var memoryToBaseVals = map[uint64]map[string]uint64{
	10 * parse.Gigabyte: map[string]uint64{
		SharedBuffersKey:      2560 * parse.Megabyte,
		EffectiveCacheKey:     7680 * parse.Megabyte,
		MaintenanceWorkMemKey: 1280 * parse.Megabyte,
		WorkMemKey:            64 * parse.Megabyte,
	},
	12 * parse.Gigabyte: map[string]uint64{
		SharedBuffersKey:      3 * parse.Gigabyte,
		EffectiveCacheKey:     9 * parse.Gigabyte,
		MaintenanceWorkMemKey: 1536 * parse.Megabyte,
		WorkMemKey:            78643 * parse.Kilobyte,
	},
	32 * parse.Gigabyte: map[string]uint64{

		SharedBuffersKey:      8 * parse.Gigabyte,
		EffectiveCacheKey:     24 * parse.Gigabyte,
		MaintenanceWorkMemKey: 2 * parse.Gigabyte,
		WorkMemKey:            209715 * parse.Kilobyte,
	},
}

// cpuVals is the different amounts of CPUs to test
var cpuVals = []int{1, 4, 5}

// memorySettingsMatrix stores the test cases for MemoryRecommend along with
// the expected values
var memorySettingsMatrix = map[uint64]map[int]map[string]string{}

func init() {
	for mem, baseMatrix := range memoryToBaseVals {
		memorySettingsMatrix[mem] = make(map[int]map[string]string)
		for _, cpus := range cpuVals {
			memorySettingsMatrix[mem][cpus] = make(map[string]string)
			cpuFactor := uint64(math.Round(float64(cpus) / 2.0))

			memorySettingsMatrix[mem][cpus][SharedBuffersKey] = parse.BytesToPGFormat(baseMatrix[SharedBuffersKey])
			memorySettingsMatrix[mem][cpus][EffectiveCacheKey] = parse.BytesToPGFormat(baseMatrix[EffectiveCacheKey])
			memorySettingsMatrix[mem][cpus][MaintenanceWorkMemKey] = parse.BytesToPGFormat(baseMatrix[MaintenanceWorkMemKey])
			memorySettingsMatrix[mem][cpus][WorkMemKey] = parse.BytesToPGFormat(baseMatrix[WorkMemKey] / cpuFactor)
		}
	}
}

func TestNewMemoryRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		cpus := rand.Intn(128)
		r := NewMemoryRecommender(mem, cpus)
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

func TestMemoryRecommenderRecommendWindows(t *testing.T) {
	cases := []struct {
		desc        string
		totalMemory uint64
		cpus        int
		want        string
	}{
		{
			desc:        "1GB",
			totalMemory: 1 * parse.Gigabyte,
			cpus:        1,
			want:        "6553" + parse.KB, // from pgtune
		},
		{
			desc:        "1GB, 4 cpus",
			totalMemory: 1 * parse.Gigabyte,
			cpus:        4,
			want:        "3276" + parse.KB, // from pgtune
		},
		{
			desc:        "2GB",
			totalMemory: 2 * parse.Gigabyte,
			cpus:        1,
			want:        "13107" + parse.KB, // from pgtune
		},
		{
			desc:        "2GB, 5 cpus",
			totalMemory: 2 * parse.Gigabyte,
			cpus:        5,
			want:        "4369" + parse.KB, // from pgtune
		},
		{
			desc:        "3GB",
			totalMemory: 3 * parse.Gigabyte,
			cpus:        1,
			want:        "21845" + parse.KB, // from pgtune
		},
		{
			desc:        "3GB, 3 cpus",
			totalMemory: 3 * parse.Gigabyte,
			cpus:        3,
			want:        "10922" + parse.KB, // from pgtune
		},
		{
			desc:        "8GB",
			totalMemory: 8 * parse.Gigabyte,
			cpus:        1,
			want:        "64" + parse.MB, // from pgtune
		},
		{
			desc:        "8GB, 8 cpus",
			totalMemory: 8 * parse.Gigabyte,
			cpus:        8,
			want:        "16" + parse.MB, // from pgtune
		},
		{
			desc:        "16GB",
			totalMemory: 16 * parse.Gigabyte,
			cpus:        1,
			want:        "135441" + parse.KB, // from pgtune
		},
		{
			desc:        "16GB, 10 cpus",
			totalMemory: 16 * parse.Gigabyte,
			cpus:        10,
			want:        "27088" + parse.KB, // from pgtune
		},
	}

	for _, c := range cases {
		mr := &MemoryRecommender{c.totalMemory, c.cpus}
		if got := mr.recommendWindows(); got != c.want {
			t.Errorf("%s: incorrect value: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestMemoryRecommenderRecommend(t *testing.T) {
	for totalMemory, outerMatrix := range memorySettingsMatrix {
		for cpus, cases := range outerMatrix {
			mr := &MemoryRecommender{totalMemory, cpus}
			testRecommender(t, mr, cases)
		}
	}
}

func TestMemoryRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &MemoryRecommender{1, 1}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestMemorySettingsGroup(t *testing.T) {
	for totalMemory, outerMatrix := range memorySettingsMatrix {
		for cpus, matrix := range outerMatrix {
			config := NewSystemConfig(totalMemory, cpus, "10")
			sg := GetSettingsGroup(MemoryLabel, config)
			testSettingGroup(t, sg, matrix, MemoryLabel, MemoryKeys)
		}
	}
}
