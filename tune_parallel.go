package main

import (
	"fmt"
	"math"
)

const (
	maxWorkerProcessesKey       = "max_worker_processes"
	maxParallelWorkersGatherKey = "max_parallel_workers_per_gather"
	maxParallelWorkers          = "max_parallel_workers"

	errOneCPU = "cannot make recommendations with just 1 CPU"
)

var parallelKeys = []string{
	maxWorkerProcessesKey,
	maxParallelWorkersGatherKey,
	maxParallelWorkers,
}

type parallelRecommender struct {
	cpus int
}

func (r *parallelRecommender) Recommend(key string) string {
	var val string
	if r.cpus <= 1 {
		panic(errOneCPU)
	}
	if key == maxWorkerProcessesKey || key == maxParallelWorkers {
		val = fmt.Sprintf("%d", r.cpus)
	} else if key == maxParallelWorkersGatherKey {
		val = fmt.Sprintf("%d", int(math.Round(float64(r.cpus)/2.0)))
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}
