package main

import (
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

const testKey = "test_setting"

var testRegex = keyToRegex(testKey)

func TestIsIn(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		arr  []string
		want bool
	}{
		{
			desc: "yes, len 1",
			key:  "foo",
			arr:  []string{"foo"},
			want: true,
		},
		{
			desc: "no, len 0",
			key:  "foo",
			arr:  []string{},
			want: false,
		},
		{
			desc: "no, len 1",
			key:  "bar",
			arr:  []string{"foo"},
			want: false,
		},
		{
			desc: "no, len 3",
			key:  "bar",
			arr:  []string{"foo1", "foo2", "foo3"},
			want: false,
		},
		{
			desc: "yes, len 3",
			key:  "foo2",
			arr:  []string{"foo1", "foo2", "foo3"},
			want: true,
		},
	}

	for _, c := range cases {
		if got := isIn(c.key, c.arr); got != c.want {
			t.Errorf("%s: incorrect value: got %v want %v", c.desc, got, c.want)
		}
	}
}

func TestKeyToParseFn(t *testing.T) {
	cases := []struct {
		desc       string
		key        string
		parseInput string
		want       float64
	}{
		{
			desc:       "memory key",
			key:        pgtune.MemoryKeys[0],
			parseInput: "10" + parse.GB,
			want:       float64(10 * parse.Gigabyte),
		},
		{
			desc:       "wal key",
			key:        pgtune.WALKeys[0],
			parseInput: "5" + parse.MB,
			want:       float64(5 * parse.Megabyte),
		},
		{
			desc:       "other key",
			key:        pgtune.MiscKeys[0],
			parseInput: "501.0",
			want:       501.0,
		},
	}

	for _, c := range cases {
		got, err := keyToParseFn(c.key)(c.parseInput)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if got != c.want {
			t.Errorf("%s: incorrect result: got %v want %v", c.desc, got, c.want)
		}
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
