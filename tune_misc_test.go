package main

import (
	"testing"
)

func TestMiscRecommenderRecommend(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		want string
	}{
		{
			desc: checkpointKey,
			key:  checkpointKey,
			want: checkpointDefault,
		},
		{
			desc: statsTargetKey,
			key:  statsTargetKey,
			want: statsTargetDefault,
		},
		{
			desc: maxConnectionsKey,
			key:  maxConnectionsKey,
			want: maxConnectionsDefault,
		},
		{
			desc: randomPageCostKey,
			key:  randomPageCostKey,
			want: randomPageCostDefault,
		},
		{
			desc: effectiveIOKey,
			key:  effectiveIOKey,
			want: effectiveIODefault,
		},
	}

	for _, c := range cases {
		r := &miscRecommender{}
		got := r.Recommend(c.key)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
		}
	}
}

func TestMiscRecommenderRecommendPanic(t *testing.T) {
	func() {
		r := &miscRecommender{}
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		r.Recommend("foo")
	}()
}
