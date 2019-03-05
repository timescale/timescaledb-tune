package tstune

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

type testBufferCloser struct {
	b         bytes.Buffer
	shouldErr bool
}

func (b *testBufferCloser) Write(p []byte) (int, error) {
	if b.shouldErr {
		return 0, fmt.Errorf(errTestWriter)
	}
	return b.b.Write(p)
}

func (b *testBufferCloser) Close() error { return nil }

func TestBackup(t *testing.T) {
	oldOSCreateFn := osCreateFn
	now := time.Now()
	lines := []string{"foo", "bar", "baz", "quaz"}
	r := stringSliceToBytesReader(lines)
	cfs, err := getConfigFileState(r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	wantFileName := backupFilePrefix + now.Format(backupDateFmt)
	wantPath := path.Join(os.TempDir(), wantFileName)

	osCreateFn = func(_ string) (io.WriteCloser, error) {
		return nil, fmt.Errorf("erroring")
	}

	path, err := backup(cfs)
	if path != wantPath {
		t.Errorf("incorrect path in error case: got\n%s\nwant\n%s", path, wantPath)
	}
	if err == nil {
		t.Errorf("unexpected lack of error for bad create")
	}
	want := fmt.Sprintf(errBackupNotCreatedFmt, wantPath, "erroring")
	if got := err.Error(); got != want {
		t.Errorf("incorrect error: got\n%s\nwant\n%s", got, want)
	}

	var buf testBufferCloser
	osCreateFn = func(p string) (io.WriteCloser, error) {
		if p != wantPath {
			t.Errorf("incorrect backup path: got %s want %s", p, wantPath)
		}
		return &buf, nil
	}
	path, err = backup(cfs)
	if path != wantPath {
		t.Errorf("incorrect path in correct case: got\n%s\nwant\n%s", path, wantPath)
	}
	if err != nil {
		t.Errorf("unexpected error for backup: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(buf.b.Bytes()))
	i := 0
	for scanner.Scan() {
		if scanner.Err() != nil {
			t.Errorf("unexpected error while scanning: %v", scanner.Err())
		}
		got := scanner.Text()
		if want := lines[i]; got != want {
			t.Errorf("incorrect line at %d: got\n%s\nwant\n%s", i, got, want)
		}
		i++
	}

	osCreateFn = oldOSCreateFn
}

func TestGetBackups(t *testing.T) {
	errGlob := "glob error"
	correctFile1 := path.Join(os.TempDir(), "timescaledb_tune.backup201901181122")
	correctFile2 := path.Join(os.TempDir(), "timescaledb_tune.backup201901191200")
	cases := []struct {
		desc        string
		onDiskFiles []string
		globErr     bool
		want        []string
		errMsg      string
	}{
		{
			desc:        "error on glob",
			onDiskFiles: []string{"foo"},
			globErr:     true,
			errMsg:      errGlob,
		},
		{
			desc:        "no matching files",
			onDiskFiles: []string{},
			want:        []string{},
		},
		{
			desc:        "invalid file",
			onDiskFiles: []string{"foo"},
			want:        []string{},
		},
		{
			desc:        "one correct file",
			onDiskFiles: []string{correctFile1},
			want:        []string{correctFile1},
		},
		{
			desc:        "two correct files",
			onDiskFiles: []string{correctFile1, correctFile2},
			want:        []string{correctFile1, correctFile2},
		},
		{
			desc:        "two correct files with wrong files",
			onDiskFiles: []string{"foo", correctFile1, "bar", correctFile2, "baz"},
			want:        []string{correctFile1, correctFile2},
		},
	}
	oldFilepathGlobFn := filepathGlobFn
	for _, c := range cases {
		filepathGlobFn = func(_ string) ([]string, error) {
			if c.globErr {
				return nil, fmt.Errorf(errGlob)
			}
			return c.onDiskFiles, nil
		}

		files, err := getBackups()
		if c.errMsg == "" && err != nil {
			t.Errorf("%s: unexpected error: got %v", c.desc, err)
		} else if c.errMsg != "" {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			}
			if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: unexpected error msg: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		}
		if got := len(files); got != len(c.want) {
			t.Errorf("%s: incorrect size of files: got %d want %d", c.desc, got, len(c.want))
		} else {
			for i, wantFile := range c.want {
				if got := files[i]; got != wantFile {
					t.Errorf("%s: incorrect file at index %d: got %s want %s", c.desc, i, got, wantFile)
				}
			}
		}
	}
	filepathGlobFn = oldFilepathGlobFn
}

func TestFSRestorer(t *testing.T) {
	fileContents := []byte("oneline\ntwoline\nthreeline\n")
	tmpfile, err := ioutil.TempFile("", "timescaledb-tune-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err = tmpfile.Write(fileContents); err != nil {
		t.Fatal(err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := tmpfile.Name()
	tmpfile2, err := ioutil.TempFile("", "timescaledb-tune-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile2.Name()) // clean up

	confPath := tmpfile2.Name()

	r := &fsRestorer{}
	err = r.Restore("", "")
	if err == nil {
		t.Fatalf("expected an error 1")
	}
	err = r.Restore(backupPath, "")
	if err == nil {
		t.Fatalf("expected an error 2")
	}
	err = r.Restore(backupPath, confPath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	backupContents, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	destContents, err := ioutil.ReadFile(tmpfile2.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := string(destContents); got != string(backupContents) {
		t.Errorf("contents not the same: got\n%s\nwant\n%s", got, string(backupContents))
	}
}
