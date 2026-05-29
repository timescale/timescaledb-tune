package pgtune

// NullRecommender is a Recommender that returns NoRecommendation for all keys
type NullRecommender struct {
}

// IsAvailable returns whether this Recommender is usable given the system resources. Always true.
func (r *NullRecommender) IsAvailable() bool {
	return true
}

// Recommend returns the recommended PostgreSQL formatted value for the conf
// file for a given key.
func (r *NullRecommender) Recommend(key string) string {
	return NoRecommendation
}
