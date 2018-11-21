package pgtune

import "fmt"

// Keys in the conf file that are tunable but not in the other groupings
const (
	CheckpointKey     = "checkpoint_completion_target"
	StatsTargetKey    = "default_statistics_target"
	MaxConnectionsKey = "max_connections"
	RandomPageCostKey = "random_page_cost"
	EffectiveIOKey    = "effective_io_concurrency" // linux only

	checkpointDefault     = "0.9"
	statsTargetDefault    = "500"
	maxConnectionsDefault = "20"
	randomPageCostDefault = "1.1"
	effectiveIODefault    = "200"
)

// MiscKeys is an array of miscellaneous keys that are tunable
var MiscKeys = []string{
	StatsTargetKey,
	RandomPageCostKey,
	EffectiveIOKey,
	CheckpointKey,
	MaxConnectionsKey,
}

// MiscRecommender gives recommendations for MiscKeys based on system resources.
type MiscRecommender struct{}

// NewMiscRecommender returns a MiscRecommender (unaffected by system resources).
func NewMiscRecommender() *MiscRecommender {
	return &MiscRecommender{}
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *MiscRecommender) Recommend(key string) string {
	var val string
	if key == CheckpointKey {
		val = checkpointDefault
	} else if key == StatsTargetKey {
		val = statsTargetDefault
	} else if key == MaxConnectionsKey {
		val = maxConnectionsDefault
	} else if key == RandomPageCostKey {
		val = randomPageCostDefault
	} else if key == EffectiveIOKey {
		val = effectiveIODefault
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}
