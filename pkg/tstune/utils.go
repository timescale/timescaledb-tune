package tstune

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/timescale/timescaledb-tune/pkg/pgutils"
)

// ValidPGVersions is a slice representing the major versions of PostgreSQL
// for which recommendations can be generated.
var ValidPGVersions = []string{
	pgutils.MajorVersion17,
	pgutils.MajorVersion16,
	pgutils.MajorVersion15,
	pgutils.MajorVersion14,
	pgutils.MajorVersion13,
	pgutils.MajorVersion12,
	pgutils.MajorVersion11,
	pgutils.MajorVersion10,
	pgutils.MajorVersion96,
}

// allows us to substitute mock versions in tests
var getPGConfigVersionFn = pgutils.GetPGConfigVersionAtPath

// allows us to substitute mock versions in tests
var osStatFn = os.Stat

// fileExists is a simple check for stating if a file exists and if any error
// occurs it returns false.
func fileExists(name string) bool {
	// for our purposes, any error is a problem, so assume it does not exist
	if _, err := osStatFn(name); err != nil {
		return false
	}
	return true
}

func pathIsDir(name string) bool {
	fi, err := osStatFn(name)
	// for our purposes, any error is a problem, so it is not a directory
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// dirPathToFile will construct a full path if the given path
// is a directory. This allows us to also accept directory paths for
// well-known files (postgresql.conf, pg_config)
func dirPathToFile(path string, defaultFilename string) string {
	if len(path) > 0 && pathIsDir(path) {
		return filepath.Join(path, defaultFilename)
	}
	return path
}

// isCloseEnough checks whether a provided value actual is within +/- the
// fudge factor fudge of target.
func isCloseEnough(actual, target, fudge float64) bool {
	return math.Abs((target-actual)/target) <= fudge
}

// isIn checks whether a given string s is inside the []string arr.
func isIn(s string, arr []string) bool {
	for _, x := range arr {
		if s == x {
			return true
		}
	}
	return false
}

// validatePGMajorVersion tests whether majorVersion is a major version of
// PostgreSQL that this package knows how to handle.
func validatePGMajorVersion(majorVersion string) error {
	if !isIn(majorVersion, ValidPGVersions) {
		return fmt.Errorf(errUnsupportedMajorFmt, majorVersion)
	}
	return nil
}

// getPGMajorVersion extracts the major version of PostgreSQL according to
// the output of pg_config located at binPath. It validates that it is a
// version that tstune knows how to handle.
func getPGMajorVersion(binPath string) (string, error) {
	version, err := getPGConfigVersionFn(binPath)
	if err != nil {
		return "", fmt.Errorf(errCouldNotExecuteFmt, binPath, err)
	}
	majorVersion, err := pgutils.ToPGMajorVersion(string(version))
	if err != nil {
		return "", err
	}
	if err = validatePGMajorVersion(majorVersion); err != nil {
		return "", err
	}
	return majorVersion, nil
}
