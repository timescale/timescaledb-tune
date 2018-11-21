package main

import (
	"fmt"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

const (
	walBuffersKey = "wal_buffers"
	minWalKey     = "min_wal_size"
	maxWalKey     = "max_wal_size"

	walBuffersThreshold = 2 * parse.Gigabyte
	walBuffersDefault   = 16 * parse.Megabyte
	minWalBytes         = 4 * parse.Gigabyte
	maxWalBytes         = 8 * parse.Gigabyte
)

var walKeys = []string{
	walBuffersKey,
	minWalKey,
	maxWalKey,
}

type walRecommender struct {
	totalMem uint64
}

func (r *walRecommender) Recommend(key string) string {
	var val string
	if key == walBuffersKey {
		if r.totalMem < walBuffersThreshold {
			temp := (float64(r.totalMem) / float64(parse.Gigabyte)) * (7864.0 * float64(parse.Kilobyte))
			val = parse.BytesToPGFormat(uint64(temp))
		} else {
			val = parse.BytesToPGFormat(walBuffersDefault)
		}
	} else if key == minWalKey {
		val = parse.BytesToPGFormat(minWalBytes)
	} else if key == maxWalKey {
		val = parse.BytesToPGFormat(maxWalBytes)
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}
