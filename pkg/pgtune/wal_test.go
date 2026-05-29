package pgtune

import (
	"fmt"
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

var promscaleWALDiskToMaxBytes = map[uint64]uint64{
	walDiskUnset:          promscaleDefaultMaxWALBytes,
	walDiskDivideUnevenly: 4928 * parse.Megabyte, // nearest 16MB segment
	walDiskDivideEvenly:   5280 * parse.Megabyte,
}

// walSettingsMatrix stores the test cases for WALRecommender along with the
// expected values for WAL keys.
var walSettingsMatrix = map[uint64]map[uint64]map[string]string{}

// walSettingsMatrix stores the test cases for WALRecommender along with the
// expected values for WAL keys.
var promscaleWalSettingsMatrix = map[uint64]map[uint64]map[string]string{}

func init() {
	for memory, walBuffers := range memoryToWALBuffers {
		walSettingsMatrix[memory] = make(map[uint64]map[string]string)
		for walSize := range walDiskToMaxBytes {
			walSettingsMatrix[memory][walSize] = make(map[string]string)
			walSettingsMatrix[memory][walSize][MinWALKey] = parse.BytesToPGFormat(walDiskToMaxBytes[walSize] / 2)
			walSettingsMatrix[memory][walSize][MaxWALKey] = parse.BytesToPGFormat(walDiskToMaxBytes[walSize])
			walSettingsMatrix[memory][walSize][WALBuffersKey] = parse.BytesToPGFormat(walBuffers)
			walSettingsMatrix[memory][walSize][CheckpointTimeoutKey] = NoRecommendation
			walSettingsMatrix[memory][walSize][WALCompressionKey] = NoRecommendation
		}
	}

	for memory, walBuffers := range memoryToWALBuffers {
		promscaleWalSettingsMatrix[memory] = make(map[uint64]map[string]string)
		for walSize := range walDiskToMaxBytes {
			promscaleWalSettingsMatrix[memory][walSize] = make(map[string]string)
			promscaleWalSettingsMatrix[memory][walSize][MinWALKey] = parse.BytesToPGFormat(promscaleWALDiskToMaxBytes[walSize] / 2)
			promscaleWalSettingsMatrix[memory][walSize][MaxWALKey] = parse.BytesToPGFormat(promscaleWALDiskToMaxBytes[walSize])
			promscaleWalSettingsMatrix[memory][walSize][WALBuffersKey] = parse.BytesToPGFormat(walBuffers)
			promscaleWalSettingsMatrix[memory][walSize][CheckpointTimeoutKey] = promscaleDefaultCheckpointTimeout
			promscaleWalSettingsMatrix[memory][walSize][WALCompressionKey] = promscaleDefaultWALCompression
		}
	}
}

func TestWALSettingsGroup_GetRecommender(t *testing.T) {
	cases := []struct {
		profile     Profile
		recommender string
	}{
		{DefaultProfile, "*pgtune.WALRecommender"},
		{PromscaleProfile, "*pgtune.PromscaleWALRecommender"},
	}

	sg := WALSettingsGroup{totalMemory: 1, walDiskSize: 1}
	for _, k := range cases {
		r := sg.GetRecommender(k.profile)
		y := fmt.Sprintf("%T", r)
		if y != k.recommender {
			t.Errorf("Expected to get a %s using the %s profile but got %s", k.recommender, k.profile, y)
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

func TestPromscaleWALRecommenderRecommend(t *testing.T) {
	for totalMemory, outerMatrix := range promscaleWalSettingsMatrix {
		for walSize, matrix := range outerMatrix {
			r := NewPromscaleWALRecommender(totalMemory, walSize)
			testRecommender(t, r, WALKeys, matrix)
		}
	}
}

func TestPromscaleWALRecommenderCheckpointTimeout(t *testing.T) {
	// recommendation for checkpoint timeout should not be impacted by totalMemory or walDiskSize
	for i := uint64(0); i < 1000000; i++ {
		r := NewPromscaleWALRecommender(i, i)
		if v := r.Recommend(CheckpointTimeoutKey); v != promscaleDefaultCheckpointTimeout {
			t.Errorf("Expected %s for %s, but got %s", promscaleDefaultCheckpointTimeout, CheckpointTimeoutKey, v)
		}
	}
}

func TestWALRecommenderNoRecommendation(t *testing.T) {
	r := NewWALRecommender(0, 0)
	if r.Recommend("foo") != NoRecommendation {
		t.Errorf("Recommendation was provided for %s when there should have been none", "foo")
	}

	if r.Recommend(CheckpointTimeoutKey) != NoRecommendation {
		t.Errorf("Recommendation was provided for %s when there should have been none", CheckpointTimeoutKey)
	}
}

func TestWALSettingsGroup(t *testing.T) {
	for totalMemory, outerMatrix := range walSettingsMatrix {
		for walSize, matrix := range outerMatrix {
			config := getDefaultTestSystemConfig(t)
			config.Memory = totalMemory
			config.WALDiskSize = walSize
			sg := GetSettingsGroup(WALLabel, config)
			testSettingGroup(t, sg, DefaultProfile, matrix, WALLabel, WALKeys)
		}
	}

	for totalMemory, outerMatrix := range promscaleWalSettingsMatrix {
		for walSize, matrix := range outerMatrix {
			config := getDefaultTestSystemConfig(t)
			config.Memory = totalMemory
			config.WALDiskSize = walSize
			sg := GetSettingsGroup(WALLabel, config)
			testSettingGroup(t, sg, PromscaleProfile, matrix, WALLabel, WALKeys)
		}
	}
}

func TestWALFloatParserParseFloat(t *testing.T) {
	v := &WALFloatParser{}

	s := "8" + parse.GB
	want := float64(8 * parse.Gigabyte)
	got, err := v.ParseFloat(MaxWALKey, s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}

	s = "1000"
	want = 1000.0
	got, err = v.ParseFloat(WALBuffersKey, s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}

	s = "33" + parse.Minutes.String()
	conversion, _ := parse.TimeConversion(parse.Minutes, parse.Milliseconds)
	want = 33.0 * conversion
	got, err = v.ParseFloat(CheckpointTimeoutKey, s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}
}
