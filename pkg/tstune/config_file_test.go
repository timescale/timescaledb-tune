package tstune

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestGetConfigFileState(t *testing.T) {
	sharedLibLine := "shared_preload_libraries = 'timescaledb' # comment"
	memoryLine := "#shared_buffers = 64MB"
	walLine := "min_wal_size = 0GB # weird"
	cases := []struct {
		desc  string
		lines []string
		want  *configFileState
	}{
		{
			desc:  "empty file",
			lines: []string{},
			want: &configFileState{
				lines:            []string{},
				tuneParseResults: make(map[string]*tunableParseResult),
				sharedLibResult:  nil,
			},
		},
		{
			desc:  "single irrelevant line",
			lines: []string{"foo"},
			want: &configFileState{
				lines:            []string{"foo"},
				tuneParseResults: make(map[string]*tunableParseResult),
				sharedLibResult:  nil,
			},
		},
		{
			desc:  "shared lib line only",
			lines: []string{sharedLibLine},
			want: &configFileState{
				lines:            []string{sharedLibLine},
				tuneParseResults: make(map[string]*tunableParseResult),
				sharedLibResult: &sharedLibResult{
					idx:          0,
					commented:    false,
					hasTimescale: true,
					commentGroup: "",
					libs:         "timescaledb",
				},
			},
		},
		{
			desc:  "multi-line",
			lines: []string{"foo", sharedLibLine, "bar", memoryLine, walLine, "baz"},
			want: &configFileState{
				lines: []string{"foo", sharedLibLine, "bar", memoryLine, walLine, "baz"},
				/*tuneParseResults: map[string]*tunableParseResult{
					sharedBuffersKey: &tunableParseResult{
						idx:       3,
						commented: true,
						key:       sharedBuffersKey,
						value:     "64MB",
						extra:     "",
					},
					minWalKey: &tunableParseResult{
						idx:       4,
						commented: false,
						key:       minWalKey,
						value:     "0GB",
						extra:     " # weird",
					},
				},*/
				sharedLibResult: &sharedLibResult{
					idx:          1,
					commented:    false,
					hasTimescale: true,
					commentGroup: "",
					libs:         "timescaledb",
				},
			},
		},
	}

	for _, c := range cases {
		buf := bytes.NewBufferString(strings.Join(c.lines, "\n"))
		cfs, _ := getConfigFileState(buf)
		if got := len(cfs.lines); got != len(c.want.lines) {
			t.Errorf("%s: incorrect number of cfs lines: got %d want %d", c.desc, got, len(c.want.lines))
		} else {
			for i, got := range cfs.lines {
				if want := c.want.lines[i]; got != want {
					t.Errorf("%s: incorrect line at %d: got\n%s\nwant\n%s", c.desc, i, got, want)
				}
			}
		}

		if c.want.sharedLibResult != nil {
			if cfs.sharedLibResult == nil {
				t.Errorf("%s: unexpected nil shared lib result", c.desc)
			} else {
				want := fmt.Sprintf("%v", c.want.sharedLibResult)
				if got := fmt.Sprintf("%v", cfs.sharedLibResult); got != want {
					t.Errorf("%s: incorrect sharedLibResult: got %s want %s", c.desc, got, want)
				}
			}
		}

		if len(c.want.tuneParseResults) > 0 {
			if got := len(cfs.tuneParseResults); got != len(c.want.tuneParseResults) {
				t.Errorf("%s: incorrect tuneParseResults size: got %d want %d", c.desc, got, len(c.want.tuneParseResults))
			} else {
				for k, v := range c.want.tuneParseResults {
					want := fmt.Sprintf("%v", v)
					if got, ok := cfs.tuneParseResults[k]; fmt.Sprintf("%v", got) != want || !ok {
						t.Errorf("%s: incorrect tuneParseResults for %s: got %s want %s", c.desc, k, fmt.Sprintf("%v", got), want)
					}
				}
			}
		}
	}
}

var errDefault = fmt.Errorf("erroring")

type testWriter struct {
	shouldErr bool
	lines     []string
}

func (w *testWriter) Write(buf []byte) (int, error) {
	if w.shouldErr {
		return 0, errDefault
	}
	w.lines = append(w.lines, string(buf))
	return 0, nil
}

func TestConfigFileStateWriteTo(t *testing.T) {
	cases := []struct {
		desc      string
		lines     []string
		shouldErr bool
	}{
		{
			desc:      "empty",
			lines:     []string{},
			shouldErr: false,
		},
		{
			desc:      "one line",
			lines:     []string{"foo"},
			shouldErr: false,
		},
		{
			desc:      "many lines",
			lines:     []string{"foo", "bar", "baz", "quaz"},
			shouldErr: false,
		},
		{
			desc:      "error",
			lines:     []string{"foo"},
			shouldErr: true,
		},
	}

	for _, c := range cases {
		cfs := &configFileState{lines: c.lines}
		w := &testWriter{c.shouldErr, []string{}}
		_, err := cfs.WriteTo(w)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		} else if c.shouldErr && err != errDefault {
			t.Errorf("%s: unexpected type of error: %v", c.desc, err)
		}

		if len(c.lines) > 0 && !c.shouldErr {
			if got := len(w.lines); got != len(c.lines) {
				t.Errorf("%s: incorrect output len: got %d want %d", c.desc, got, len(c.lines))
			}
			for i, want := range c.lines {
				if got := w.lines[i]; got != want+"\n" {
					t.Errorf("%s: incorrect line at %d: got %s want %s", c.desc, i, got, want+"\n")
				}
			}
		}
	}
}
