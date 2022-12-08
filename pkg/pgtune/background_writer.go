package pgtune

const (
	BgwriterFlushAfterKey = "bgwriter_flush_after"

	promscaleDefaultBgwriterFlushAfter = "0"
)

// BgwriterLabel is the label used to refer to the background writer settings group
const BgwriterLabel = "background writer"

var BgwriterKeys = []string{
	BgwriterFlushAfterKey,
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
	case BgwriterFlushAfterKey:
		return promscaleDefaultBgwriterFlushAfter
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
