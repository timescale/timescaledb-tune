package pgtune

import (
	"math/rand"
	"testing"
)

func TestNewSystemConfig(t *testing.T) {
	for i := 0; i < 1000; i++ {
		mem := rand.Uint64()
		cpus := rand.Intn(32)
		pgVersion := "10"
		if i%2 == 0 {
			pgVersion = "9.6"
		}

		config := NewSystemConfig(mem, cpus, pgVersion)
		if config.Memory != mem {
			t.Errorf("incorrect memory: got %d want %d", config.Memory, mem)
		}
		if config.CPUs != cpus {
			t.Errorf("incorrect cpus: got %d want %d", config.CPUs, cpus)
		}
		if config.PGMajorVersion != pgVersion {
			t.Errorf("incorrect pg version: got %s want %s", config.PGMajorVersion, pgVersion)
		}
	}
}

func TestGetSettingsGroup(t *testing.T) {
	okLabels := []string{MemoryLabel, ParallelLabel, WALLabel, MiscLabel}
	config := NewSystemConfig(1024, 4, "10")
	for _, label := range okLabels {
		sg := GetSettingsGroup(label, config)
		if sg == nil {
			t.Errorf("settings group unexpectedly nil for label %s", label)
		}
		switch x := sg.(type) {
		case *MemorySettingsGroup:
			if x.totalMemory != config.Memory || x.cpus != config.CPUs {
				t.Errorf("memory settings group incorrect: got %d,%d want %d,%d", x.totalMemory, x.cpus, config.Memory, config.CPUs)
			}
		case *ParallelSettingsGroup:
			if x.cpus != config.CPUs {
				t.Errorf("parallel settings group incorrect: got %d want %d", x.cpus, config.CPUs)
			}
			if x.pgVersion != config.PGMajorVersion {
				t.Errorf("parallel settings group incorrect: got %s want %s", x.pgVersion, config.PGMajorVersion)
			}
		case *WALSettingsGroup:
			if x.totalMemory != config.Memory {
				t.Errorf("WAL settings group incorrect: got %d want %d", x.totalMemory, config.Memory)
			}
		case *MiscSettingsGroup:
		default:
			t.Errorf("unexpected type for settings group %T", x)
		}
	}

	// this should panic on unknown label
	func() {
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		GetSettingsGroup("foo", config)
	}()
}

func testSettingGroup(t *testing.T, sg SettingsGroup, cases map[string]string, wantLabel string, wantKeys []string) {
	// No matter how many calls, all calls should return the same
	for i := 0; i < 1000; i++ {
		if got := sg.Label(); got != wantLabel {
			t.Errorf("incorrect label: got %s want %s", got, wantLabel)
		}
		if got := sg.Keys(); got != nil {
			for i, k := range got {
				if k != wantKeys[i] {
					t.Errorf("incorrect key at %d: got %s want %s", i, k, wantKeys[i])
				}
			}
		} else {
			t.Errorf("keys is nil")
		}
		r := sg.GetRecommender()

		testRecommender(t, r, cases)
	}
}

func testRecommender(t *testing.T, r Recommender, cases map[string]string) {
	for key, want := range cases {
		if got := r.Recommend(key); got != want {
			t.Errorf("incorrect result for key %s: got\n%s\nwant\n%s", key, got, want)
		}
	}
}
