package main

import "fmt"

const (
	walBuffersKey = "wal_buffers"
	minWalKey     = "min_wal_size"
	maxWalKey     = "max_wal_size"

	walBuffersThreshold = 2 * gigabyte
	walBuffersDefault   = 16 * megabyte
	minWalBytes         = 4 * gigabyte
	maxWalBytes         = 8 * gigabyte
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
			temp := (float64(r.totalMem) / float64(gigabyte)) * (7864.0 * float64(kilobyte))
			val = bytesPGFormat(uint64(temp))
		} else {
			val = bytesPGFormat(walBuffersDefault)
		}
	} else if key == minWalKey {
		val = bytesPGFormat(minWalBytes)
	} else if key == maxWalKey {
		val = bytesPGFormat(maxWalBytes)
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}
