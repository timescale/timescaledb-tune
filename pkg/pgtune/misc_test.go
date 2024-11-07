package pgtune

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgutils"
)

// memoryToLocks is a mapping of the different memory levels we want tested and
// the corresponding number of locks at that level.
var memoryToLocks = map[uint64]string{
	7 * parse.Gigabyte:  maxLocksValues[0],
	8 * parse.Gigabyte:  maxLocksValues[1],
	15 * parse.Gigabyte: maxLocksValues[1],
	16 * parse.Gigabyte: maxLocksValues[2],
	24 * parse.Gigabyte: maxLocksValues[2],
	32 * parse.Gigabyte: maxLocksValues[3],
	80 * parse.Gigabyte: maxLocksValues[3],
}

// connsToMaxConns is a mapping of the user given connection values we want
// tested and the corresponding number of actual max connections assigned.
var connsToMaxConns = map[uint64]uint64{
	MaxConnectionsDefault - 10: MaxConnectionsDefault - 10,
	MaxConnectionsDefault:      MaxConnectionsDefault,
	MaxConnectionsDefault + 10: MaxConnectionsDefault + 10,
}

// miscSettingsMatrix is a matrix that holds the test cases and desired key/value
// pairs. The first key is the memory level (uint64), the second is the user
// given connections (uint64), and the innermost map is the key-value pairs
// we expect
var miscSettingsMatrix = map[uint64]map[uint64]map[string]string{}

func init() {
	// Initialize the miscSettingsMatrix by creating a key-value map for every
	// memory level for every connections given
	for mem, maxLocks := range memoryToLocks {
		miscSettingsMatrix[mem] = make(map[uint64]map[string]string)
		for conns, maxConns := range connsToMaxConns {
			miscSettingsMatrix[mem][conns] = make(map[string]string)
			miscSettingsMatrix[mem][conns][MaxLocksPerTxKey] = maxLocks
			miscSettingsMatrix[mem][conns][MaxConnectionsKey] = fmt.Sprintf("%d", maxConns)

			miscSettingsMatrix[mem][conns][CheckpointKey] = checkpointDefault
			miscSettingsMatrix[mem][conns][StatsTargetKey] = statsTargetDefault
			miscSettingsMatrix[mem][conns][RandomPageCostKey] = randomPageCostDefault
			miscSettingsMatrix[mem][conns][EffectiveIOKey] = effectiveIODefaultOldVersions
			miscSettingsMatrix[mem][conns][AutovacuumMaxWorkersKey] = autovacuumMaxWorkersDefault
			miscSettingsMatrix[mem][conns][AutovacuumNaptimeKey] = autovacuumNaptimeDefault
		}
	}
}

func TestGetMaxConns(t *testing.T) {
	cases := []struct {
		desc string
		mem  uint64
		want uint64
	}{
		{
			desc: "really small instance (1GB)",
			mem:  1 * parse.Gigabyte,
			want: minMaxConns,
		},
		{
			desc: "small instance boundary (2GB)",
			mem:  2 * parse.Gigabyte,
			want: minMaxConns,
		},
		{
			desc: "medium instance (3GB)",
			mem:  3 * parse.Gigabyte,
			want: 50,
		},
		{
			desc: "medium instance boundary (4GB)",
			mem:  4 * parse.Gigabyte,
			want: 50,
		},
		{
			desc: "big instance",
			mem:  5 * parse.Gigabyte,
			want: 75,
		},
		{
			desc: "big instance boundary (6GB)",
			mem:  6 * parse.Gigabyte,
			want: 75,
		},
		{
			desc: "large instance",
			mem:  7 * parse.Gigabyte,
			want: MaxConnectionsDefault,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := getMaxConns(c.mem); got != c.want {
				t.Errorf("incorrect conns: got %d want %d", got, c.want)
			}
		})
	}
}

func TestGetEffectiveIOConcurrency(t *testing.T) {
	cases := []struct {
		pgMajorVersion string
		want           string
	}{
		{
			pgutils.MajorVersion96,
			effectiveIODefaultOldVersions,
		},
		{
			pgutils.MajorVersion10,
			effectiveIODefaultOldVersions,
		},
		{
			pgutils.MajorVersion11,
			effectiveIODefaultOldVersions,
		},
		{
			pgutils.MajorVersion12,
			effectiveIODefaultOldVersions,
		},
		{
			pgutils.MajorVersion13,
			effectiveIODefault,
		},
		{
			/* a new version, not yet released */
			"15",
			effectiveIODefault,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("test effective_io_concurrency (v%s)", c.pgMajorVersion), func(t *testing.T) {
			if got := getEffectiveIOConcurrency(c.pgMajorVersion); got != c.want {
				t.Errorf("incorrect effective_io_concurrency: got %s, want %s", got, c.want)
			}
		})
	}
}

func TestDefaultToastCompression(t *testing.T) {
	cases := []struct {
		pgMajorVersions []string
		want            string
	}{
		{
			[]string{pgutils.MajorVersion96, pgutils.MajorVersion10, pgutils.MajorVersion11, pgutils.MajorVersion12, pgutils.MajorVersion13},
			NoRecommendation,
		},
		{
			[]string{pgutils.MajorVersion14, pgutils.MajorVersion15, pgutils.MajorVersion16, pgutils.MajorVersion17,
				"18", //future versions
			},
			"lz4",
		},
	}
	for _, c := range cases {
		for _, v := range c.pgMajorVersions {
			t.Run("default_toast_compression:"+v, func(t *testing.T) {
				r := NewMiscRecommender(1000, 32, v)

				rec := r.Recommend(DefaultToastCompression)
				if rec != c.want {
					t.Errorf("wanted %s got: %s", c.want, rec)
				}

			})
		}
	}
}

func TestJIT(t *testing.T) {
	cases := []struct {
		pgMajorVersions []string
		want            string
	}{
		{
			[]string{pgutils.MajorVersion96, pgutils.MajorVersion10, pgutils.MajorVersion11},
			NoRecommendation,
		},
		{
			[]string{pgutils.MajorVersion12, pgutils.MajorVersion13, pgutils.MajorVersion14, pgutils.MajorVersion15, pgutils.MajorVersion16, pgutils.MajorVersion17,
				"18", //future versions
			},
			"off",
		},
	}
	for _, c := range cases {
		for _, v := range c.pgMajorVersions {
			t.Run("default_toast_compression:"+v, func(t *testing.T) {
				r := NewMiscRecommender(1000, 32, v)

				rec := r.Recommend(Jit)
				if rec != c.want {
					t.Errorf("wanted %s got: %s", c.want, rec)
				}

			})
		}
	}
}

func TestNewMiscRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		conns := rand.Uint64()
		r := NewMiscRecommender(mem, conns, pgutils.MajorVersion12)
		if r == nil {
			t.Errorf("unexpected nil recommender")
			continue
		}

		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect memory: got %d want %d", got, mem)
		}
		if got := r.maxConns; got != conns {
			t.Errorf("recommender has incorrect conns: got %d want %d", got, conns)
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestMiscRecommenderRecommend(t *testing.T) {
	for totalMemory, outerMatrix := range miscSettingsMatrix {
		for maxConns, matrix := range outerMatrix {
			r := &MiscRecommender{totalMemory, maxConns, pgutils.MajorVersion10}
			testRecommender(t, r, MiscKeys, matrix)
		}
	}
}

func TestMiscRecommenderNoRecommendation(t *testing.T) {
	r := &MiscRecommender{}
	if r.Recommend("foo") != NoRecommendation {
		t.Error("Recommendation was provided when there should have been none")
	}
}

func TestMiscSettingsGroup(t *testing.T) {
	for totalMemory, outerMatrix := range miscSettingsMatrix {
		for maxConns, matrix := range outerMatrix {
			config, err := NewSystemConfig(totalMemory, 8, "10", walDiskUnset, maxConns, MaxBackgroundWorkersDefault)
			if err != nil {
				t.Errorf("unexpected error on system config creation: got %v", err)
			}
			sg := GetSettingsGroup(MiscLabel, config)

			testSettingGroup(t, sg, DefaultProfile, matrix, MiscLabel, MiscKeys)
		}
	}
}
