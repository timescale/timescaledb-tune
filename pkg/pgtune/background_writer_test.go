package pgtune

import (
	"fmt"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
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
	if val := r.Recommend(BgwriterDelayKey); val != NoRecommendation {
		t.Errorf("Expected no recommendation for key %s but got %s", BgwriterDelayKey, val)
	}
	if val := r.Recommend(BgwriterLRUMaxPagesKey); val != NoRecommendation {
		t.Errorf("Expected no recommendation for key %s but got %s", BgwriterLRUMaxPagesKey, val)
	}

	// the promscale profile should have recommendations
	r = sg.GetRecommender(PromscaleProfile)
	if val := r.Recommend(BgwriterDelayKey); val != promscaleDefaultBgwriterDelay {
		t.Errorf("Expected %s for key %s but got %s", promscaleDefaultBgwriterDelay, BgwriterDelayKey, val)
	}
	if val := r.Recommend(BgwriterLRUMaxPagesKey); val != promscaleDefaultBgwriterLRUMaxPages {
		t.Errorf("Expected %s for key %s but got %s", promscaleDefaultBgwriterLRUMaxPages, BgwriterLRUMaxPagesKey, val)
	}
}

func TestPromscaleBgwriterRecommender(t *testing.T) {
	r := PromscaleBgwriterRecommender{}
	if !r.IsAvailable() {
		t.Error("PromscaleBgwriterRecommender should always be available")
	}
	if val := r.Recommend(BgwriterDelayKey); val != promscaleDefaultBgwriterDelay {
		t.Errorf("Expected %s for key %s but got %s", promscaleDefaultBgwriterDelay, BgwriterDelayKey, val)
	}
	if val := r.Recommend(BgwriterLRUMaxPagesKey); val != promscaleDefaultBgwriterLRUMaxPages {
		t.Errorf("Expected %s for key %s but got %s", promscaleDefaultBgwriterLRUMaxPages, BgwriterLRUMaxPagesKey, val)
	}
}

func TestBgwriterFloatParserParseFloat(t *testing.T) {
	v := &BgwriterFloatParser{}

	s := "100"
	want := 100.0
	got, err := v.ParseFloat(BgwriterLRUMaxPagesKey, s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}

	s = "33" + parse.Minutes.String()
	conversion, _ := parse.TimeConversion(parse.Minutes, parse.Milliseconds)
	want = 33.0 * conversion
	got, err = v.ParseFloat(BgwriterDelayKey, s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}
}
