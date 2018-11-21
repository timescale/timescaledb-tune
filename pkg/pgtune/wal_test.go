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
		if got := r.totalMem; got != mem {
			t.Errorf("recommender has incorrect memory: got %d want %d", got, mem)
		}
	}
}

func TestWALRecommenderRecommend(t *testing.T) {
	valFmt := "%d%s"
	cases := []struct {
		desc     string
		key      string
		totalMem uint64
		want     string
	}{
		{
			desc:     "wal_buffers, 1GB",
			key:      WALBuffersKey,
			totalMem: parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, 7864, parse.KB), // from pgtune
		},
		{
			desc:     "wal_buffers, 1.5GB",
			key:      WALBuffersKey,
			totalMem: uint64(1.5 * float64(parse.Gigabyte)),
			want:     fmt.Sprintf(valFmt, 11796, parse.KB), // from pgtune
		},
		{
			desc:     "wal_buffers, 2GB",
			key:      WALBuffersKey,
			totalMem: 2 * parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, 16, parse.MB),
		},
		{
			desc:     "wal_buffers, > 2GB",
			key:      WALBuffersKey,
			totalMem: 10 * parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, walBuffersDefault/parse.Megabyte, parse.MB),
		},
		{
			desc:     "min_wal_size is constant #1",
			key:      MinWALKey,
			totalMem: parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, minWalBytes/parse.Gigabyte, parse.GB),
		},
		{
			desc:     "min_wal_size is constant #2",
			key:      MinWALKey,
			totalMem: 10 * parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, minWalBytes/parse.Gigabyte, parse.GB),
		},
		{
			desc:     "max_wal_size is constant #1",
			key:      MaxWALKey,
			totalMem: 1 * parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, maxWalBytes/parse.Gigabyte, parse.GB),
		},
		{
			desc:     "max_wal_size is constant #2",
			key:      MaxWALKey,
			totalMem: 10 * parse.Gigabyte,
			want:     fmt.Sprintf(valFmt, maxWalBytes/parse.Gigabyte, parse.GB),
		},
	}

	for _, c := range cases {
		r := &WALRecommender{c.totalMem}
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
