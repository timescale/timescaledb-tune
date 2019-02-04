package tstune

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	backupFilePrefix = "timescaledb_tune.backup"
	backupDateFmt    = "200601021504"

	errBackupNotCreatedFmt = "could not create backup at %s: %v"
)

// allows us to substitute mock versions in tests
var filepathGlobFn = filepath.Glob
var osCreateFn = func(path string) (io.Writer, error) {
	return os.Create(path)
}

// backup writes the conf file state to the system's temporary directory
// with a well known name format so it can potentially be restored.
func backup(cfs *configFileState) (string, error) {
	backupName := backupFilePrefix + time.Now().Format(backupDateFmt)
	backupPath := path.Join(os.TempDir(), backupName)
	bf, err := osCreateFn(backupPath)
	if err != nil {
		return backupPath, fmt.Errorf(errBackupNotCreatedFmt, backupPath, err)
	}
	_, err = cfs.WriteTo(bf)
	return backupPath, err
}

// getBackups returns a list of files that match timescaledb-tune's backup
// filename format.
func getBackups() ([]string, error) {
	backupPattern := path.Join(os.TempDir(), backupFilePrefix+"*")
	files, err := filepathGlobFn(backupPattern)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	stripPrefix := path.Join(os.TempDir(), backupFilePrefix)
	for _, f := range files {
		datePart := strings.Replace(f, stripPrefix, "", -1)
		_, err := time.Parse(backupDateFmt, datePart)
		if err != nil {
			continue
		}
		ret = append(ret, f)
	}
	return ret, nil
}

type restorer interface {
	Restore(string, string) error
}

type fsRestorer struct{}

func (r *fsRestorer) Restore(backupPath, confPath string) error {
	backupFile, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	defer backupFile.Close()

	backupCFS, err := getConfigFileState(backupFile)
	if err != nil {
		return err
	}

	confFile, err := os.OpenFile(confPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer confFile.Close()

	_, err = backupCFS.WriteTo(confFile)
	if err != nil {
		return err
	}

	return nil
}
