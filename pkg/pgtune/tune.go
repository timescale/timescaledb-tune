// Package pgtune provides the resources and interfaces for getting tuning
// recommendations based on heuristics and knowledge from the online pgtune tool
// for groups of settings in a PostgreSQL conf file.
package pgtune

import "fmt"

const (
	osWindows                = "windows"
	errMaxConnsTooLowFmt     = "maxConns must be 0 OR >= %d: got %d"
	errMaxBGWorkersTooLowFmt = "maxBGWorkers must be >= %d: got %d"
)

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

// SystemConfig represents a system's resource configuration, to be used when generating
// recommendations for different SettingsGroups.
type SystemConfig struct {
	Memory         uint64
	MilliCPUs      int
	PGMajorVersion string
	WALDiskSize    uint64
	maxConns       uint64
	MaxBGWorkers   int
}

// NewSystemConfig returns a new SystemConfig with the given parameters.
func NewSystemConfig(totalMemory uint64, milliCPUS int, pgVersion string, walDiskSize uint64, maxConns uint64, maxBGWorkers int) (*SystemConfig, error) {
	if maxConns != 0 && maxConns < minMaxConns {
		return nil, fmt.Errorf(errMaxConnsTooLowFmt, minMaxConns, maxConns)
	}
	if maxBGWorkers < MaxBackgroundWorkersDefault {
		return nil, fmt.Errorf(errMaxBGWorkersTooLowFmt, MaxBackgroundWorkersDefault, maxBGWorkers)
	}
	return &SystemConfig{
		Memory:         totalMemory,
		MilliCPUs:      milliCPUS,
		PGMajorVersion: pgVersion,
		WALDiskSize:    walDiskSize,
		maxConns:       maxConns,
		MaxBGWorkers:   maxBGWorkers,
	}, nil
}

// GetSettingsGroup returns the corresponding SettingsGroup for a given label, initialized
// according to the system resources of totalMemory and cpus. Panics if unknown label.
func GetSettingsGroup(label string, config *SystemConfig) SettingsGroup {
	switch {
	case label == MemoryLabel:
		return &MemorySettingsGroup{config.Memory, config.MilliCPUs, config.maxConns}
	case label == ParallelLabel:
		return &ParallelSettingsGroup{config.PGMajorVersion, config.MilliCPUs, config.MaxBGWorkers}
	case label == WALLabel:
		return &WALSettingsGroup{config.Memory, config.WALDiskSize}
	case label == MiscLabel:
		return &MiscSettingsGroup{config.Memory, config.maxConns, config.PGMajorVersion}
	}
	panic("unknown label: " + label)
}
