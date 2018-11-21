package pgtune

import (
	"fmt"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

// Keys in the conf file that are tuned related to the WAL
const (
	WALBuffersKey = "wal_buffers"
	MinWALKey     = "min_wal_size"
	MaxWALKey     = "max_wal_size"

	walBuffersThreshold = 2 * parse.Gigabyte
	walBuffersDefault   = 16 * parse.Megabyte
	minWalBytes         = 4 * parse.Gigabyte
	maxWalBytes         = 8 * parse.Gigabyte
)

// WALLabel is the label used to refer to the WAL settings group
const WALLabel = "WAL"

// WALKeys is an array of keys that are tunable for the WAL
var WALKeys = []string{
	WALBuffersKey,
	MinWALKey,
	MaxWALKey,
}

// WALRecommender gives recommendations for WALKeys based on system resources
type WALRecommender struct {
	totalMemory uint64
}

// NewWALRecommender returns a WALRecommender that recommends based on the given
// totalMemory bytes.
func NewWALRecommender(totalMemory uint64) *WALRecommender {
	return &WALRecommender{totalMemory}
}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *WALRecommender) IsAvailable() bool {
	return true
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *WALRecommender) Recommend(key string) string {
	var val string
	if key == WALBuffersKey {
		if r.totalMemory < walBuffersThreshold {
			temp := (float64(r.totalMemory) / float64(parse.Gigabyte)) * (7864.0 * float64(parse.Kilobyte))
			val = parse.BytesToPGFormat(uint64(temp))
		} else {
			val = parse.BytesToPGFormat(walBuffersDefault)
		}
	} else if key == MinWALKey {
		val = parse.BytesToPGFormat(minWalBytes)
	} else if key == MaxWALKey {
		val = parse.BytesToPGFormat(maxWalBytes)
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}

// WALSettingsGroup is the SettingsGroup to represent settings that affect WAL usage.
type WALSettingsGroup struct {
	totalMemory uint64
}

// Label should always return the value WALLabel.
func (sg *WALSettingsGroup) Label() string { return WALLabel }

// Keys should always return the WALKeys slice.
func (sg *WALSettingsGroup) Keys() []string { return WALKeys }

// GetRecommender should return a new WALRecommender.
func (sg *WALSettingsGroup) GetRecommender() Recommender {
	return NewWALRecommender(sg.totalMemory)
}
