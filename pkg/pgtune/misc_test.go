package pgtune

import (
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

var miscSettingsMatrix = map[uint64]map[string]string{
	7 * parse.Gigabyte:  map[string]string{MaxLocksPerTx: maxLocksValues[0]},
	8 * parse.Gigabyte:  map[string]string{MaxLocksPerTx: maxLocksValues[1]},
	15 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[1]},
	16 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[2]},
	24 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[2]},
	32 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[3]},
	80 * parse.Gigabyte: map[string]string{MaxLocksPerTx: maxLocksValues[3]},
}

func init() {
	for level := range miscSettingsMatrix {
		miscSettingsMatrix[level][CheckpointKey] = checkpointDefault
		miscSettingsMatrix[level][StatsTargetKey] = statsTargetDefault
		miscSettingsMatrix[level][MaxConnectionsKey] = maxConnectionsDefault
		miscSettingsMatrix[level][RandomPageCostKey] = randomPageCostDefault
		miscSettingsMatrix[level][EffectiveIOKey] = effectiveIODefault
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
	for totalMemory, matrix := range miscSettingsMatrix {
		r := &MiscRecommender{totalMemory}
		testRecommender(t, r, matrix)
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
	for totalMemory, matrix := range miscSettingsMatrix {
		config := NewSystemConfig(totalMemory, 8, "10")
		sg := GetSettingsGroup(MiscLabel, config)
		testSettingGroup(t, sg, matrix, MiscLabel, MiscKeys)
	}
}
