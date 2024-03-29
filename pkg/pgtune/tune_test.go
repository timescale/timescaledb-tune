package pgtune

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

const (
	testMaxConnsSpecial = 0
	testMaxConnsBad     = 1
	testMaxConns        = minMaxConns
)

func getDefaultTestSystemConfig(t *testing.T) *SystemConfig {
	config, err := NewSystemConfig(1024, 4, "10", walDiskUnset, testMaxConns, MaxBackgroundWorkersDefault)
	if err != nil {
		t.Errorf("unexpected error: got %v", err)
	}
	return config
}

func TestNewSystemConfig(t *testing.T) {
	for i := 0; i < 1000; i++ {
		mem := rand.Uint64()
		cpus := rand.Intn(32)
		pgVersion := "10"
		if i%2 == 0 {
			pgVersion = "9.6"
		}

		config, err := NewSystemConfig(mem, cpus, pgVersion, walDiskUnset, testMaxConns, MaxBackgroundWorkersDefault)
		if err != nil {
			t.Errorf("unexpected error: got %v", err)
		}
		if config.Memory != mem {
			t.Errorf("incorrect memory: got %d want %d", config.Memory, mem)
		}
		if config.CPUs != cpus {
			t.Errorf("incorrect cpus: got %d want %d", config.CPUs, cpus)
		}
		if config.PGMajorVersion != pgVersion {
			t.Errorf("incorrect pg version: got %s want %s", config.PGMajorVersion, pgVersion)
		}
		if config.maxConns != testMaxConns {
			t.Errorf("incorrect max conns: got %d want %d", config.maxConns, testMaxConns)
		}
		if config.MaxBGWorkers != MaxBackgroundWorkersDefault {
			t.Errorf("incorrect max background workers: got %d want %d", config.MaxBGWorkers, MaxBackgroundWorkersDefault)
		}

		// test invalid number of connections
		_, err = NewSystemConfig(mem, cpus, pgVersion, walDiskUnset, testMaxConnsBad, MaxBackgroundWorkersDefault)
		wantErr := fmt.Sprintf(errMaxConnsTooLowFmt, minMaxConns, testMaxConnsBad)
		if err == nil {
			t.Errorf("unexpected lack of error")
		} else if got := err.Error(); got != wantErr {
			t.Errorf("unexpected error: got\n%s\nwant\n%s", got, wantErr)
		}

		// test 0 connections
		config, err = NewSystemConfig(mem, cpus, pgVersion, walDiskUnset, testMaxConnsSpecial, MaxBackgroundWorkersDefault)
		if err != nil {
			t.Errorf("unexpected error: got %v", err)
		}
		if config.maxConns != testMaxConnsSpecial {
			t.Errorf("incorrect max conns: got %d want %d", config.maxConns, testMaxConnsSpecial)
		}

		// test invalid number of background workers
		_, err = NewSystemConfig(mem, cpus, pgVersion, walDiskUnset, testMaxConns, MaxBackgroundWorkersDefault-1)
		wantErr = fmt.Sprintf(errMaxBGWorkersTooLowFmt, MaxBackgroundWorkersDefault, MaxBackgroundWorkersDefault-1)
		if err == nil {
			t.Errorf("unexpected lack of error")
		} else if got := err.Error(); got != wantErr {
			t.Errorf("unexpected error: got\n%s\nwant\n%s", got, wantErr)
		}

	}
}

func TestGetSettingsGroup(t *testing.T) {
	okLabels := []string{MemoryLabel, ParallelLabel, WALLabel, BgwriterLabel, MiscLabel}
	config := getDefaultTestSystemConfig(t)
	for _, label := range okLabels {
		sg := GetSettingsGroup(label, config)
		if sg == nil {
			t.Errorf("settings group unexpectedly nil for label %s", label)
		}
		switch x := sg.(type) {
		case *MemorySettingsGroup:
			if x.totalMemory != config.Memory || x.cpus != config.CPUs {
				t.Errorf("memory group incorrect (memory): got %d want %d", x.totalMemory, config.Memory)
			}
			if x.cpus != config.CPUs {
				t.Errorf("memory group incorrect (CPUs): got %d want %d", x.cpus, config.CPUs)
			}
		case *ParallelSettingsGroup:
			if x.cpus != config.CPUs {
				t.Errorf("parallel group incorrect (CPUs): got %d want %d", x.cpus, config.CPUs)
			}
			if x.pgVersion != config.PGMajorVersion {
				t.Errorf("parallel group incorrect (PG version): got %s want %s", x.pgVersion, config.PGMajorVersion)
			}
		case *WALSettingsGroup:
			if x.totalMemory != config.Memory {
				t.Errorf("WAL group incorrect (memory): got %d want %d", x.totalMemory, config.Memory)
			}
			if x.walDiskSize != config.WALDiskSize {
				t.Errorf("WAL group incorrect (wal disk): got %d want %d", x.walDiskSize, config.WALDiskSize)
			}
		case *BgwriterSettingsGroup:
			// nothing to check here
			continue
		case *MiscSettingsGroup:
			if x.totalMemory != config.Memory {
				t.Errorf("Misc group incorrect (memory): got %d want %d", x.totalMemory, config.Memory)
			}
			if x.maxConns != config.maxConns {
				t.Errorf("Misc group incorrect (max conns): got %d want %d", x.maxConns, config.maxConns)
			}
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

func testSettingGroup(t *testing.T, sg SettingsGroup, profile Profile, cases map[string]string, wantLabel string, wantKeys []string) {
	t.Helper()

	// No matter how many calls, all calls should return the same
	for i := 0; i < 1000; i++ {
		if got := sg.Label(); got != wantLabel {
			t.Errorf("incorrect label: got %s want %s", got, wantLabel)
		}
		if got := sg.Keys(); got == nil {
			t.Errorf("keys is nil")
		}
		for i, k := range sg.Keys() {
			if k != wantKeys[i] {
				t.Errorf("incorrect key at %d: got %s want %s", i, k, wantKeys[i])
			}
		}

		r := sg.GetRecommender(profile)

		testRecommender(t, r, sg.Keys(), cases)
	}
}

// testRecommender is a helper method for testing whether a Recommender gives
// the appropriate values for a set of keys.
//
// Rather than iterating over the 'wants' map to get the keys, we iterate over
// a separate 'keys' parameter that should include _all_ keys a Recommender
// handles. This makes sure that when new keys are added, our tests are comprehensive,
// since otherwise the Recommender will panic on an unknown key.
func testRecommender(t *testing.T, r Recommender, keys []string, wants map[string]string) {
	t.Helper()

	for _, key := range keys {
		want := wants[key]
		if got := r.Recommend(key); got != want {
			t.Errorf("%T: incorrect result for key %s: got\n%s\nwant\n%s", r, key, got, want)
		}
	}
}

func TestParseProfile(t *testing.T) {
	cases := []struct {
		input    string
		expected Profile
	}{
		{input: DefaultProfile.String(), expected: DefaultProfile},
		{input: PromscaleProfile.String(), expected: PromscaleProfile},
		{input: strings.ToUpper(PromscaleProfile.String()), expected: PromscaleProfile},
	}
	for _, kase := range cases {
		actual, err := ParseProfile(kase.input)
		if err != nil {
			t.Errorf("expected %v for input %s but got an error: %v", kase.expected, kase.input, err)
		}
		if actual != kase.expected {
			t.Errorf("expected %v for input %s but got %v", kase.expected, kase.input, actual)
		}
	}

	if actual, err := ParseProfile("garbage"); err == nil {
		t.Errorf("expected to get an error for unrecognized input, but did not. got %v", actual)
	}
}
