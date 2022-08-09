package pgtune

import (
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

func TestBytesFloatParserParseFloat(t *testing.T) {
	s := "8" + parse.GB
	want := float64(8 * parse.Gigabyte)
	v := &bytesFloatParser{}
	got, err := v.ParseFloat("foo", s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}
}

func TestNumericFloatParserParseFloat(t *testing.T) {
	s := "8.245"
	want := 8.245
	v := &numericFloatParser{}
	got, err := v.ParseFloat("foo", s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}
}

func TestGetFloatParser(t *testing.T) {
	switch x := (GetFloatParser(&MemoryRecommender{})).(type) {
	case *bytesFloatParser:
	default:
		t.Errorf("wrong validator type for MemoryRecommender: got %T", x)
	}

	switch x := (GetFloatParser(&WALRecommender{})).(type) {
	case *WALFloatParser:
	default:
		t.Errorf("wrong validator type for WALRecommender: got %T", x)
	}

	switch x := (GetFloatParser(&PromscaleWALRecommender{})).(type) {
	case *WALFloatParser:
	default:
		t.Errorf("wrong validator type for PromscaleWALRecommender: got %T", x)
	}

	switch x := (GetFloatParser(&ParallelRecommender{})).(type) {
	case *numericFloatParser:
	default:
		t.Errorf("wrong validator type for ParallelRecommender: got %T", x)
	}

	switch x := (GetFloatParser(&PromscaleBgwriterRecommender{})).(type) {
	case *BgwriterFloatParser:
	default:
		t.Errorf("wrong validator type for PromscaleBgwriterRecommender: got %T", x)
	}

	switch x := (GetFloatParser(&MiscRecommender{})).(type) {
	case *numericFloatParser:
	default:
		t.Errorf("wrong validator type for MiscRecommender: got %T", x)
	}

	switch x := (GetFloatParser(&NullRecommender{})).(type) {
	case *numericFloatParser:
	default:
		t.Errorf("wrong validator type for NullRecommender: got %T", x)
	}
}
