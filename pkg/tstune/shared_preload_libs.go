package tstune

import (
	"regexp"
	"strings"
)

const (
	extName             = "timescaledb"
	pgTextsearchExtName = "pg_textsearch"
)

var sharedRegex = regexp.MustCompile("(#+?\\s*)?shared_preload_libraries = '(.*?)'.*")

// sharedLibResult holds the results of extracting/parsing the shared_preload_libraries
// line of a postgresql.conf file.
type sharedLibResult struct {
	idx             int    // the line index where this result was parsed
	commented       bool   // whether the line is currently commented out (i.e., prepended by #)
	hasTimescale    bool   // whether 'timescaledb' appears in the list of libraries
	hasPgTextsearch bool   // whether 'pg_textsearch' appears in the list of libraries
	commentGroup    string // the combination of # + spaces that appear before the key / value
	libs            string // the string value of the libraries currently set in the config file
}

// parseLineForSharedLibResult attempts to parse a line of the config file by
// matching it against the shared_preload_libraries regex. If the line is
// parseable by the regex, then the representation of that line is returned;
// otherwise, nil.
func parseLineForSharedLibResult(line string) *sharedLibResult {
	res := sharedRegex.FindStringSubmatch(line)
	if len(res) > 0 {
		return &sharedLibResult{
			commented:       len(res[1]) > 0,
			hasTimescale:    hasLibToken(res[2], extName),
			hasPgTextsearch: hasLibToken(res[2], pgTextsearchExtName),
			commentGroup:    res[1],
			libs:            res[2],
		}
	}
	return nil
}

// hasLibToken reports whether libs contains name as a comma-separated token,
// so substrings like "my_timescaledb" or "pg_textsearch_ext" don't match.
func hasLibToken(libs, name string) bool {
	for _, tok := range strings.Split(libs, ",") {
		if strings.TrimSpace(tok) == name {
			return true
		}
	}
	return false
}

// pgTextsearchDetected reports whether pg_textsearch will actually be loaded
// once tuning completes. A commented-out shared_preload_libraries line gets
// uncommented (with timescaledb merged in) by processSharedLibLine, so a
// pg_textsearch entry there still counts — the caller has already gated on
// whether the tuning flow will proceed to write those changes.
func pgTextsearchDetected(res *sharedLibResult) bool {
	return res != nil && res.hasPgTextsearch
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
