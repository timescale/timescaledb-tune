package pgtune

import (
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// memoryToWALBuffers provides a mapping from test case memory levels to the
// expected WAL buffers setting. This is used to generate the test cases for
// WALRecommender, stored in walSettingsMatrix.
var memoryToWALBuffers = map[uint64]uint64{
	1 * parse.Gigabyte:                    7864 * parse.Kilobyte,
	uint64(1.5 * float64(parse.Gigabyte)): 11796 * parse.Kilobyte,
	2 * parse.Gigabyte:                    walBuffersDefault,
	10 * parse.Gigabyte:                   walBuffersDefault,
}

// walSettingsMatrix stores the test cases for WALRecommender along with the
// expected values for WAL keys.
var walSettingsMatrix = map[uint64]map[string]string{}

func init() {
	for memory, walBuffers := range memoryToWALBuffers {
		walSettingsMatrix[memory] = make(map[string]string)
		walSettingsMatrix[memory][MinWALKey] = parse.BytesToPGFormat(minWALBytes)
		walSettingsMatrix[memory][MaxWALKey] = parse.BytesToPGFormat(maxWALBytes)
		walSettingsMatrix[memory][WALBuffersKey] = parse.BytesToPGFormat(walBuffers)
	}
}

func TestNewWALRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		r := NewWALRecommender(mem)
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

func TestWALRecommenderRecommend(t *testing.T) {
	for totalMemory, matrix := range walSettingsMatrix {
		r := &WALRecommender{totalMemory}
		testRecommender(t, r, matrix)
	}
}

func TestWALRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &WALRecommender{0}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestWALSettingsGroup(t *testing.T) {
	for totalMemory, matrix := range walSettingsMatrix {
		config := NewSystemConfig(totalMemory, 4, "10")
		sg := GetSettingsGroup(WALLabel, config)
		testSettingGroup(t, sg, matrix, WALLabel, WALKeys)
	}
}
