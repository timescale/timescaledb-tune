package pgtune

import (
	"math/rand"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

func TestNewMiscRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		mem := rand.Uint64()
		r := NewMiscRecommender(mem)
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}
		if got := r.totalMemory; got != mem {
			t.Errorf("recommender has incorrect memory: got %d want %d", got, mem)
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestMiscRecommenderRecommend(t *testing.T) {
	cases := []struct {
		desc        string
		totalMemory uint64
		key         string
		want        string
	}{
		{
			desc: CheckpointKey,
			key:  CheckpointKey,
			want: checkpointDefault,
		},
		{
			desc: StatsTargetKey,
			key:  StatsTargetKey,
			want: statsTargetDefault,
		},
		{
			desc: MaxConnectionsKey,
			key:  MaxConnectionsKey,
			want: maxConnectionsDefault,
		},
		{
			desc: RandomPageCostKey,
			key:  RandomPageCostKey,
			want: randomPageCostDefault,
		},
		{
			desc:        MaxLocksPerTx + " < 8gb",
			totalMemory: 7 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[0],
		},
		{
			desc:        MaxLocksPerTx + " = 8gb",
			totalMemory: 8 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[1],
		},
		{
			desc:        MaxLocksPerTx + " > 8gb, < 16GB",
			totalMemory: 15 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[1],
		},
		{
			desc:        MaxLocksPerTx + " = 16GB",
			totalMemory: 16 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[2],
		},
		{
			desc:        MaxLocksPerTx + " > 16gb, < 32GB",
			totalMemory: 24 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[2],
		},
		{
			desc:        MaxLocksPerTx + " = 32GB",
			totalMemory: 32 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[3],
		},
		{
			desc:        MaxLocksPerTx + " > 32gb",
			totalMemory: 80 * parse.Gigabyte,
			key:         MaxLocksPerTx,
			want:        maxLocksValues[3],
		},
		{
			desc: EffectiveIOKey,
			key:  EffectiveIOKey,
			want: effectiveIODefault,
		},
	}

	for _, c := range cases {
		r := &MiscRecommender{c.totalMemory}
		got := r.Recommend(c.key)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestMiscRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &MiscRecommender{}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}

func TestMiscSettingsGroup(t *testing.T) {
	config := NewSystemConfig(1024, 4, "10")
	sg := GetSettingsGroup(MiscLabel, config)
	// no matter how many calls, all calls should return the same
	for i := 0; i < 1000; i++ {
		if got := sg.Label(); got != MiscLabel {
			t.Errorf("incorrect label: got %s want %s", got, MiscLabel)
		}
		if got := sg.Keys(); got != nil {
			for i, k := range got {
				if k != MiscKeys[i] {
					t.Errorf("incorrect key at %d: got %s want %s", i, k, MiscKeys[i])
				}
			}
		} else {
			t.Errorf("keys is nil")
		}
		_ = sg.GetRecommender().(*MiscRecommender)
		// the above will panic if not true
	}
}
