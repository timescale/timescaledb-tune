// Package pgtune provides the resources and interfaces for getting tuning
// recommendations based on heuristics and knowledge from the online pgtune tool
// for groups of settings in a PostgreSQL conf file.
package pgtune

const osWindows = "windows"

// Recommender is an interface that gives setting recommendations for a given
// key, usually grouped by logical settings groups (e.g. MemoryRecommender for memory settings).
type Recommender interface {
	// IsAvailable returns whether this Recommender is usable given the system resources.
	IsAvailable() bool
	// Recommend returns the recommended PostgreSQL formatted value for the conf file for a given key.
	Recommend(string) string
}

// SettingsGroup is an interface that defines a group of related settings that share
// a Recommender and can be processed together.
type SettingsGroup interface {
	// Label is the canonical name for the group and should be grammatically correct when
	// followed by 'settings', e.g., "memory" -> "memory settings", "parallelism" -> "parallelism settings", etc.
	Label() string
	// Keys are the parameter names/keys as they appear in the PostgreSQL conf file, e.g. "shared_buffers".
	Keys() []string
	// GetRecommender returns the Recommender that should be used for this group of settings.
	GetRecommender() Recommender
}

// GetSettingsGroup returns the corresponding SettingsGroup for a given label, initialized
// according to the system resources of totalMemory and cpus. Panics if unknown label.
func GetSettingsGroup(label string, totalMemory uint64, cpus int) SettingsGroup {
	switch {
	case label == MemoryLabel:
		return &MemorySettingsGroup{totalMemory, cpus}
	case label == ParallelLabel:
		return &ParallelSettingsGroup{cpus}
	case label == WALLabel:
		return &WALSettingsGroup{totalMemory}
	case label == MiscLabel:
		return &MiscSettingsGroup{}
	}
	panic("unknown label: " + label)
}
