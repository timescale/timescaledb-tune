package pgtune

import (
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// pgTextsearchDefaultMatrix maps total memory to the expected
// pg_textsearch.memory_limit value under the default profile (totalMemory / 4).
var pgTextsearchDefaultMatrix = map[uint64]string{
	2 * parse.Gigabyte:  parse.BytesToPGFormat(512 * parse.Megabyte),
	4 * parse.Gigabyte:  parse.BytesToPGFormat(1 * parse.Gigabyte),
	8 * parse.Gigabyte:  parse.BytesToPGFormat(2 * parse.Gigabyte),
	16 * parse.Gigabyte: parse.BytesToPGFormat(4 * parse.Gigabyte),
	32 * parse.Gigabyte: parse.BytesToPGFormat(8 * parse.Gigabyte),
	64 * parse.Gigabyte: parse.BytesToPGFormat(16 * parse.Gigabyte),
}

// pgTextsearchPromscaleMatrix maps total memory to the expected
// pg_textsearch.memory_limit value under the promscale profile (totalMemory / 8),
// reflecting that shared_buffers takes mem/2 in that profile.
var pgTextsearchPromscaleMatrix = map[uint64]string{
	2 * parse.Gigabyte:  parse.BytesToPGFormat(256 * parse.Megabyte),
	4 * parse.Gigabyte:  parse.BytesToPGFormat(512 * parse.Megabyte),
	8 * parse.Gigabyte:  parse.BytesToPGFormat(1 * parse.Gigabyte),
	16 * parse.Gigabyte: parse.BytesToPGFormat(2 * parse.Gigabyte),
	32 * parse.Gigabyte: parse.BytesToPGFormat(4 * parse.Gigabyte),
	64 * parse.Gigabyte: parse.BytesToPGFormat(8 * parse.Gigabyte),
}

func TestNewPgTextsearchRecommender(t *testing.T) {
	for i := 0; i < 1000; i++ {
		mem := rand.Uint64()
		r := NewPgTextsearchRecommender(mem, DefaultProfile, true)
		if r == nil {
			t.Fatalf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect memory: got %d want %d", got, mem)
		}
	}
}

func TestPgTextsearchRecommenderIsAvailable(t *testing.T) {
	cases := []struct {
		desc    string
		enabled bool
		want    bool
	}{
		{"enabled (pg_textsearch in shared_preload_libraries)", true, true},
		{"disabled (pg_textsearch absent)", false, false},
	}
	for _, c := range cases {
		r := NewPgTextsearchRecommender(8*parse.Gigabyte, DefaultProfile, c.enabled)
		if got := r.IsAvailable(); got != c.want {
			t.Errorf("%s: IsAvailable() = %v, want %v", c.desc, got, c.want)
		}
	}
}

func TestPgTextsearchRecommenderRecommendDefault(t *testing.T) {
	for mem, want := range pgTextsearchDefaultMatrix {
		r := NewPgTextsearchRecommender(mem, DefaultProfile, true)
		if got := r.Recommend(MemoryLimitKey); got != want {
			t.Errorf("memory=%d: incorrect %s: got %s want %s", mem, MemoryLimitKey, got, want)
		}
	}
}

func TestPgTextsearchRecommenderRecommendPromscale(t *testing.T) {
	for mem, want := range pgTextsearchPromscaleMatrix {
		r := NewPgTextsearchRecommender(mem, PromscaleProfile, true)
		if got := r.Recommend(MemoryLimitKey); got != want {
			t.Errorf("memory=%d: incorrect %s: got %s want %s", mem, MemoryLimitKey, got, want)
		}
	}
}

func TestPgTextsearchRecommenderNoRecommendation(t *testing.T) {
	r := NewPgTextsearchRecommender(8*parse.Gigabyte, DefaultProfile, true)
	if got := r.Recommend("foo"); got != NoRecommendation {
		t.Errorf("unexpected recommendation for unknown key: got %q", got)
	}
}

func TestPgTextsearchSettingsGroup(t *testing.T) {
	for mem, want := range pgTextsearchDefaultMatrix {
		config := getDefaultTestSystemConfig(t)
		config.Memory = mem
		config.PgTextsearchEnabled = true

		sg := GetSettingsGroup(PgTextsearchLabel, config)
		matrix := map[string]string{MemoryLimitKey: want}
		testSettingGroup(t, sg, DefaultProfile, matrix, PgTextsearchLabel, PgTextsearchKeys)
	}
}

func TestPgTextsearchSettingsGroupPromscale(t *testing.T) {
	for mem, want := range pgTextsearchPromscaleMatrix {
		config := getDefaultTestSystemConfig(t)
		config.Memory = mem
		config.PgTextsearchEnabled = true

		sg := GetSettingsGroup(PgTextsearchLabel, config)
		matrix := map[string]string{MemoryLimitKey: want}
		testSettingGroup(t, sg, PromscaleProfile, matrix, PgTextsearchLabel, PgTextsearchKeys)
	}
}

func TestPgTextsearchSettingsGroupGatedByEnabled(t *testing.T) {
	config := getDefaultTestSystemConfig(t)
	config.Memory = 8 * parse.Gigabyte

	config.PgTextsearchEnabled = false
	sg := GetSettingsGroup(PgTextsearchLabel, config)
	if sg.GetRecommender(DefaultProfile).IsAvailable() {
		t.Errorf("recommender should be unavailable when pg_textsearch is not in shared_preload_libraries")
	}

	config.PgTextsearchEnabled = true
	sg = GetSettingsGroup(PgTextsearchLabel, config)
	if !sg.GetRecommender(DefaultProfile).IsAvailable() {
		t.Errorf("recommender should be available when pg_textsearch is in shared_preload_libraries")
	}
}
