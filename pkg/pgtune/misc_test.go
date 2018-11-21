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
