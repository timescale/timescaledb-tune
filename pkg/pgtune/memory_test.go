package pgtune

import (
	"math"
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgutils"
)

// memoryToBaseVals provides a memory from test memory levels to expected "base"
// memory settings. These "base" values are the values if there is only 1 CPU
// and 20 max connections to the database. Most settings are actually
// unaffected by number of CPUs and max connections; the exception is work_mem,
// so the adjustment is done in the init function
var memoryToBaseVals = map[uint64]map[string]uint64{
	10 * parse.Gigabyte: {
		SharedBuffersKey:      2560 * parse.Megabyte,
		EffectiveCacheKey:     7680 * parse.Megabyte,
		MaintenanceWorkMemKey: 1280 * parse.Megabyte,
		WorkMemKey:            64 * parse.Megabyte,
	},
	12 * parse.Gigabyte: {
		SharedBuffersKey:      3 * parse.Gigabyte,
		EffectiveCacheKey:     9 * parse.Gigabyte,
		MaintenanceWorkMemKey: 1536 * parse.Megabyte,
		WorkMemKey:            78643 * parse.Kilobyte,
	},
	32 * parse.Gigabyte: {
		SharedBuffersKey:      8 * parse.Gigabyte,
		EffectiveCacheKey:     24 * parse.Gigabyte,
		MaintenanceWorkMemKey: maintenanceWorkMemLimit,
		WorkMemKey:            209715 * parse.Kilobyte,
	},
}

// highCPUs is the number of CPUs that is high enough that work_mem would normally
// fall below the minimum (64KB) using the standard formula
const highMilliCPUs = 9000 * pgutils.MilliScaleFactor

var (
	// cpuVals is the different amounts of CPUs to test
	milliCPUVals = []int{1 * pgutils.MilliScaleFactor, 4 * pgutils.MilliScaleFactor, 5 * pgutils.MilliScaleFactor, highMilliCPUs, 500, 100}
	// connVals is the different number of conns to test
	connVals = []uint64{0, 19, 20, 50}
	// memorySettingsMatrix stores the test cases for MemoryRecommend along with
	// the expected values
	memorySettingsMatrix = map[uint64]map[int]map[uint64]map[string]string{}
)

func init() {
	for mem, baseMatrix := range memoryToBaseVals {
		memorySettingsMatrix[mem] = make(map[int]map[uint64]map[string]string)
		for _, milliCPUs := range milliCPUVals {
			memorySettingsMatrix[mem][milliCPUs] = make(map[uint64]map[string]string)
			for _, conns := range connVals {
				memorySettingsMatrix[mem][milliCPUs][conns] = make(map[string]string)

				memorySettingsMatrix[mem][milliCPUs][conns][SharedBuffersKey] = parse.BytesToPGFormat(baseMatrix[SharedBuffersKey])
				memorySettingsMatrix[mem][milliCPUs][conns][EffectiveCacheKey] = parse.BytesToPGFormat(baseMatrix[EffectiveCacheKey])
				memorySettingsMatrix[mem][milliCPUs][conns][MaintenanceWorkMemKey] = parse.BytesToPGFormat(baseMatrix[MaintenanceWorkMemKey])

				if milliCPUs == highMilliCPUs {
					memorySettingsMatrix[mem][milliCPUs][conns][WorkMemKey] = parse.BytesToPGFormat(workMemMin)
				} else {
					// CPU only affects work_mem in groups of 2 (i.e. 2 and 3 CPUs are treated as the same)
					cpuFactor := math.Max(math.Round(float64(milliCPUs)/(2.0*pgutils.MilliScaleFactor)), 1)
					// Our work_mem values are derived by a certain amount of memory lost/gained when
					// moving away from baseConns
					connFactor := float64(MaxConnectionsDefault) / float64(baseConns)
					if conns != 0 {
						connFactor = float64(conns) / float64(baseConns)
					}

					memorySettingsMatrix[mem][milliCPUs][conns][WorkMemKey] =
						parse.BytesToPGFormat(uint64(float64(baseMatrix[WorkMemKey]) / connFactor / cpuFactor))
				}
			}
		}
	}
}

func TestNewMemoryRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		milliCPus := rand.Intn(128) * pgutils.MilliScaleFactor
		r := NewMemoryRecommender(mem, milliCPus, MaxConnectionsDefault)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect memory: got %d want %d", got, mem)
		}
		if got := r.milliCPUs; got != milliCPus {
			t.Errorf("recommender has incorrect milliCpus: got %d want %d", got, milliCPus)
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
		milliCPUs   int
		conns       uint64
		want        string
	}{
		{
			desc:        "1GB",
			totalMemory: 1 * parse.Gigabyte,
			milliCPUs:   1 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "6553" + parse.KB, // from pgtune
		},
		{
			desc:        "1GB with fractions of CPU",
			totalMemory: 1 * parse.Gigabyte,
			milliCPUs:   500,
			conns:       baseConns,
			want:        "6553" + parse.KB, // from pgtune
		},
		{
			desc:        "1GB, 10 conns",
			totalMemory: 1 * parse.Gigabyte,
			milliCPUs:   1 * pgutils.MilliScaleFactor,
			conns:       10,
			want:        "13107" + parse.KB, // from pgtune
		},
		{
			desc:        "1GB, 4 cpus",
			totalMemory: 1 * parse.Gigabyte,
			milliCPUs:   4 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "3276" + parse.KB, // from pgtune
		},
		{
			desc:        "2GB",
			totalMemory: 2 * parse.Gigabyte,
			milliCPUs:   1 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "13107" + parse.KB, // from pgtune
		},
		{
			desc:        "2GB, 5 cpus",
			totalMemory: 2 * parse.Gigabyte,
			milliCPUs:   5 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "4369" + parse.KB, // from pgtune
		},
		{
			desc:        "3GB",
			totalMemory: 3 * parse.Gigabyte,
			milliCPUs:   1 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "21845" + parse.KB, // from pgtune
		},
		{
			desc:        "3GB, 3 cpus",
			totalMemory: 3 * parse.Gigabyte,
			milliCPUs:   3 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "10922" + parse.KB, // from pgtune
		},
		{
			desc:        "8GB",
			totalMemory: 8 * parse.Gigabyte,
			milliCPUs:   1 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "64" + parse.MB, // from pgtune
		},
		{
			desc:        "8GB, 8 cpus",
			totalMemory: 8 * parse.Gigabyte,
			milliCPUs:   8 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "16" + parse.MB, // from pgtune
		},
		{
			desc:        "16GB",
			totalMemory: 16 * parse.Gigabyte,
			milliCPUs:   1 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "135441" + parse.KB, // from pgtune
		},
		{
			desc:        "16GB, 10 cpus",
			totalMemory: 16 * parse.Gigabyte,
			milliCPUs:   10 * pgutils.MilliScaleFactor,
			conns:       baseConns,
			want:        "27088" + parse.KB, // from pgtune
		},
		{
			desc:        "1GB, 9000 cpus",
			totalMemory: parse.Gigabyte,
			milliCPUs:   highMilliCPUs,
			conns:       baseConns,
			want:        "64" + parse.KB,
		},
	}

	for _, c := range cases {
		mr := NewMemoryRecommender(c.totalMemory, c.milliCPUs, c.conns)
		if got := mr.recommendWindows(); got != c.want {
			t.Errorf("%s: incorrect value: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestMemoryRecommenderRecommend(t *testing.T) {
	for totalMemory, cpuMatrix := range memorySettingsMatrix {
		for cpus, connMatrix := range cpuMatrix {
			for conns, cases := range connMatrix {
				mr := NewMemoryRecommender(totalMemory, cpus, conns)
				testRecommender(t, mr, MemoryKeys, cases)
			}
		}
	}
}

func TestMemoryRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := NewMemoryRecommender(1, 1, 1)
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestMemorySettingsGroup(t *testing.T) {
	for totalMemory, cpuMatrix := range memorySettingsMatrix {
		for cpus, connMatrix := range cpuMatrix {
			for conns, matrix := range connMatrix {
				config := getDefaultTestSystemConfig(t)
				config.MilliCPUs = cpus
				config.Memory = totalMemory
				config.maxConns = conns

				sg := GetSettingsGroup(MemoryLabel, config)
				testSettingGroup(t, sg, matrix, MemoryLabel, MemoryKeys)
			}
		}
	}
}
