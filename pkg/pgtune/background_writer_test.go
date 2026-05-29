package pgtune

import (
	"fmt"
	"testing"
)

func TestBgwriterSettingsGroup_GetRecommender(t *testing.T) {
	cases := []struct {
		profile     Profile
		recommender string
	}{
		{DefaultProfile, "*pgtune.NullRecommender"},
		{PromscaleProfile, "*pgtune.PromscaleBgwriterRecommender"},
	}

	sg := BgwriterSettingsGroup{}
	for _, k := range cases {
		r := sg.GetRecommender(k.profile)
		y := fmt.Sprintf("%T", r)
		if y != k.recommender {
			t.Errorf("Expected to get a %s using the %s profile but got %s", k.recommender, k.profile, y)
		}
	}
}

func TestBgwriterSettingsGroupRecommend(t *testing.T) {
	sg := BgwriterSettingsGroup{}

	// the default profile should provide no recommendations
	r := sg.GetRecommender(DefaultProfile)
	if val := r.Recommend(BgwriterFlushAfterKey); val != NoRecommendation {
		t.Errorf("Expected no recommendation for key %s but got %s", BgwriterFlushAfterKey, val)
	}

	// the promscale profile should have recommendations
	r = sg.GetRecommender(PromscaleProfile)
	if val := r.Recommend(BgwriterFlushAfterKey); val != promscaleDefaultBgwriterFlushAfter {
		t.Errorf("Expected %s for key %s but got %s", promscaleDefaultBgwriterFlushAfter, BgwriterFlushAfterKey, val)
	}
}

func TestPromscaleBgwriterRecommender(t *testing.T) {
	r := PromscaleBgwriterRecommender{}
	if !r.IsAvailable() {
		t.Error("PromscaleBgwriterRecommender should always be available")
	}
	if val := r.Recommend(BgwriterFlushAfterKey); val != promscaleDefaultBgwriterFlushAfter {
		t.Errorf("Expected %s for key %s but got %s", promscaleDefaultBgwriterFlushAfter, BgwriterFlushAfterKey, val)
	}
}
