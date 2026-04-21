package pgtune

import "github.com/timescale/timescaledb-tune/internal/parse"

// Keys in the conf file that are tuned related to pg_textsearch.
const (
	MemoryLimitKey = "pg_textsearch.memory_limit"
)

// PgTextsearchLabel is the label used to refer to the pg_textsearch settings group.
const PgTextsearchLabel = "pg_textsearch"

// PgTextsearchKeys is an array of keys tunable for pg_textsearch.
var PgTextsearchKeys = []string{
	MemoryLimitKey,
}

// PgTextsearchRecommender gives recommendations for pg_textsearch GUCs based on
// total system memory and the active tuning profile. It is only available when
// pg_textsearch appears in shared_preload_libraries.
type PgTextsearchRecommender struct {
	totalMemory uint64
	profile     Profile
	enabled     bool
}

// NewPgTextsearchRecommender returns a PgTextsearchRecommender for the given
// total memory and profile. The enabled flag reflects whether pg_textsearch
// was detected in shared_preload_libraries.
func NewPgTextsearchRecommender(totalMemory uint64, profile Profile, enabled bool) *PgTextsearchRecommender {
	return &PgTextsearchRecommender{totalMemory, profile, enabled}
}

// IsAvailable returns true only when pg_textsearch is present in
// shared_preload_libraries.
func (r *PgTextsearchRecommender) IsAvailable() bool {
	return r.enabled
}

// Recommend returns the recommended PostgreSQL-formatted value for a given key.
func (r *PgTextsearchRecommender) Recommend(key string) string {
	if key != MemoryLimitKey {
		return NoRecommendation
	}
	// memory_limit caps the DSA shared memory pg_textsearch uses for its
	// in-memory write buffers. The promscale profile sets shared_buffers to
	// totalMemory/2 rather than the default totalMemory/4, so leave a
	// proportionally smaller share here to keep overall memory use bounded.
	divisor := uint64(4)
	if r.profile == PromscaleProfile {
		divisor = 8
	}
	return parse.BytesToPGFormat(r.totalMemory / divisor)
}

// PgTextsearchSettingsGroup is the SettingsGroup for pg_textsearch.
type PgTextsearchSettingsGroup struct {
	totalMemory uint64
	enabled     bool
}

// Label should always return PgTextsearchLabel.
func (sg *PgTextsearchSettingsGroup) Label() string { return PgTextsearchLabel }

// DisplayLabel renders the group's heading verbatim so it appears as
// "pg_textsearch settings recommendations" rather than "Pg_textsearch ...".
func (sg *PgTextsearchSettingsGroup) DisplayLabel() string { return PgTextsearchLabel }

// Keys should always return PgTextsearchKeys.
func (sg *PgTextsearchSettingsGroup) Keys() []string { return PgTextsearchKeys }

// GetRecommender returns a PgTextsearchRecommender honoring the given profile.
func (sg *PgTextsearchSettingsGroup) GetRecommender(profile Profile) Recommender {
	return NewPgTextsearchRecommender(sg.totalMemory, profile, sg.enabled)
}
