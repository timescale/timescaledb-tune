package tstune

import (
	"regexp"
	"strings"
)

const extName = "timescaledb"

var sharedRegex = regexp.MustCompile("(#+?\\s*)?shared_preload_libraries = '(.*?)'.*")

// sharedLibResult holds the results of extracting/parsing the shared_preload_libraries
// line of a postgresql.conf file.
type sharedLibResult struct {
	idx          int    // the line index where this result was parsed
	commented    bool   // whether the line is currently commented out (i.e., prepended by #)
	hasTimescale bool   // whether 'timescaledb' appears in the list of libraries
	commentGroup string // the combination of # + spaces that appear before the key / value
	libs         string // the string value of the libraries currently set in the config file
}

// parseLineForSharedLibResult attempts to parse a line of the config file by
// matching it against the shared_preload_libraries regex. If the line is
// parseable by the regex, then the representation of that line is returned;
// otherwise, nil.
func parseLineForSharedLibResult(line string) *sharedLibResult {
	res := sharedRegex.FindStringSubmatch(line)
	if len(res) > 0 {
		return &sharedLibResult{
			commented:    len(res[1]) > 0,
			hasTimescale: strings.Contains(res[2], extName),
			commentGroup: res[1],
			libs:         res[2],
		}
	}
	return nil
}

// updateSharedLibLine takes a given line that matched the shared_preload_libraries
// regex and updates it to validly include the 'timescaledb' extension.
func updateSharedLibLine(line string, parseResult *sharedLibResult) string {
	res := line
	if parseResult.commented {
		res = strings.Replace(res, parseResult.commentGroup, "", 1)
	}

	if parseResult.hasTimescale {
		return res
	}
	newLibsVal := "= '"
	if len(parseResult.libs) > 0 {
		newLibsVal += parseResult.libs + ","
	}
	newLibsVal += extName + "'"
	replaceVal := "= '" + parseResult.libs + "'"
	res = strings.Replace(res, replaceVal, newLibsVal, 1)

	return res
}
