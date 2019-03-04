package pgutils

import (
	"fmt"
	"os/exec"
	"regexp"
)

// Major version strings for recent PostgreSQL versions
const (
	MajorVersion94 = "9.4"
	MajorVersion95 = "9.5"
	MajorVersion96 = "9.6"
	MajorVersion10 = "10"
	MajorVersion11 = "11"
)

const (
	errCouldNotParseVersionFmt = "unable to parse PG version string: %s"
	errUnknownMajorVersionFmt  = "unknown major PG version: %s"
)

var pgVersionRegex = regexp.MustCompile("^PostgreSQL ([0-9]+?).([0-9]+?).*")

// ToPGMajorVersion returns the major PostgreSQL version associated with a given
// version string, as given from an invocation of `pg_config --version`. This
// string has the form of "PostgreSQL X.Y[.Z (extra)]". For versions before 10,
// the major version is defined as X.Y, whereas starting with 10, it is defined
// as just X. That is, "PostgreSQL 10.3" returns "10" and "PostgreSQL 9.6.4"
// returns "9.6".
func ToPGMajorVersion(val string) (string, error) {
	res := pgVersionRegex.FindStringSubmatch(val)
	if len(res) != 3 {
		return "", fmt.Errorf(errCouldNotParseVersionFmt, val)
	}
	switch res[1] {
	case MajorVersion11:
		fallthrough
	case MajorVersion10:
		return res[1], nil
	case "9":
		fallthrough
	case "8":
		fallthrough
	case "7":
		return res[1] + "." + res[2], nil
	default:
		return "", fmt.Errorf(errUnknownMajorVersionFmt, val)
	}
}

// GetPGConfigVersion executes the pg_config binary (assuming it is in PATH) to
// get the version of PostgreSQL associated with it.
func GetPGConfigVersion() (string, error) {
	return GetPGConfigVersionAtPath("pg_config")
}

// GetPGConfigVersionAtPath executes the (pg_config) binary at path to get the
// version of PostgreSQL associated with it.
func GetPGConfigVersionAtPath(path string) (string, error) {
	output, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
