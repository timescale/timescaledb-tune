package tstune

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

const (
	osMac   = "darwin"
	osLinux = "linux"

	fileNameMac       = "/usr/local/var/postgres/postgresql.conf"
	fileNameDebianFmt = "/etc/postgresql/%s/main/postgresql.conf"
	fileNameRPMFmt    = "/var/lib/pgsql/%s/data/postgresql.conf"
	fileNameArch      = "/var/lib/postgres/data/postgresql.conf"

	errConfigNotFoundFmt   = "could not find postgresql.conf at any of these locations:\n%v"
	errBackupNotCreatedFmt = "could not create backup at %s: %v"

	backupFilePrefix = "timescaledb_tune.backup"
	backupDateFmt    = "200601021504"
)

// allows us to substitute mock versions in tests
var osStatFn = os.Stat
var osCreateFn = func(path string) (io.Writer, error) {
	return os.Create(path)
}

// fileExists is a simple check for stating if a file exists and if any error
// occurs it returns false.
func fileExists(name string) bool {
	// for our purposes, any error is a problem, so assume it does not exist
	if _, err := osStatFn(name); err != nil {
		return false
	}
	return true
}

// getConfigFilePath attempts to find the postgresql.conf file using path heuristics
// for different operating systems. If successful it returns the full path to
// the file; otherwise, it returns with an empty path and error.
func getConfigFilePath(os, pgVersion string) (string, error) {
	tried := []string{}
	try := func(format string, args ...interface{}) string {
		fileName := fmt.Sprintf(format, args...)
		tried = append(tried, fileName)
		if fileExists(fileName) {
			return fileName
		}
		return ""
	}
	switch os {
	case osMac:
		fileName := try(fileNameMac)
		if fileName != "" {
			return fileName, nil
		}
	case osLinux:
		fileName := try(fileNameDebianFmt, pgVersion)
		if fileName != "" {
			return fileName, nil
		}
		fileName = try(fileNameRPMFmt, pgVersion)
		if fileName != "" {
			return fileName, nil
		}

		fileName = try(fileNameArch)
		if fileName != "" {
			return fileName, nil
		}
	}
	return "", fmt.Errorf(errConfigNotFoundFmt, strings.Join(tried, "\n"))
}

type tunableParseResult struct {
	idx       int
	commented bool
	missing   bool
	key       string
	value     string
	extra     string
}

// configFileState represents the postgresql.conf file, including all of its
// lines, the parsed result of the shared_preload_libraries line, and parse results
// for parameters we care about tuning
type configFileState struct {
	lines            []string                       // all the lines, to be updated for output
	sharedLibResult  *sharedLibResult               // parsing result for shared lib line
	tuneParseResults map[string]*tunableParseResult // mapping of each tunable param to its parsed line result
}

// getConfigFileState returns the current state of the configuration file by
// reading it line by line and parsing those lines we particularly care about.
func getConfigFileState(r io.Reader) (*configFileState, error) {
	cfs := &configFileState{
		lines:            []string{},
		tuneParseResults: make(map[string]*tunableParseResult),
	}
	i := 0
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return nil, fmt.Errorf("could not read postgresql.conf: %v", scanner.Err())
		}
		line := scanner.Text()
		temp := parseLineForSharedLibResult(line)
		if temp != nil {
			temp.idx = i
			cfs.sharedLibResult = temp
		} else {
			for k, regex := range regexes {
				tpr := parseWithRegex(line, regex)
				if tpr != nil {
					tpr.idx = i
					cfs.tuneParseResults[k] = tpr
				}
			}
		}
		cfs.lines = append(cfs.lines, line)
		i++
	}
	return cfs, nil
}

// Backup writes the conf file state to the system's temporary directory
// with a well known name format so it can potentially be restored.
func (cfs *configFileState) Backup() (string, error) {
	backupName := backupFilePrefix + time.Now().Format(backupDateFmt)
	backupPath := path.Join(os.TempDir(), backupName)
	bf, err := osCreateFn(backupPath)
	if err != nil {
		return backupPath, fmt.Errorf(errBackupNotCreatedFmt, backupPath, err)
	}
	_, err = cfs.WriteTo(bf)
	return backupPath, err
}

func (cfs *configFileState) WriteTo(w io.Writer) (int64, error) {
	ret := int64(0)
	for _, l := range cfs.lines {
		n, err := w.Write([]byte(l + "\n"))
		if err != nil {
			return 0, err
		}
		ret += int64(n)
	}
	return ret, nil
}
