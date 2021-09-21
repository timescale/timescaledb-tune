package tstune

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/timescale/timescaledb-tune/pkg/pgutils"
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

func TestDirPathToFile(t *testing.T) {
	currentDir := "."
	missingFile := "ghost.txt"
	defaultFile := "replace.txt"
	cases := []struct {
		desc     string
		dirname  string
		filename string
		want     string
	}{
		{
			desc:     "augmented directory with default file",
			dirname:  currentDir,
			filename: defaultFile,
			want:     filepath.Join(currentDir, defaultFile),
		},
		{
			desc:     "could not find file - no action",
			dirname:  missingFile,
			filename: defaultFile,
			want:     missingFile,
		},
	}

	for _, c := range cases {
		if got := dirPathToFile(c.dirname, c.filename); got != c.want {
			t.Errorf("%s: incorrect result: got %v want %v", c.desc, got, c.want)
		}
	}
}

func TestIsIn(t *testing.T) {
	limit := 1000
	arr := []string{}
	for i := 0; i < limit; i++ {
		arr = append(arr, fmt.Sprintf("str%d", i))
	}

	// Should always be in the arr
	for i := 0; i < limit*10; i++ {
		s := fmt.Sprintf("str%d", rand.Intn(limit))
		if !isIn(s, arr) {
			t.Errorf("should be in the arr: %s", s)
		}
	}

	// Should never be in the arr
	for i := 0; i < limit*10; i++ {
		s := fmt.Sprintf("str%d", limit+rand.Intn(limit))
		if isIn(s, arr) {
			t.Errorf("should not be in the arr: %s", s)
		}
	}
}

func TestGetPGMajorVersion(t *testing.T) {
	okPath96 := "pg_config_9.6"
	okPath10 := "pg_config_10"
	okPath11 := "pg_config_11"
	okPath12 := "pg_config_12"
	okPath13 := "pg_config_13"
	okPath14 := "pg_config_14"
	okPath95 := "pg_config_9.5"
	okPath60 := "pg_config_6.0"
	cases := []struct {
		desc    string
		binPath string
		want    string
		errMsg  string
	}{
		{
			desc:    "failed execute",
			binPath: "pg_config_bad",
			errMsg:  fmt.Sprintf(errCouldNotExecuteFmt, "pg_config_bad", exec.ErrNotFound),
		},
		{
			desc:    "failed major parse",
			binPath: okPath60,
			errMsg:  "unknown major PG version: PostgreSQL 6.0.5",
		},
		{
			desc:    "failed unsupported",
			binPath: okPath95,
			errMsg:  fmt.Sprintf(errUnsupportedMajorFmt, "9.5"),
		},
		{
			desc:    "success 9.6",
			binPath: okPath96,
			want:    pgutils.MajorVersion96,
		},
		{
			desc:    "success 10",
			binPath: okPath10,
			want:    pgutils.MajorVersion10,
		},
		{
			desc:    "success 11",
			binPath: okPath11,
			want:    pgutils.MajorVersion11,
		},
		{
			desc:    "success 12",
			binPath: okPath12,
			want:    pgutils.MajorVersion12,
		},
		{
			desc:    "success 13",
			binPath: okPath13,
			want:    pgutils.MajorVersion13,
		},
		{
			desc:    "success 14",
			binPath: okPath14,
			want:    pgutils.MajorVersion14,
		},
	}

	oldVersionFn := getPGConfigVersionFn
	getPGConfigVersionFn = func(binPath string) (string, error) {
		switch binPath {
		case okPath60:
			return "PostgreSQL 6.0.5", nil
		case okPath95:
			return "PostgreSQL 9.5.10", nil
		case okPath96:
			return "PostgreSQL 9.6.6", nil
		case okPath10:
			return "PostgreSQL 10.5 (Debian7)", nil
		case okPath11:
			return "PostgreSQL 11.1", nil
		case okPath12:
			return "PostgreSQL 12.4", nil
		case okPath13:
			return "PostgreSQL 13.2", nil
		case okPath14:
			return "PostgreSQL 14.0", nil
		default:
			return "", exec.ErrNotFound
		}
	}

	for _, c := range cases {
		got, err := getPGMajorVersion(c.binPath)
		if len(c.errMsg) == 0 {
			if err != nil {
				t.Errorf("%s: unexpected error: got %v", c.desc, err)
			}
			if got != c.want {
				t.Errorf("%s: incorrect major version: got %s want %s", c.desc, got, c.want)
			}
		} else {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			}
			if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error:\ngot\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		}
	}

	getPGConfigVersionFn = oldVersionFn
}

func TestValidatePGMajorVersion(t *testing.T) {
	cases := map[string]bool{
		pgutils.MajorVersion96: true,
		pgutils.MajorVersion10: true,
		pgutils.MajorVersion11: true,
		pgutils.MajorVersion12: true,
		pgutils.MajorVersion13: true,
		pgutils.MajorVersion14: true,
		"9.5":                  false,
		"1.2.3":                false,
		"9.6.6":                false,
		"10.2":                 false,
		"11.0":                 false,
	}
	for majorVersion, valid := range cases {
		err := validatePGMajorVersion(majorVersion)
		if valid && err != nil {
			t.Errorf("unexpected error: got %v", err)
		} else if !valid {
			if err == nil {
				t.Errorf("unexpected lack of error")
			}
			want := fmt.Errorf(errUnsupportedMajorFmt, majorVersion).Error()
			if got := err.Error(); got != want {
				t.Errorf("unexpected error: got %v want %v", got, want)
			}
		}
	}
}
