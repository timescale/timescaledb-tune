package pgtune

import (
	"testing"
)

func TestNewMiscRecommender(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		r := NewMiscRecommender()
		if r == nil {
			t.Errorf("unexpected nil recommender")
		}

		if !r.IsAvailable() {
			t.Errorf("unexpectedly not available")
		}
	}
}

func TestMiscRecommenderRecommend(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		want string
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
			desc: EffectiveIOKey,
			key:  EffectiveIOKey,
			want: effectiveIODefault,
		},
	}

	for _, c := range cases {
		r := &MiscRecommender{}
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
	mem := uint64(1024)
	cpus := 4
	sg := GetSettingsGroup(MiscLabel, mem, cpus)
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
