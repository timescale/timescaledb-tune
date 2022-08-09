package pgtune

import "github.com/timescale/timescaledb-tune/internal/parse"

const (
	BgwriterDelayKey       = "bgwriter_delay"
	BgwriterLRUMaxPagesKey = "bgwriter_lru_maxpages"

	promscaleDefaultBgwriterDelay       = "10ms"
	promscaleDefaultBgwriterLRUMaxPages = "100000"
)

// BgwriterLabel is the label used to refer to the background writer settings group
const BgwriterLabel = "background writer"

var BgwriterKeys = []string{
	BgwriterDelayKey,
	BgwriterLRUMaxPagesKey,
}

// PromscaleBgwriterRecommender gives recommendations for the background writer for the promscale profile
type PromscaleBgwriterRecommender struct{}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *PromscaleBgwriterRecommender) IsAvailable() bool {
	return true
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *PromscaleBgwriterRecommender) Recommend(key string) string {
	switch key {
	case BgwriterDelayKey:
		return promscaleDefaultBgwriterDelay
	case BgwriterLRUMaxPagesKey:
		return promscaleDefaultBgwriterLRUMaxPages
	default:
		return NoRecommendation
	}
}

// BgwriterSettingsGroup is the SettingsGroup to represent settings that affect the background writer.
type BgwriterSettingsGroup struct {
	totalMemory uint64
	cpus        int
	maxConns    uint64
}

// Label should always return the value BgwriterLabel.
func (sg *BgwriterSettingsGroup) Label() string { return BgwriterLabel }

// Keys should always return the BgwriterKeys slice.
func (sg *BgwriterSettingsGroup) Keys() []string { return BgwriterKeys }

// GetRecommender should return a new Recommender.
func (sg *BgwriterSettingsGroup) GetRecommender(profile Profile) Recommender {
	switch profile {
	case PromscaleProfile:
		return &PromscaleBgwriterRecommender{}
	default:
		return &NullRecommender{}
	}
}

type BgwriterFloatParser struct{}

func (v *BgwriterFloatParser) ParseFloat(key string, s string) (float64, error) {
	switch key {
	case BgwriterDelayKey:
		val, units, err := parse.PGFormatToTime(s, parse.Milliseconds, parse.VarTypeInteger)
		if err != nil {
			return val, err
		}
		conv, err := parse.TimeConversion(units, parse.Milliseconds)
		if err != nil {
			return val, err
		}
		return val * conv, nil
	default:
		bfp := &numericFloatParser{}
		return bfp.ParseFloat(key, s)
	}
}
