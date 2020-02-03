package pgtune

import (
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

const (
	walDiskUnset          = 0
	walDiskDivideUnevenly = 8 * parse.Gigabyte
	walDiskDivideEvenly   = 8800 * parse.Megabyte
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

var walDiskToMaxBytes = map[uint64]uint64{
	walDiskUnset:          defaultMaxWALBytes,
	walDiskDivideUnevenly: 4928 * parse.Megabyte, // nearest 16MB segment
	walDiskDivideEvenly:   5280 * parse.Megabyte,
}

// walSettingsMatrix stores the test cases for WALRecommender along with the
// expected values for WAL keys.
var walSettingsMatrix = map[uint64]map[uint64]map[string]string{}

func init() {
	for memory, walBuffers := range memoryToWALBuffers {
		walSettingsMatrix[memory] = make(map[uint64]map[string]string)
		for walSize := range walDiskToMaxBytes {
			walSettingsMatrix[memory][walSize] = make(map[string]string)
			walSettingsMatrix[memory][walSize][MinWALKey] = parse.BytesToPGFormat(walDiskToMaxBytes[walSize] / 2)
			walSettingsMatrix[memory][walSize][MaxWALKey] = parse.BytesToPGFormat(walDiskToMaxBytes[walSize])
			walSettingsMatrix[memory][walSize][WALBuffersKey] = parse.BytesToPGFormat(walBuffers)
		}
	}
}

func TestNewWALRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		r := NewWALRecommender(mem, walDiskUnset)
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
	for totalMemory, outerMatrix := range walSettingsMatrix {
		for walSize, matrix := range outerMatrix {
			r := NewWALRecommender(totalMemory, walSize)
			testRecommender(t, r, WALKeys, matrix)
		}
	}
}

func TestWALRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := NewWALRecommender(0, 0)
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestWALSettingsGroup(t *testing.T) {
	for totalMemory, outerMatrix := range walSettingsMatrix {
		for walSize, matrix := range outerMatrix {
			config := getDefaultTestSystemConfig(t)
			config.Memory = totalMemory
			config.WALDiskSize = walSize
			sg := GetSettingsGroup(WALLabel, config)
			testSettingGroup(t, sg, matrix, WALLabel, WALKeys)
		}
	}
}
