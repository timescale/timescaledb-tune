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

func Test_boolFloatParser_ParseFloat(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    float64
		wantErr bool
	}{
		{
			name:    "on",
			arg:     "on",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "oN",
			arg:     "oN",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "'ON'",
			arg:     "'ON'",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "off",
			arg:     "off",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "OfF",
			arg:     "OfF",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "'OFF'",
			arg:     "'OFF'",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "true",
			arg:     "true",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "false",
			arg:     "false",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "yes",
			arg:     "yes",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "no",
			arg:     "no",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "1",
			arg:     "1",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "0",
			arg:     "0",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "bob",
			arg:     "bob",
			want:    0.0,
			wantErr: true,
		},
		{
			name:    "99",
			arg:     "99",
			want:    0.0,
			wantErr: true,
		},
		{
			name:    "0.1",
			arg:     "0.1",
			want:    0.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &boolFloatParser{}
			got, err := v.ParseFloat("", tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFloat() got = %v, want %v", got, tt.want)
			}
		})
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
	case *numericFloatParser:
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
