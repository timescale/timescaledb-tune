package main

import (
	"fmt"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

func TestMemoryRecommenderRecommendWindows(t *testing.T) {
	cases := []struct {
		desc     string
		totalMem uint64
		cpus     int
		want     string
	}{
		{
			desc:     "1GB",
			totalMem: 1 * parse.Gigabyte,
			cpus:     1,
			want:     "6553" + parse.KB, // from pgtune
		},
		{
			desc:     "1GB, 4 cpus",
			totalMem: 1 * parse.Gigabyte,
			cpus:     4,
			want:     "3276" + parse.KB, // from pgtune
		},
		{
			desc:     "2GB",
			totalMem: 2 * parse.Gigabyte,
			cpus:     1,
			want:     "13107" + parse.KB, // from pgtune
		},
		{
			desc:     "2GB, 5 cpus",
			totalMem: 2 * parse.Gigabyte,
			cpus:     5,
			want:     "4369" + parse.KB, // from pgtune
		},
		{
			desc:     "3GB",
			totalMem: 3 * parse.Gigabyte,
			cpus:     1,
			want:     "21845" + parse.KB, // from pgtune
		},
		{
			desc:     "3GB, 3 cpus",
			totalMem: 3 * parse.Gigabyte,
			cpus:     3,
			want:     "10922" + parse.KB, // from pgtune
		},
		{
			desc:     "8GB",
			totalMem: 8 * parse.Gigabyte,
			cpus:     1,
			want:     "64" + parse.MB, // from pgtune
		},
		{
			desc:     "8GB, 8 cpus",
			totalMem: 8 * parse.Gigabyte,
			cpus:     8,
			want:     "16" + parse.MB, // from pgtune
		},
		{
			desc:     "16GB",
			totalMem: 16 * parse.Gigabyte,
			cpus:     1,
			want:     "135441" + parse.KB, // from pgtune
		},
		{
			desc:     "16GB, 10 cpus",
			totalMem: 16 * parse.Gigabyte,
			cpus:     10,
			want:     "27088" + parse.KB, // from pgtune
		},
	}

	for _, c := range cases {
		mr := &memoryRecommender{c.totalMem, c.cpus}
		if got := mr.recommendWindows(); got != c.want {
			t.Errorf("%s: incorrect value: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestMemoryRecommenderRecommend(t *testing.T) {
	valFmt := "%d%s"
	cases := []struct {
		desc     string
		key      string
		totalMem uint64
		cpus     int
		want     string
	}{
		{
			desc:     "shared_buffers, uneven divide",
			key:      sharedBuffersKey,
			totalMem: 10 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, 2560, parse.MB),
		},
		{
			desc:     "shared_buffers, even divide",
			key:      sharedBuffersKey,
			totalMem: 8 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, 2, parse.GB),
		},
		{
			desc:     "effective key, uneven divide",
			key:      effectiveCacheKey,
			totalMem: 10 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, uint64(7.5*1024.0), parse.MB),
		},
		{
			desc:     "effective key, even divide",
			key:      effectiveCacheKey,
			totalMem: 12 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, 9, parse.GB),
		},
		{
			desc:     "maintenance_work_mem",
			key:      maintenanceWorkMemKey,
			totalMem: 6 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, 768, parse.MB),
		},
		{
			desc:     "maintenance_work_mem, over max",
			key:      maintenanceWorkMemKey,
			totalMem: 32 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, 2, parse.GB),
		},
		{
			desc:     "work_mem",
			key:      workMemKey,
			totalMem: 8 * parse.Gigabyte,
			cpus:     1,
			want:     fmt.Sprintf(valFmt, 52428, parse.KB),
		},
		{
			desc:     "work_mem, multiple CPUs",
			key:      workMemKey,
			totalMem: 8 * parse.Gigabyte,
			cpus:     4,
			want:     fmt.Sprintf(valFmt, 26214, parse.KB),
		},
	}

	for _, c := range cases {
		mr := &memoryRecommender{c.totalMem, c.cpus}
		got := mr.Recommend(c.key)
		if got != c.want {
			t.Fatalf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestMemoryRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &memoryRecommender{1, 1}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}
