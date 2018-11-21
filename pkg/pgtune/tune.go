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
