package tstune

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

func stringSliceToBytesReader(lines []string) *bytes.Buffer {
	return bytes.NewBufferString(strings.Join(lines, "\n"))
}

func TestFileExists(t *testing.T) {
	existsName := "exists.txt"
	errorName := "error.txt"
	cases := []struct {
		desc     string
		filename string
		want     bool
	}{
		{
			desc:     "found file",
			filename: existsName,
			want:     true,
		},
		{
			desc:     "not found file",
			filename: "ghost.txt",
			want:     false,
		},
		{
			desc:     "error in stat",
			filename: errorName,
			want:     false,
		},
	}

	oldOSStatFn := osStatFn
	osStatFn = func(name string) (os.FileInfo, error) {
		if name == existsName {
			return nil, nil
		} else if name == errorName {
			return nil, fmt.Errorf("this is an error")
		} else {
			return nil, os.ErrNotExist
		}
	}

	for _, c := range cases {
		if got := fileExists(c.filename); got != c.want {
			t.Errorf("%s: incorrect result: got %v want %v", c.desc, got, c.want)
		}
	}

	osStatFn = oldOSStatFn
}

func TestRemoveDuplicatesProcessor(t *testing.T) {
	lines := []*configLine{
		{content: "foo = 'bar'"},
		{content: "foo = 'baz'"},
		{content: "foo = 'quaz'"},
	}
	p := &removeDuplicatesProcessor{regex: keyToRegexQuoted("foo")}
	p.Process(lines[0])
	if lines[0].remove {
		t.Errorf("first instance incorrectly marked for remove")
	}

	check := func(idx int) {
		err := p.Process(lines[idx])
		if err != nil {
			t.Errorf("unexpected error on test %d: %v", idx, err)
		}
		if !lines[idx-1].remove {
			t.Errorf("configLine not marked to remove on test %d", idx)
		}
		if lines[idx].remove {
			t.Errorf("configLine incorrectly marked to remove on test %d", idx)
		}
	}

	check(1)
	check(2)
}

func TestGetRemoveDuplicatesProcessors(t *testing.T) {
	cases := []struct {
		desc string
		keys []string
	}{
		{
			desc: "no keys",
			keys: []string{},
		},
		{
			desc: "one key",
			keys: []string{"foo"},
		},
		{
			desc: "two keys",
			keys: []string{"foo", "bar"},
		},
	}

	for _, c := range cases {
		procs := getRemoveDupeProcessors(c.keys)
		if got := len(procs); got != len(c.keys) {
			t.Errorf("%s: incorrect length: got %d want %d", c.desc, got, len(c.keys))
		} else {
			for i, key := range c.keys {
				rdp := procs[i].(*removeDuplicatesProcessor)
				want := keyToRegexQuoted(key).String()
				if got := rdp.regex.String(); got != want {
					t.Errorf("%s: incorrect proc at %d: got %s want %s", c.desc, i, got, want)
				}
			}
		}
	}
}

func TestGetConfigFilePath(t *testing.T) {
	cases := []struct {
		desc      string
		os        string
		pgVersion string
		files     []string
		wantFile  string
		shouldErr bool
	}{
		{
			desc:      "mac - yes",
			os:        osMac,
			files:     []string{fileNameMac},
			wantFile:  fileNameMac,
			shouldErr: false,
		},
		{
			desc:      "mac - no",
			os:        osMac,
			files:     []string{"/etc"},
			wantFile:  "",
			shouldErr: true,
		},
		{
			desc:      "linux - pg10+debian",
			os:        osLinux,
			pgVersion: pgMajor10,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "10")},
			wantFile:  fmt.Sprintf(fileNameDebianFmt, "10"),
			shouldErr: false,
		},
		{
			desc:      "linux - pg9.6+debian",
			os:        osLinux,
			pgVersion: pgMajor96,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "9.6")},
			wantFile:  fmt.Sprintf(fileNameDebianFmt, "9.6"),
			shouldErr: false,
		},
		{
			desc:      "linux - mismatch+debian",
			os:        osLinux,
			pgVersion: pgMajor96,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "10")},
			wantFile:  "",
			shouldErr: true,
		},
		{
			desc:      "linux - pg10+rpm",
			os:        osLinux,
			pgVersion: pgMajor10,
			files:     []string{fmt.Sprintf(fileNameRPMFmt, "10")},
			wantFile:  fmt.Sprintf(fileNameRPMFmt, "10"),
			shouldErr: false,
		},
		{
			desc:      "linux - pg9.6+rpm",
			os:        osLinux,
			pgVersion: pgMajor96,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "9.6")},
			wantFile:  fmt.Sprintf(fileNameDebianFmt, "9.6"),
			shouldErr: false,
		},

		{
			desc:      "linux - mismatch+rpm",
			os:        osLinux,
			pgVersion: pgMajor96,
			files:     []string{fmt.Sprintf(fileNameRPMFmt, "10")},
			wantFile:  "",
			shouldErr: true,
		},
		{
			desc:      "linux - arch",
			os:        osLinux,
			files:     []string{fileNameArch},
			wantFile:  fileNameArch,
			shouldErr: false,
		},

		{
			desc:      "linux - no",
			os:        osLinux,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "9.0")},
			wantFile:  "",
			shouldErr: true,
		},
	}

	oldOSStatFn := osStatFn
	for _, c := range cases {
		osStatFn = func(fn string) (os.FileInfo, error) {
			for _, s := range c.files {
				if fn == s {
					return nil, nil
				}
			}
			return nil, os.ErrNotExist
		}
		filename, err := getConfigFilePath(c.os, c.pgVersion)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		if c.shouldErr && filename != "" {
			t.Errorf("%s: unexpected filename in error case: got %s", c.desc, filename)
		}

		if got := filename; got != c.wantFile {
			t.Errorf("%s: incorrect filename: got %s want %s", c.desc, got, c.wantFile)
		}
	}
	osStatFn = oldOSStatFn
}

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
				lines:            []*configLine{},
				tuneParseResults: make(map[string]*tunableParseResult),
				sharedLibResult:  nil,
			},
		},
		{
			desc:  "single irrelevant line",
			lines: []string{"foo"},
			want: &configFileState{
				lines:            []*configLine{{content: "foo"}},
				tuneParseResults: make(map[string]*tunableParseResult),
				sharedLibResult:  nil,
			},
		},
		{
			desc:  "shared lib line only",
			lines: []string{sharedLibLine},
			want: &configFileState{
				lines:            []*configLine{{content: sharedLibLine}},
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
				lines: []*configLine{
					{content: "foo"},
					{content: sharedLibLine},
					{content: "bar"},
					{content: memoryLine},
					{content: walLine},
					{content: "baz"},
				},
				tuneParseResults: map[string]*tunableParseResult{
					pgtune.SharedBuffersKey: {
						idx:       3,
						commented: true,
						key:       pgtune.SharedBuffersKey,
						value:     "64MB",
						extra:     "",
					},
					pgtune.MinWALKey: {
						idx:       4,
						commented: false,
						key:       pgtune.MinWALKey,
						value:     "0GB",
						extra:     " # weird",
					},
				},
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
				if want := c.want.lines[i].content; got.content != want {
					t.Errorf("%s: incorrect line at %d: got\n%s\nwant\n%s", c.desc, i, got.content, want)
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

type errReader struct {
	count uint64
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.count > 1 {
		return 0, fmt.Errorf("erroring")
	}
	p[len(p)-1] = '\n'
	r.count++
	return 1, nil
}

func TestGetConfigFileStateErr(t *testing.T) {
	r := &errReader{}
	cfs, err := getConfigFileState(r)
	if cfs != nil {
		t.Errorf("cfs not nil: %v", cfs)
	}
	if err == nil {
		t.Errorf("err is nil")
	}
}

const errProcess = "process error"

type countProcessor struct {
	count     int
	shouldErr bool
}

func (p *countProcessor) Process(_ *configLine) error {
	if p.shouldErr {
		return fmt.Errorf(errProcess)
	}
	p.count++
	return nil
}

func TestConfigFileStateProcessLines(t *testing.T) {
	countProc1 := &countProcessor{}
	countProc2 := &countProcessor{}
	procs := []configLineProcessor{countProc1, countProc2}
	lines := []string{"foo", "bar", "baz"}
	wantCount := len(lines)
	r := stringSliceToBytesReader(lines)
	cfs, err := getConfigFileState(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = cfs.ProcessLines(procs...)
	if err != nil {
		t.Errorf("unexpected error in processing: %v", err)
	}
	if got := countProc1.count; got != wantCount {
		t.Errorf("incorrect count for countProc1: got %d want %d", got, wantCount)
	}
	if got := countProc2.count; got != wantCount {
		t.Errorf("incorrect count for countProc2: got %d want %d", got, wantCount)
	}

	badCountProc := &countProcessor{shouldErr: true}
	procs = append(procs, badCountProc)
	err = cfs.ProcessLines(procs...)
	if err == nil {
		t.Errorf("unexpected lack of error")
	}
	if got := err.Error(); got != errProcess {
		t.Errorf("unexpected error: got %s want %s", got, errProcess)
	}
}

const (
	errTestTruncate = "truncate error"
	errTestSeek     = "seek error"
)

type testTruncateWriter struct {
	*testWriter
	seekErr     bool
	truncateErr bool
}

func (w *testTruncateWriter) Seek(_ int64, _ int) (int64, error) {
	if w.seekErr {
		return 0, fmt.Errorf(errTestSeek)
	}
	return 0, nil
}

func (w *testTruncateWriter) Truncate(_ int64) error {
	if w.truncateErr {
		return fmt.Errorf(errTestTruncate)
	}
	return nil
}

func TestConfigFileStateWriteTo(t *testing.T) {
	cases := []struct {
		desc      string
		lines     []string
		removeIdx int
		errMsg    string
		w         io.Writer
	}{
		{
			desc:      "empty",
			lines:     []string{},
			removeIdx: -1,
			w:         &testWriter{false, []string{}},
		},
		{
			desc:      "one line",
			lines:     []string{"foo"},
			removeIdx: -1,
			w:         &testWriter{false, []string{}},
		},
		{
			desc:      "many lines",
			lines:     []string{"foo", "bar", "baz", "quaz"},
			removeIdx: -1,
			w:         &testWriter{false, []string{}},
		},
		{
			desc:      "many lines w/ truncating",
			lines:     []string{"foo", "bar", "baz", "quaz"},
			removeIdx: -1,
			w:         &testTruncateWriter{&testWriter{false, []string{}}, false, false},
		},
		{
			desc:      "many lines, remove middle line",
			lines:     []string{"foo", "bar", "baz"},
			removeIdx: 1,
			w:         &testWriter{false, []string{}},
		},
		{
			desc:      "error in truncate",
			lines:     []string{"foo"},
			removeIdx: -1,
			errMsg:    errTestTruncate,
			w:         &testTruncateWriter{&testWriter{true, []string{}}, false, true},
		},
		{
			desc:      "error in seek",
			lines:     []string{"foo"},
			removeIdx: -1,
			errMsg:    errTestSeek,
			w:         &testTruncateWriter{&testWriter{true, []string{}}, true, false},
		},
		{
			desc:      "error in write w/o truncating",
			lines:     []string{"foo"},
			removeIdx: -1,
			errMsg:    errTestWriter,
			w:         &testWriter{true, []string{}},
		},
		{
			desc:      "error in write w/ truncating",
			lines:     []string{"foo"},
			removeIdx: -1,
			errMsg:    errTestWriter,
			w:         &testTruncateWriter{&testWriter{true, []string{}}, false, false},
		},
	}

	for _, c := range cases {
		r := stringSliceToBytesReader(c.lines)
		cfs, err := getConfigFileState(r)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.desc, err)
		}

		if c.removeIdx >= 0 {
			cfs.lines[c.removeIdx].remove = true
		}

		_, err = cfs.WriteTo(c.w)
		if c.errMsg == "" && err != nil {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if c.errMsg != "" {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: unexpected type of error: %v", c.desc, err)
			}
		}

		var w *testWriter
		switch temp := c.w.(type) {
		case *testWriter:
			w = temp
		case *testTruncateWriter:
			w = temp.testWriter
		}

		lineCntModifier := 0
		if c.removeIdx >= 0 {
			lineCntModifier = 1
		}

		if len(c.lines) > 0 && c.errMsg == "" {
			if got := len(w.lines); got != len(c.lines)-lineCntModifier {
				t.Errorf("%s: incorrect output len: got %d want %d", c.desc, got, len(c.lines)-lineCntModifier)
			}
			idxModifier := 0
			for i, want := range c.lines {
				if i == c.removeIdx {
					idxModifier = 1
					continue
				}
				if got := w.lines[i-idxModifier]; got != want+"\n" {
					t.Errorf("%s: incorrect line at %d: got %s want %s", c.desc, i, got, want+"\n")
				}
			}
		}
	}
}
