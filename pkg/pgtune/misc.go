package pgtune

import (
	"fmt"
	"math"
	"runtime"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// Keys in the conf file that are tunable but not in the other groupings
const (
	CheckpointKey     = "checkpoint_completion_target"
	StatsTargetKey    = "default_statistics_target"
	MaxConnectionsKey = "max_connections"
	RandomPageCostKey = "random_page_cost"
	MaxLocksPerTxKey  = "max_locks_per_transaction"
	EffectiveIOKey    = "effective_io_concurrency" // linux only

	checkpointDefault     = "0.9"
	statsTargetDefault    = "500"
	randomPageCostDefault = "1.1"
	effectiveIODefault    = "200"
)

// MaxConnectionsDefault is the recommended default value for max_connections.
const MaxConnectionsDefault uint64 = 50

// maxLocksValues gives the number of locks for a power-2 memory starting
// with sub-8GB. i.e.:
// < 8GB = 64
// >=8GB, < 16GB = 128
// >=16GB, < 32GB = 256
// >=32GB = 512
var maxLocksValues = []string{"64", "128", "256", "512"}

// MiscLabel is the label used to refer to the miscellaneous settings group
const MiscLabel = "miscellaneous"

// MiscKeys is an array of miscellaneous keys that are tunable
var MiscKeys = []string{
	StatsTargetKey,
	RandomPageCostKey,
	CheckpointKey,
	MaxConnectionsKey,
	MaxLocksPerTxKey,
	EffectiveIOKey,
}

// MiscRecommender gives recommendations for MiscKeys based on system resources.
type MiscRecommender struct {
	totalMemory uint64
	maxConns    uint64
}

// NewMiscRecommender returns a MiscRecommender (unaffected by system resources).
func NewMiscRecommender(totalMemory, maxConns uint64) *MiscRecommender {
	return &MiscRecommender{totalMemory, maxConns}
}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *MiscRecommender) IsAvailable() bool {
	return true
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
		conns := MaxConnectionsDefault
		if r.maxConns > conns {
			conns = r.maxConns
		}
		val = fmt.Sprintf("%d", conns)
	} else if key == RandomPageCostKey {
		val = randomPageCostDefault
	} else if key == MaxLocksPerTxKey {
		for i := len(maxLocksValues) - 1; i >= 1; i-- {
			limit := uint64(math.Pow(2.0, float64(2+i)))
			if r.totalMemory >= limit*parse.Gigabyte {
				return maxLocksValues[i]
			}
		}
		return maxLocksValues[0]
	} else if key == EffectiveIOKey {
		val = effectiveIODefault
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}

// MiscSettingsGroup is the SettingsGroup to represent settings that do not fit in other SettingsGroups.
type MiscSettingsGroup struct {
	totalMemory uint64
	maxConns    uint64
}

// Label should always return the value MiscLabel.
func (sg *MiscSettingsGroup) Label() string { return MiscLabel }

// Keys should always return the MiscKeys slice.
func (sg *MiscSettingsGroup) Keys() []string {
	if runtime.GOOS != "linux" {
		return MiscKeys[:len(MiscKeys)-1]
	}
	return MiscKeys
}

// GetRecommender should return a new MiscRecommender.
func (sg *MiscSettingsGroup) GetRecommender() Recommender {
	return NewMiscRecommender(sg.totalMemory, sg.maxConns)
}
