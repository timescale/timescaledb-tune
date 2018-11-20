package main

import (
	"testing"
)

func TestParallelRecommenderRecommend(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		cpus int
		want string
	}{
		{
			desc: "max_worker_processes, 2",
			key:  maxWorkerProcessesKey,
			cpus: 2,
			want: "2",
		},
		{
			desc: "max_worker_processes, 4",
			key:  maxWorkerProcessesKey,
			cpus: 4,
			want: "4",
		},
		{
			desc: "max_worker_processes, 5",
			key:  maxWorkerProcessesKey,
			cpus: 5,
			want: "5",
		},
		{
			desc: "max_parallel_workers, 2",
			key:  maxParallelWorkers,
			cpus: 2,
			want: "2",
		},
		{
			desc: "max_parallel_workers, 4",
			key:  maxParallelWorkers,
			cpus: 4,
			want: "4",
		},
		{
			desc: "max_parallel_workers, 5",
			key:  maxParallelWorkers,
			cpus: 5,
			want: "5",
		},
		{
			desc: "max_parallel_workers_per_gather, 2",
			key:  maxParallelWorkersGatherKey,
			cpus: 2,
			want: "1",
		},
		{
			desc: "max_parallel_workers_per_gather, 4",
			key:  maxParallelWorkersGatherKey,
			cpus: 4,
			want: "2",
		},
		{
			desc: "max_parallel_workers_per_gather, 5",
			key:  maxParallelWorkersGatherKey,
			cpus: 5,
			want: "3",
		},
	}

	for _, c := range cases {
		r := &parallelRecommender{c.cpus}
		got := r.Recommend(c.key)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestParallelRecommenderRecommendPanics(t *testing.T) {
	func() {
		r := &parallelRecommender{5}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()

	func() {
		r := &parallelRecommender{1}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}
