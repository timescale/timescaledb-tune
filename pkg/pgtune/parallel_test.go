package pgtune

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/timescale/timescaledb-tune/pkg/pgutils"
)

// parallelSettingsMatrix stores the test cases for ParallelRecommender along
// with the expected values for its keys
var parallelSettingsMatrix = map[int]map[int]map[string]string{
	2: {
		MaxBackgroundWorkersDefault: {
			MaxBackgroundWorkers:        fmt.Sprintf("%d", MaxBackgroundWorkersDefault),
			MaxWorkerProcessesKey:       fmt.Sprintf("%d", 2+minBuiltInProcesses+MaxBackgroundWorkersDefault),
			MaxParallelWorkersGatherKey: "1",
			MaxParallelWorkers:          "2",
		},
		MaxBackgroundWorkersDefault * 2: {
			MaxBackgroundWorkers:        fmt.Sprintf("%d", MaxBackgroundWorkersDefault*2),
			MaxWorkerProcessesKey:       fmt.Sprintf("%d", 2+minBuiltInProcesses+MaxBackgroundWorkersDefault*2),
			MaxParallelWorkersGatherKey: "1",
			MaxParallelWorkers:          "2",
		},
	},
	4: {
		MaxBackgroundWorkersDefault: {
			MaxBackgroundWorkers:        fmt.Sprintf("%d", MaxBackgroundWorkersDefault),
			MaxWorkerProcessesKey:       fmt.Sprintf("%d", 4+minBuiltInProcesses+MaxBackgroundWorkersDefault),
			MaxParallelWorkersGatherKey: "2",
			MaxParallelWorkers:          "4",
		},
		MaxBackgroundWorkersDefault * 4: {
			MaxBackgroundWorkers:        fmt.Sprintf("%d", MaxBackgroundWorkersDefault*4),
			MaxWorkerProcessesKey:       fmt.Sprintf("%d", 4+minBuiltInProcesses+MaxBackgroundWorkersDefault*4),
			MaxParallelWorkersGatherKey: "2",
			MaxParallelWorkers:          "4",
		},
	},
	5: {
		MaxBackgroundWorkersDefault: {
			MaxBackgroundWorkers:        fmt.Sprintf("%d", MaxBackgroundWorkersDefault),
			MaxWorkerProcessesKey:       fmt.Sprintf("%d", 5+minBuiltInProcesses+MaxBackgroundWorkersDefault),
			MaxParallelWorkersGatherKey: "3",
			MaxParallelWorkers:          "5",
		},
		MaxBackgroundWorkersDefault * 5: {
			MaxBackgroundWorkers:        fmt.Sprintf("%d", MaxBackgroundWorkersDefault*5),
			MaxWorkerProcessesKey:       fmt.Sprintf("%d", 5+minBuiltInProcesses+MaxBackgroundWorkersDefault*5),
			MaxParallelWorkersGatherKey: "3",
			MaxParallelWorkers:          "5",
		},
	},
}

func TestNewParallelRecommender(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1000000; i++ {
		cpus := rand.Intn(128)
		// ensure a minimum of background workers
		workers := rand.Intn(128-MaxBackgroundWorkersDefault+1) + MaxBackgroundWorkersDefault
		r := NewParallelRecommender(cpus, workers)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.cpus; got != cpus {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}
		if got := r.maxBGWorkers; got != workers {
			t.Errorf("recommender has incorrect workers: got %d want %d", got, workers)
		}
	}
}

func TestParallelRecommenderIsAvailable(t *testing.T) {
	if r := NewParallelRecommender(0, MaxBackgroundWorkersDefault); r.IsAvailable() {
		t.Errorf("unexpectedly available for 0 cpus")
	}
	if r := NewParallelRecommender(1, MaxBackgroundWorkersDefault); r.IsAvailable() {
		t.Errorf("unexpectedly available for 1 cpus")
	}

	for i := 2; i < 1000; i++ {
		if r := NewParallelRecommender(i, MaxBackgroundWorkersDefault); !r.IsAvailable() {
			t.Errorf("unexpected UNavailable for %d cpus", i)
		}
	}
}

func TestParallelRecommenderRecommend(t *testing.T) {
	for cpus, tempMatrix := range parallelSettingsMatrix {
		for workers, matrix := range tempMatrix {
			r := &ParallelRecommender{cpus, workers}
			testRecommender(t, r, ParallelKeys, matrix)
		}
	}
}

func TestParallelRecommenderRecommendPanics(t *testing.T) {
	// test invalid key panic
	func() {
		r := &ParallelRecommender{5, MaxBackgroundWorkersDefault}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()

	// test invalid CPU panic
	func() {
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r := &ParallelRecommender{1, MaxBackgroundWorkersDefault}
		r.Recommend("foo")
	}()

	// test invalid worker panic
	func() {
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r := &ParallelRecommender{5, MaxBackgroundWorkersDefault - 1}
		r.Recommend("foo")
	}()
}

func TestParallelSettingsGroup(t *testing.T) {
	keyCount := len(ParallelKeys)
	for cpus, tempMatrix := range parallelSettingsMatrix {
		for workers, matrix := range tempMatrix {
			config := getDefaultTestSystemConfig(t)
			config.CPUs = cpus
			config.PGMajorVersion = pgutils.MajorVersion96 // 9.6 lacks one key
			config.MaxBGWorkers = workers
			sg := GetSettingsGroup(ParallelLabel, config)
			if got := len(sg.Keys()); got != keyCount-1 {
				t.Errorf("incorrect number of keys for PG %s: got %d want %d", pgutils.MajorVersion96, got, keyCount-1)
			}
			testSettingGroup(t, sg, matrix, ParallelLabel, ParallelKeys)

			// PG10 adds a key
			config.PGMajorVersion = pgutils.MajorVersion10
			sg = GetSettingsGroup(ParallelLabel, config)
			if got := len(sg.Keys()); got != keyCount {
				t.Errorf("incorrect number of keys for PG %s: got %d want %d", pgutils.MajorVersion10, got, keyCount)
			}
			testSettingGroup(t, sg, matrix, ParallelLabel, ParallelKeys)

			config.PGMajorVersion = pgutils.MajorVersion11
			sg = GetSettingsGroup(ParallelLabel, config)
			if got := len(sg.Keys()); got != keyCount {
				t.Errorf("incorrect number of keys for PG %s: got %d want %d", pgutils.MajorVersion11, got, keyCount)
			}
			testSettingGroup(t, sg, matrix, ParallelLabel, ParallelKeys)
		}
	}

}
