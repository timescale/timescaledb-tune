package pgtune

import (
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

var memSettingsMatrix = map[uint64]map[string]string{
	7 * parse.Gigabyte:  map[string]string{MaxLocksPerTx: maxLocksValues[0]},
	8 * parse.Gigabyte:  map[string]string{MaxLocksPerTx: maxLocksValues[1]},
	15 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[1]},
	16 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[2]},
	24 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[2]},
	32 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[3]},
	80 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[3]},
}

func init() {
	for level := range memSettingsMatrix {
		memSettingsMatrix[level][CheckpointKey] = checkpointDefault
		memSettingsMatrix[level][StatsTargetKey] = statsTargetDefault
		memSettingsMatrix[level][MaxConnectionsKey] = maxConnectionsDefault
		memSettingsMatrix[level][RandomPageCostKey] = randomPageCostDefault
		memSettingsMatrix[level][EffectiveIOKey] = effectiveIODefault
	}
}

func TestNewMiscRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		r := NewMiscRecommender(mem)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect memory: got %d want %d", got, mem)
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestMiscRecommenderRecommend(t *testing.T) {
	for totalMemory, kvs := range memSettingsMatrix {
		r := &MiscRecommender{totalMemory}
		for key, want := range kvs {
			if got := r.Recommend(key); got != want {
				t.Errorf("%d-%s: incorrect result: got\n%s\nwant\n%s", totalMemory, key, got, want)
			}
		}
	}
}

func TestMiscRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &MiscRecommender{}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestMiscSettingsGroup(t *testing.T) {
	for totalMemory, kvs := range memSettingsMatrix {
		config := NewSystemConfig(totalMemory, 8, "10")
		sg := GetSettingsGroup(MiscLabel, config)
		// no matter how many calls, all calls should return the same
		for i := 0; i < 1000; i++ {
			if got := sg.Label(); got != MiscLabel {
				t.Errorf("incorrect label: got %s want %s", got, MiscLabel)
			}
			if got := sg.Keys(); got != nil {
				for i, k := range got {
					if k != MiscKeys[i] {
						t.Errorf("incorrect key at %d: got %s want %s", i, k, MiscKeys[i])
					}
				}
			} else {
				t.Errorf("keys is nil")
			}
			r := sg.GetRecommender().(*MiscRecommender)
			// the above will panic if not true

			for key, want := range kvs {
				if got := r.Recommend(key); got != want {
					t.Errorf("%d-%s: incorrect result: got\n%s\nwant\n%s", totalMemory, key, got, want)
				}
			}
		}
	}
}
