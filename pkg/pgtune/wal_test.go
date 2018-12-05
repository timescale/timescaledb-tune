package pgtune

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

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
	valFmt := "%d%s"
	cases := []struct {
		desc        string
		key         string
		totalMemory uint64
		want        string
	}{
		{
			desc:        "wal_buffers, 1GB",
			key:         WALBuffersKey,
			totalMemory: parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, 7864, parse.KB), // from pgtune
		},
		{
			desc:        "wal_buffers, 1.5GB",
			key:         WALBuffersKey,
			totalMemory: uint64(1.5 * float64(parse.Gigabyte)),
			want:        fmt.Sprintf(valFmt, 11796, parse.KB), // from pgtune
		},
		{
			desc:        "wal_buffers, 2GB",
			key:         WALBuffersKey,
			totalMemory: 2 * parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, 16, parse.MB),
		},
		{
			desc:        "wal_buffers, > 2GB",
			key:         WALBuffersKey,
			totalMemory: 10 * parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, walBuffersDefault/parse.Megabyte, parse.MB),
		},
		{
			desc:        "min_wal_size is constant #1",
			key:         MinWALKey,
			totalMemory: parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, minWalBytes/parse.Gigabyte, parse.GB),
		},
		{
			desc:        "min_wal_size is constant #2",
			key:         MinWALKey,
			totalMemory: 10 * parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, minWalBytes/parse.Gigabyte, parse.GB),
		},
		{
			desc:        "max_wal_size is constant #1",
			key:         MaxWALKey,
			totalMemory: 1 * parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, maxWalBytes/parse.Gigabyte, parse.GB),
		},
		{
			desc:        "max_wal_size is constant #2",
			key:         MaxWALKey,
			totalMemory: 10 * parse.Gigabyte,
			want:        fmt.Sprintf(valFmt, maxWalBytes/parse.Gigabyte, parse.GB),
		},
	}

	for _, c := range cases {
		r := &WALRecommender{c.totalMemory}
		got := r.Recommend(c.key)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
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
	mem := uint64(1024)
	cpus := 4
	sg := GetSettingsGroup(WALLabel, "10", mem, cpus)
	// no matter how many calls, all calls should return the same
	for i := 0; i < 1000; i++ {
		if got := sg.Label(); got != WALLabel {
			t.Errorf("incorrect label: got %s want %s", got, WALLabel)
		}
		if got := sg.Keys(); got != nil {
			for i, k := range got {
				if k != WALKeys[i] {
					t.Errorf("incorrect key at %d: got %s want %s", i, k, WALKeys[i])
				}
			}
		} else {
			t.Errorf("keys is nil")
		}
		r := sg.GetRecommender().(*WALRecommender)
		// the above will panic if not true

		if r.totalMemory != mem {
			t.Errorf("recommender has wrong number of mem: got %d want %d", r.totalMemory, mem)
		}
	}
}
