package tstune

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

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
				tuneParseResults: map[string]*tunableParseResult{
					pgtune.SharedBuffersKey: &tunableParseResult{
						idx:       3,
						commented: true,
						key:       pgtune.SharedBuffersKey,
						value:     "64MB",
						extra:     "",
					},
					pgtune.MinWALKey: &tunableParseResult{
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
		} else if c.shouldErr && err.Error() != errTestWriter {
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
