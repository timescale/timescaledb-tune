package main

import (
	"bufio"
	"fmt"
	"io"
)

// configFileState represents that the postgresql.conf file, including all of its
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
