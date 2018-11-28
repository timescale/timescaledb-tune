package tstune

import (
	"fmt"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
)

func TestBytesFloatParserParseFloat(t *testing.T) {
	s := "8" + parse.GB
	want := float64(8 * parse.Gigabyte)
	v := &bytesFloatParser{}
	got, err := v.ParseFloat(s)
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
	got, err := v.ParseFloat(s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("incorrect result: got %f want %f", got, want)
	}
}

const testKey = "test_setting"

var testRegex = keyToRegex(testKey)

func TestKeyToRegex(t *testing.T) {
	regex := keyToRegex("foo")
	want := fmt.Sprintf(tuneRegexFmt, "foo")
	if got := regex.String(); got != want {
		t.Errorf("incorrect regex: got %s want %s", got, want)
	}
}

func TestParseWithRegex(t *testing.T) {
	cases := []struct {
		desc  string
		input string
		want  *tunableParseResult
	}{
		{
			desc:  "simple correct",
			input: testKey + " = 50.0",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra:     "",
			},
		},
		{
			desc:  "correct, comment at end",
			input: testKey + " = 50.0 # do not change!",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra:     " # do not change!",
			},
		},
		{
			desc:  "correct, comment at end no space",
			input: testKey + " = 50.0# do not change!",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra:     "# do not change!",
			},
		},
		{
			desc:  "correct, comment at end more space",
			input: testKey + " = 50.0    # do not change!",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra:     "    # do not change!",
			},
		},
		{
			desc: "correct, comment at end tabs",
			input: testKey + " = 50.0	# do not change!",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra: "	# do not change!",
			},
		},
		{
			desc: "correct, tabs at the end",
			input: testKey + " = 50.0			",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra: "			",
			},
		},
		{
			desc:  "simple correct, commented",
			input: "#" + testKey + " = 50.0",
			want: &tunableParseResult{
				commented: true,
				key:       testKey,
				value:     "50.0",
				extra:     "",
			},
		},
		{
			desc: "commented with spaces",
			input: "	#	" + testKey + " = 50.0",
			want: &tunableParseResult{
				commented: true,
				key:       testKey,
				value:     "50.0",
				extra:     "",
			},
		},
		{
			desc: "commented with ending comment",
			input: "#	" + testKey + " = 50.0	# do not change",
			want: &tunableParseResult{
				commented: true,
				key:       testKey,
				value:     "50.0",
				extra: "	# do not change",
			},
		},
		{
			desc:  "incorrect, do not accept comments with starting #",
			input: testKey + " = 50.0 do not change!",
			want:  nil,
		},
	}

	for _, c := range cases {
		res := parseWithRegex(c.input, testRegex)
		if res == nil && c.want != nil {
			t.Errorf("%s: result was unexpectedly nil: want %v", c.desc, c.want)
		} else if res != nil && c.want == nil {
			t.Errorf("%s: result was unexpectedly non-nil: got %v", c.desc, res)
		} else if c.want != nil {
			if got := res.commented; got != c.want.commented {
				t.Errorf("%s: incorrect commented: got %v want %v", c.desc, got, c.want.commented)
			}
			if got := res.key; got != c.want.key {
				t.Errorf("%s: incorrect key: got %v want %v", c.desc, got, c.want.key)
			}
			if got := res.value; got != c.want.value {
				t.Errorf("%s: incorrect value: got %s want %s", c.desc, got, c.want.value)
			}
			if got := res.extra; got != c.want.extra {
				t.Errorf("%s: incorrect extra: got %s want %s", c.desc, got, c.want.extra)
			}
		}
	}
}
