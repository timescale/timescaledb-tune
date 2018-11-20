package main

import (
	"fmt"
	"testing"
)

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
			key:      walBuffersKey,
			totalMem: 1 * gigabyte,
			want:     fmt.Sprintf(valFmt, 7864, kb), // from pgtune
		},
		{
			desc:     "wal_buffers, 1.5GB",
			key:      walBuffersKey,
			totalMem: uint64(1.5 * float64(gigabyte)),
			want:     fmt.Sprintf(valFmt, 11796, kb), // from pgtune
		},
		{
			desc:     "wal_buffers, 2GB",
			key:      walBuffersKey,
			totalMem: 2 * gigabyte,
			want:     fmt.Sprintf(valFmt, 16, mb),
		},
		{
			desc:     "wal_buffers, > 2GB",
			key:      walBuffersKey,
			totalMem: 10 * gigabyte,
			want:     fmt.Sprintf(valFmt, walBuffersDefault/megabyte, mb),
		},
		{
			desc:     "min_wal_size is constant #1",
			key:      minWalKey,
			totalMem: 1 * gigabyte,
			want:     fmt.Sprintf(valFmt, minWalBytes/gigabyte, gb),
		},
		{
			desc:     "min_wal_size is constant #2",
			key:      minWalKey,
			totalMem: 10 * gigabyte,
			want:     fmt.Sprintf(valFmt, minWalBytes/gigabyte, gb),
		},
		{
			desc:     "max_wal_size is constant #1",
			key:      maxWalKey,
			totalMem: 1 * gigabyte,
			want:     fmt.Sprintf(valFmt, maxWalBytes/gigabyte, gb),
		},
		{
			desc:     "max_wal_size is constant #2",
			key:      maxWalKey,
			totalMem: 10 * gigabyte,
			want:     fmt.Sprintf(valFmt, maxWalBytes/gigabyte, gb),
		},
	}

	for _, c := range cases {
		r := &walRecommender{c.totalMem}
		got := r.Recommend(c.key)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestWALRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &walRecommender{0}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}
