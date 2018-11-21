package pgtune

import (
	"math/rand"
	"testing"
)

func TestNewParallelRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		cpus := rand.Intn(128)
		r := NewParallelRecommender(cpus)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.cpus; got != cpus {
			t.Errorf("recommender has incorrect cpus: got %d want %d", got, cpus)
		}
	}
}

func TestParallelRecommenderRecommend(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		cpus int
		want string
	}{
		{
			desc: "max_worker_processes, 2",
			key:  MaxWorkerProcessesKey,
			cpus: 2,
			want: "2",
		},
		{
			desc: "max_worker_processes, 4",
			key:  MaxWorkerProcessesKey,
			cpus: 4,
			want: "4",
		},
		{
			desc: "max_worker_processes, 5",
			key:  MaxWorkerProcessesKey,
			cpus: 5,
			want: "5",
		},
		{
			desc: "max_parallel_workers, 2",
			key:  MaxParallelWorkers,
			cpus: 2,
			want: "2",
		},
		{
			desc: "max_parallel_workers, 4",
			key:  MaxParallelWorkers,
			cpus: 4,
			want: "4",
		},
		{
			desc: "max_parallel_workers, 5",
			key:  MaxParallelWorkers,
			cpus: 5,
			want: "5",
		},
		{
			desc: "max_parallel_workers_per_gather, 2",
			key:  MaxParallelWorkersGatherKey,
			cpus: 2,
			want: "1",
		},
		{
			desc: "max_parallel_workers_per_gather, 4",
			key:  MaxParallelWorkersGatherKey,
			cpus: 4,
			want: "2",
		},
		{
			desc: "max_parallel_workers_per_gather, 5",
			key:  MaxParallelWorkersGatherKey,
			cpus: 5,
			want: "3",
		},
	}

	for _, c := range cases {
		r := &ParallelRecommender{c.cpus}
		got := r.Recommend(c.key)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestParallelRecommenderRecommendPanics(t *testing.T) {
	func() {
		r := &ParallelRecommender{5}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()

	func() {
		r := &ParallelRecommender{1}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}
