package main

import "fmt"

const (
	checkpointKey     = "checkpoint_completion_target"
	statsTargetKey    = "default_statistics_target"
	maxConnectionsKey = "max_connections"
	randomPageCostKey = "random_page_cost"
	// linux only
	effectiveIOKey = "effective_io_concurrency"

	checkpointDefault     = "0.9"
	statsTargetDefault    = "500"
	maxConnectionsDefault = "20"
	randomPageCostDefault = "1.1"
	effectiveIODefault    = "200"
)

var otherKeys = []string{
	statsTargetKey,
	randomPageCostKey,
	effectiveIOKey,
	checkpointKey,
	maxConnectionsKey,
}

type miscRecommender struct{}

func (r *miscRecommender) Recommend(key string) string {
	var val string
	if key == checkpointKey {
		val = checkpointDefault
	} else if key == statsTargetKey {
		val = statsTargetDefault
	} else if key == maxConnectionsKey {
		val = maxConnectionsDefault
	} else if key == randomPageCostKey {
		val = randomPageCostDefault
	} else if key == effectiveIOKey {
		val = effectiveIODefault
	} else {
		panic(fmt.Sprintf("unknown key: %s", key))
	}
	return val
}
