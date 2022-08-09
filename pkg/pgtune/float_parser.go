package pgtune

import (
	"strconv"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

type FloatParser interface {
	ParseFloat(string, string) (float64, error)
}

type bytesFloatParser struct{}

func (v *bytesFloatParser) ParseFloat(key string, s string) (float64, error) {
	temp, err := parse.PGFormatToBytes(s)
	return float64(temp), err
}

type numericFloatParser struct{}

func (v *numericFloatParser) ParseFloat(key string, s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// GetFloatParser returns the correct FloatParser for a given Recommender.
func GetFloatParser(r Recommender) FloatParser {
	switch r.(type) {
	case *MemoryRecommender:
		return &bytesFloatParser{}
	case *WALRecommender:
		return &WALFloatParser{}
	case *PromscaleWALRecommender:
		return &WALFloatParser{}
	case *PromscaleBgwriterRecommender:
		return &BgwriterFloatParser{}
	case *ParallelRecommender:
		return &numericFloatParser{}
	default:
		return &numericFloatParser{}
	}
}
