package pgtune

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

const (
	errUnrecognizedBoolValue = "unrecognized bool value: %s"
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

type boolFloatParser struct{}

func (v *boolFloatParser) ParseFloat(key string, s string) (float64, error) {
	s = strings.ToLower(s)
	s = strings.TrimLeft(s, `"'`)
	s = strings.TrimRight(s, `"'`)
	switch s {
	case "on":
		return 1.0, nil
	case "off":
		return 0.0, nil
	case "true":
		return 1.0, nil
	case "false":
		return 0.0, nil
	case "yes":
		return 1.0, nil
	case "no":
		return 0.0, nil
	case "1":
		return 1.0, nil
	case "0":
		return 0.0, nil
	default:
		return 0.0, fmt.Errorf(errUnrecognizedBoolValue, s)
	}
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
		return &numericFloatParser{}
	case *ParallelRecommender:
		return &numericFloatParser{}
	default:
		return &numericFloatParser{}
	}
}
