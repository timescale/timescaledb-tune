package tstune

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

// To make the test less flaky, we 0 out the seconds to make the comparison
// more likely to succeed.
func removeSecsFromLastTuned(time string) string {
	runes := []rune(time)
	start := len(lastTunedParam + " = '")
	runes[start+17] = '0'
	runes[start+18] = '0'
	return string(runes)
}

func TestOurParamToValue(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	want := removeSecsFromLastTuned(fmt.Sprintf(fmtOurParam, lastTunedParam, now))
	got := removeSecsFromLastTuned(ourParamString(lastTunedParam))
	if got != want {
		t.Errorf("incorrect value for %s: got %s want %s", lastTunedParam, got, want)
	}

	want = fmt.Sprintf(fmtOurParam, lastTunedVersionParam, Version)
	got = ourParamString(lastTunedVersionParam)
	if got != want {
		t.Errorf("incorrect value for %s: got %s want %s", lastTunedVersionParam, got, want)
	}

	defer func() {
		if re := recover(); re == nil {
			t.Errorf("did not panic when should")
		}
	}()
	_ = ourParamString("not_a_real_param")
}

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

func TestGetFloatParser(t *testing.T) {
	switch x := (getFloatParser(&pgtune.MemoryRecommender{})).(type) {
	case *bytesFloatParser:
	default:
		t.Errorf("wrong validator type for MemoryRecommender: got %T", x)
	}

	switch x := (getFloatParser(&pgtune.WALRecommender{})).(type) {
	case *bytesFloatParser:
	default:
		t.Errorf("wrong validator type for WALRecommender: got %T", x)
	}

	switch x := (getFloatParser(&pgtune.ParallelRecommender{})).(type) {
	case *numericFloatParser:
	default:
		t.Errorf("wrong validator type for ParallelRecommender: got %T", x)
	}

	switch x := (getFloatParser(&pgtune.MiscRecommender{})).(type) {
	case *numericFloatParser:
	default:
		t.Errorf("wrong validator type for MiscRecommender: got %T", x)
	}
}

const (
	testKey            = "test_setting"
	testKeyMeta        = "test.setting"
	testKeyMetaCorrect = "test\\.setting"
)

func TestKeyToRegex(t *testing.T) {
	regex := keyToRegex(testKey)
	want := fmt.Sprintf(tuneRegexFmt, testKey)
	if got := regex.String(); got != want {
		t.Errorf("incorrect regex: got %s want %s", got, want)
	}

	regex = keyToRegex(testKeyMeta)
	want = fmt.Sprintf(tuneRegexFmt, testKeyMetaCorrect)
	if got := regex.String(); got != want {
		t.Errorf("incorrect regex (meta symbols): got %s want %s", got, want)
	}
}

func TestKeyToRegexQuoted(t *testing.T) {
	regex := keyToRegexQuoted(testKey)
	want := fmt.Sprintf(tuneRegexQuotedFmt, testKey)
	if got := regex.String(); got != want {
		t.Errorf("incorrect regex: got %s want %s", got, want)
	}

	regex = keyToRegexQuoted(testKeyMeta)
	want = fmt.Sprintf(tuneRegexQuotedFmt, testKeyMetaCorrect)
	if got := regex.String(); got != want {
		t.Errorf("incorrect regex (meta symbols): got %s want %s", got, want)
	}
}

var testRegex = keyToRegex(testKey)

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
			desc:  "correct, no whitespace surrounding =",
			input: testKey + "=50.0",
			want: &tunableParseResult{
				commented: false,
				key:       testKey,
				value:     "50.0",
				extra:     "",
			},
		},
		{
			desc:  "correct, much whitespace surrounding =",
			input: testKey + "    =      50.0",
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

func TestParseWithRegexPanic(t *testing.T) {
	// Don't use regexp.QuoteMeta so that we can sneak meta chars into and cause
	// an extra capture group, (bar)+
	badRegex := regexp.MustCompile(fmt.Sprintf(tuneRegexFmt, "foo(bar)+"))
	line := "#foobar = 5 #commented"

	defer func() {
		if re := recover(); re == nil {
			t.Errorf("did not panic when should")
		}
	}()
	parseWithRegex(line, badRegex)
}
