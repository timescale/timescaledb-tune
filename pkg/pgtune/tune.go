// Package pgtune provides the resources and interfaces for getting tuning
// recommendations based on heuristics and knowledge from the online pgtune tool
// for groups of settings in a PostgreSQL conf file.
package pgtune

import (
	"fmt"
	"strings"
)

const (
	osWindows                = "windows"
	errMaxConnsTooLowFmt     = "maxConns must be 0 OR >= %d: got %d"
	errMaxBGWorkersTooLowFmt = "maxBGWorkers must be >= %d: got %d"
	errUnrecognizedProfile   = "unrecognized profile: %s"
)

// Profile is a specific "mode" in which timescaledb-tune can be run to provide recommendations tailored to a
// special workload type, e.g. "promscale"
type Profile int64

const (
	DefaultProfile Profile = iota
	PromscaleProfile
)

func ParseProfile(s string) (Profile, error) {
	switch strings.ToLower(s) {
	case "":
		return DefaultProfile, nil
	case "promscale":
		return PromscaleProfile, nil
	default:
		return DefaultProfile, fmt.Errorf(errUnrecognizedProfile, s)
	}
}

func (p Profile) String() string {
	switch p {
	case DefaultProfile:
		return ""
	case PromscaleProfile:
		return "promscale"
	default:
		return "unrecognized"
	}
}

const NoRecommendation = ""

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
	GetRecommender(Profile) Recommender
}

// SystemConfig represents a system's resource configuration, to be used when generating
// recommendations for different SettingsGroups.
type SystemConfig struct {
	Memory         uint64
	CPUs           int
	PGMajorVersion string
	WALDiskSize    uint64
	maxConns       uint64
	MaxBGWorkers   int
}

// NewSystemConfig returns a new SystemConfig with the given parameters.
func NewSystemConfig(totalMemory uint64, cpus int, pgVersion string, walDiskSize uint64, maxConns uint64, maxBGWorkers int) (*SystemConfig, error) {
	if maxConns != 0 && maxConns < minMaxConns {
		return nil, fmt.Errorf(errMaxConnsTooLowFmt, minMaxConns, maxConns)
	}
	if maxBGWorkers < MaxBackgroundWorkersDefault {
		return nil, fmt.Errorf(errMaxBGWorkersTooLowFmt, MaxBackgroundWorkersDefault, maxBGWorkers)
	}
	return &SystemConfig{
		Memory:         totalMemory,
		CPUs:           cpus,
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
		return &MemorySettingsGroup{config.Memory, config.CPUs, config.maxConns}
	case label == ParallelLabel:
		return &ParallelSettingsGroup{config.PGMajorVersion, config.CPUs, config.MaxBGWorkers}
	case label == WALLabel:
		return &WALSettingsGroup{config.Memory, config.WALDiskSize}
	case label == BgwriterLabel:
		return &BgwriterSettingsGroup{}
	case label == MiscLabel:
		return &MiscSettingsGroup{config.Memory, config.maxConns, config.PGMajorVersion}
	}
	panic("unknown label: " + label)
}
