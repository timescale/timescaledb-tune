package pgtune

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

func TestNewMemoryRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		cpus := rand.Intn(128)
		r := NewMemoryRecommender(mem, cpus)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}
		if got := r.cpus; got != cpus {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestMemoryRecommenderRecommendWindows(t *testing.T) {
	cases := []struct {
		desc        string
		totalMemory uint64
		cpus        int
		want        string
	}{
		{
			desc:        "1GB",
			totalMemory: 1 * parse.Gigabyte,
			cpus:        1,
			want:        "6553" + parse.KB, // from pgtune
		},
		{
			desc:        "1GB, 4 cpus",
			totalMemory: 1 * parse.Gigabyte,
			cpus:        4,
			want:        "3276" + parse.KB, // from pgtune
		},
		{
			desc:        "2GB",
			totalMemory: 2 * parse.Gigabyte,
			cpus:        1,
			want:        "13107" + parse.KB, // from pgtune
		},
		{
			desc:        "2GB, 5 cpus",
			totalMemory: 2 * parse.Gigabyte,
			cpus:        5,
			want:        "4369" + parse.KB, // from pgtune
		},
		{
			desc:        "3GB",
			totalMemory: 3 * parse.Gigabyte,
			cpus:        1,
			want:        "21845" + parse.KB, // from pgtune
		},
		{
			desc:        "3GB, 3 cpus",
			totalMemory: 3 * parse.Gigabyte,
			cpus:        3,
			want:        "10922" + parse.KB, // from pgtune
		},
		{
			desc:        "8GB",
			totalMemory: 8 * parse.Gigabyte,
			cpus:        1,
			want:        "64" + parse.MB, // from pgtune
		},
		{
			desc:        "8GB, 8 cpus",
			totalMemory: 8 * parse.Gigabyte,
			cpus:        8,
			want:        "16" + parse.MB, // from pgtune
		},
		{
			desc:        "16GB",
			totalMemory: 16 * parse.Gigabyte,
			cpus:        1,
			want:        "135441" + parse.KB, // from pgtune
		},
		{
			desc:        "16GB, 10 cpus",
			totalMemory: 16 * parse.Gigabyte,
			cpus:        10,
			want:        "27088" + parse.KB, // from pgtune
		},
	}

	for _, c := range cases {
		mr := &MemoryRecommender{c.totalMemory, c.cpus}
		if got := mr.recommendWindows(); got != c.want {
			t.Errorf("%s: incorrect value: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestMemoryRecommenderRecommend(t *testing.T) {
	valFmt := "%d%s"
	cases := []struct {
		desc        string
		key         string
		totalMemory uint64
		cpus        int
		want        string
	}{
		{
			desc:        "shared_buffers, uneven divide",
			key:         SharedBuffersKey,
			totalMemory: 10 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, 2560, parse.MB),
		},
		{
			desc:        "shared_buffers, even divide",
			key:         SharedBuffersKey,
			totalMemory: 8 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, 2, parse.GB),
		},
		{
			desc:        "effective key, uneven divide",
			key:         EffectiveCacheKey,
			totalMemory: 10 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, uint64(7.5*1024.0), parse.MB),
		},
		{
			desc:        "effective key, even divide",
			key:         EffectiveCacheKey,
			totalMemory: 12 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, 9, parse.GB),
		},
		{
			desc:        "maintenance_work_mem",
			key:         MaintenanceWorkMemKey,
			totalMemory: 6 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, 768, parse.MB),
		},
		{
			desc:        "maintenance_work_mem, over max",
			key:         MaintenanceWorkMemKey,
			totalMemory: 32 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, 2, parse.GB),
		},
		{
			desc:        "work_mem",
			key:         WorkMemKey,
			totalMemory: 8 * parse.Gigabyte,
			cpus:        1,
			want:        fmt.Sprintf(valFmt, 52428, parse.KB),
		},
		{
			desc:        "work_mem, multiple CPUs",
			key:         WorkMemKey,
			totalMemory: 8 * parse.Gigabyte,
			cpus:        4,
			want:        fmt.Sprintf(valFmt, 26214, parse.KB),
		},
	}

	for _, c := range cases {
		mr := &MemoryRecommender{c.totalMemory, c.cpus}
		got := mr.Recommend(c.key)
		if got != c.want {
			t.Fatalf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestMemoryRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &MemoryRecommender{1, 1}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestMemorySettingsGroup(t *testing.T) {
	mem := uint64(1024)
	cpus := 4
	sg := GetSettingsGroup(MemoryLabel, "10", mem, cpus)
	// no matter how many calls, all calls should return the same
	for i := 0; i < 1000; i++ {
		if got := sg.Label(); got != MemoryLabel {
			t.Errorf("incorrect label: got %s want %s", got, MemoryLabel)
		}
		if got := sg.Keys(); got != nil {
			for i, k := range got {
				if k != MemoryKeys[i] {
					t.Errorf("incorrect key at %d: got %s want %s", i, k, MemoryKeys[i])
				}
			}
		} else {
			t.Errorf("keys is nil")
		}
		r := sg.GetRecommender().(*MemoryRecommender)
		// the above will panic if not true
		if r.cpus != cpus {
			t.Errorf("recommender has wrong number of cpus: got %d want %d", r.cpus, cpus)
		}
		if r.totalMemory != mem {
			t.Errorf("recommender has wrong number of mem: got %d want %d", r.totalMemory, mem)
		}
	}
}
