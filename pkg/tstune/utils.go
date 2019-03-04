package tstune

import (
	"fmt"
	"math"

	"github.com/timescale/timescaledb-tune/pkg/pgutils"
)

// ValidPGVersions is a slice representing the major versions of PostgreSQL
// for which recommendations can be generated.
var ValidPGVersions = []string{
	pgutils.MajorVersion11,
	pgutils.MajorVersion10,
	pgutils.MajorVersion96,
}

// allows us to substitute mock versions in tests
var getPGConfigVersionFn = pgutils.GetPGConfigVersionAtPath

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
