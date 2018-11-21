package main

import (
	"fmt"
	"math"
	"runtime"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

const (
	sharedBuffersKey      = "shared_buffers"
	effectiveCacheKey     = "effective_cache_size"
	maintenanceWorkMemKey = "maintenance_work_mem"
	workMemKey            = "work_mem"

	sharedBuffersWindows = 512 * parse.Megabyte
)

var memoryKeys = []string{
	sharedBuffersKey,
	effectiveCacheKey,
	maintenanceWorkMemKey,
	workMemKey,
}

type memoryRecommender struct {
	totalMem uint64
	cpus     int
}

func (r *memoryRecommender) Recommend(key string) string {
	var val string
	if key == sharedBuffersKey {
		if runtime.GOOS == osWindows {
			val = parse.BytesToPGFormat(sharedBuffersWindows)
		} else {
			val = parse.BytesToPGFormat(r.totalMem / 4)
		}
	} else if key == effectiveCacheKey {
		val = parse.BytesToPGFormat((r.totalMem * 3) / 4)
	} else if key == maintenanceWorkMemKey {
		temp := (float64(r.totalMem) / float64(parse.Gigabyte)) * (128.0 * float64(parse.Megabyte))
		if temp > (2 * parse.Gigabyte) {
			temp = 2 * parse.Gigabyte
		}
		val = parse.BytesToPGFormat(uint64(temp))
	} else if key == workMemKey {
		if runtime.GOOS == osWindows {
			val = r.recommendWindows()
		} else {
			cpuFactor := math.Round(float64(r.cpus) / 2.0)
			temp := (float64(r.totalMem) / float64(parse.Gigabyte)) * (6.4 * float64(parse.Megabyte)) / cpuFactor
			val = parse.BytesToPGFormat(uint64(temp))
		}
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}

func (r *memoryRecommender) recommendWindows() string {
	cpuFactor := math.Round(float64(r.cpus) / 2.0)
	if r.totalMem <= 2*parse.Gigabyte {
		temp := (float64(r.totalMem) / float64(parse.Gigabyte)) * (6.4 * float64(parse.Megabyte)) / cpuFactor
		return parse.BytesToPGFormat(uint64(temp))
	}
	base := 2.0 * 6.4 * float64(parse.Megabyte)
	temp := ((float64(r.totalMem)/float64(parse.Gigabyte)-2)*(8.53336*float64(parse.Megabyte)) + base) / cpuFactor
	return parse.BytesToPGFormat(uint64(temp))
}
