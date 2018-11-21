package pgtune

const osWindows = "windows"

// Recommender is an interface that gives setting recommendations for a given
// key, usually grouped by logical settings groups (e.g. MemoryRecommender for memory settings).
type Recommender interface {
	Recommend(string) string
}
