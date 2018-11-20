package main

import (
	"fmt"
	"math"
	"runtime"
)

const (
	sharedBuffersKey      = "shared_buffers"
	effectiveCacheKey     = "effective_cache_size"
	maintenanceWorkMemKey = "maintenance_work_mem"
	workMemKey            = "work_mem"

	sharedBuffersWindows = 512 * megabyte
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
			val = bytesPGFormat(sharedBuffersWindows)
		} else {
			val = bytesPGFormat(r.totalMem / 4)
		}
	} else if key == effectiveCacheKey {
		val = bytesPGFormat((r.totalMem * 3) / 4)
	} else if key == maintenanceWorkMemKey {
		temp := (float64(r.totalMem) / float64(gigabyte)) * (128.0 * float64(megabyte))
		if temp > (2 * gigabyte) {
			temp = 2 * gigabyte
		}
		val = bytesPGFormat(uint64(temp))
	} else if key == workMemKey {
		if runtime.GOOS == osWindows {
			val = r.recommendWindows()
		} else {
			cpuFactor := math.Round(float64(r.cpus) / 2.0)
			temp := (float64(r.totalMem) / float64(gigabyte)) * (6.4 * float64(megabyte)) / cpuFactor
			val = bytesPGFormat(uint64(temp))
		}
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}

func (r *memoryRecommender) recommendWindows() string {
	cpuFactor := math.Round(float64(r.cpus) / 2.0)
	if r.totalMem <= 2*gigabyte {
		temp := (float64(r.totalMem) / float64(gigabyte)) * (6.4 * float64(megabyte)) / cpuFactor
		return bytesPGFormat(uint64(temp))
	}
	base := 2.0 * 6.4 * float64(megabyte)
	temp := ((float64(r.totalMem)/float64(gigabyte)-2)*(8.53336*float64(megabyte)) + base) / cpuFactor
	return bytesPGFormat(uint64(temp))
}
