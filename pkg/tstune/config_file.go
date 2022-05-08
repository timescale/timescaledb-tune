package tstune

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	osMac   = "darwin"
	osLinux = "linux"

	fileNameMac       = "/usr/local/var/postgres/postgresql.conf"
	fileNameMacM1     = "/opt/homebrew/var/postgres/postgresql.conf"
	fileNameDebianFmt = "/etc/postgresql/%s/main/postgresql.conf"
	fileNameRPMFmt    = "/var/lib/pgsql/%s/data/postgresql.conf"
	fileNameArch      = "/var/lib/postgres/data/postgresql.conf"
	fileNameAlpine    = "/var/lib/postgresql/data/postgresql.conf"

	errConfigNotFoundFmt = "could not find postgresql.conf at any of these locations:\n%v"
)

type truncateWriter interface {
	io.Writer
	Seek(int64, int) (int64, error)
	Truncate(int64) error
}

// getConfigFilePath attempts to find the postgresql.conf file using path heuristics
// for different operating systems. If successful it returns the full path to
// the file; otherwise, it returns with an empty path and error.
func getConfigFilePath(system, pgVersion string) (string, error) {
	tried := []string{}
	try := func(format string, args ...interface{}) string {
		fileName := fmt.Sprintf(format, args...)
		tried = append(tried, fileName)
		if fileExists(fileName) {
			return fileName
		}
		return ""
	}
	pgdata := os.Getenv("PGDATA")
	if pgdata != "" {
		fileName := try(pgdata + "/postgresql.conf")
		if fileName != "" {
			return fileName, nil
		}
	}
	switch system {
	case osMac:
		fileName := try(fileNameMac)
		if fileName != "" {
			return fileName, nil
		}
		fileName = try(fileNameMacM1)
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
		fileName = try(fileNameAlpine)
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

// configLine represents a line in the conf file with some associated metadata
// on how it should be processed when written.
type configLine struct {
	content string
	remove  bool
}

// configLineProcessor is an interface that processes a line of the conf file to
// do some manipulation, typically called in a loop or multiple times on different
// lines. For example, a dupes remover could be called on all lines and mark
// the lines that, for some key, appear multiple times for deletion.
type configLineProcessor interface {
	Process(*configLine) error
}

// removesDuplicatesProcessor is used to track and mark duplicates for removal that
// match a particular regex.
type removeDuplicatesProcessor struct {
	prev  *configLine
	regex *regexp.Regexp
}

// Process takes the input l to determine if any previous lines should be marked
// for removal. The processor alays tracks the last instance matching the regex
// it has seen, so if it encounters a new one, it can mark the previous instance
// for removal.
func (p *removeDuplicatesProcessor) Process(l *configLine) error {
	if found := parseWithRegex(l.content, p.regex); found != nil {
		if p.prev != nil {
			p.prev.remove = true
		}
		p.prev = l
	}
	return nil
}

// getRemoveDuplicatesProcessors is a convenience function for creating a slice
// of configLineProcessors (of the removeDuplicatesProcessor type) for a set of
// keys.
func getRemoveDuplicatesProcessors(keys []string) []configLineProcessor {
	ret := []configLineProcessor{}
	for _, key := range keys {
		ret = append(ret, &removeDuplicatesProcessor{regex: keyToRegexQuoted(key)})
	}
	return ret
}

// configFileState represents the postgresql.conf file, including all of its
// lines, the parsed result of the shared_preload_libraries line, and parse results
// for parameters we care about tuning
type configFileState struct {
	lines            []*configLine                  // all the lines, to be updated for output
	sharedLibResult  *sharedLibResult               // parsing result for shared lib line
	tuneParseResults map[string]*tunableParseResult // mapping of each tunable param to its parsed line result
}

// getConfigFileState returns the current state of the configuration file by
// reading it line by line and parsing those lines we particularly care about.
func getConfigFileState(r io.Reader) (*configFileState, error) {
	cfs := &configFileState{
		lines:            []*configLine{},
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
		cfs.lines = append(cfs.lines, &configLine{content: line})
		i++
	}
	return cfs, nil
}

func (cfs *configFileState) ProcessLines(processors ...configLineProcessor) error {
	var err error
	for _, line := range cfs.lines {
		for _, p := range processors {
			err = p.Process(line)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cfs *configFileState) WriteTo(w io.Writer) (int64, error) {
	// in case new output is shorter than old, need to truncate first
	switch t := w.(type) {
	case truncateWriter:
		err := t.Truncate(0)
		if err != nil {
			return 0, err
		}
		_, err = t.Seek(0, 0)
		if err != nil {
			return 0, err
		}
	}
	ret := int64(0)
	for _, l := range cfs.lines {
		if l.remove {
			continue
		}

		n, err := w.Write([]byte(l.content + "\n"))
		if err != nil {
			return 0, err
		}
		ret += int64(n)
	}
	return ret, nil
}
