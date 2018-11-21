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

func TestParallelRecommenderIsAvailable(t *testing.T) {
	if r := NewParallelRecommender(0); r.IsAvailable() {
		t.Errorf("unexpectedly available for 0 cpus")
	}
	if r := NewParallelRecommender(1); r.IsAvailable() {
		t.Errorf("unexpectedly available for 1 cpus")
	}

	for i := 2; i < 1000; i++ {
		if r := NewParallelRecommender(i); !r.IsAvailable() {
			t.Errorf("unexpected UNavailable for %d cpus", i)
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

func TestParallelSettingsGroup(t *testing.T) {
	mem := uint64(1024)
	cpus := 4
	sg := GetSettingsGroup(ParallelLabel, mem, cpus)
	// no matter how many calls, all calls should return the same
	for i := 0; i < 1000; i++ {
		if got := sg.Label(); got != ParallelLabel {
			t.Errorf("incorrect label: got %s want %s", got, ParallelLabel)
		}
		if got := sg.Keys(); got != nil {
			for i, k := range got {
				if k != ParallelKeys[i] {
					t.Errorf("incorrect key at %d: got %s want %s", i, k, ParallelKeys[i])
				}
			}
		} else {
			t.Errorf("keys is nil")
		}
		r := sg.GetRecommender().(*ParallelRecommender)
		// the above will panic if not true
		if r.cpus != cpus {
			t.Errorf("recommender has wrong number of cpus: got %d want %d", r.cpus, cpus)
		}
	}
}
