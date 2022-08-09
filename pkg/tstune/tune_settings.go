package tstune

import (
	"fmt"
	"regexp"
	"time"

	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

// Names of parameters that this tuning tool will add to the conf file.
const (
	fmtOurParam           = "%s = '%s'"
	lastTunedParam        = "timescaledb.last_tuned"
	lastTunedVersionParam = "timescaledb.last_tuned_version"
)

// ourParams is a list of parameters that the tuning program adds to the conf file
var ourParams = []string{lastTunedParam, lastTunedVersionParam}

// ourParamToValue returns the configuration file line for a given
// timescaledb-tune parameter, e.g., timescaledb.last_tuned.
func ourParamString(param string) string {
	var val string
	switch param {
	case lastTunedParam:
		val = time.Now().Format(time.RFC3339)
	case lastTunedVersionParam:
		val = Version
	default:
		panic("unknown param: " + param)
	}
	return fmt.Sprintf(fmtOurParam, param, val)
}

const (
	// tuneRegexFmt is a regular expression that is used to match a line in the
	// conf file that just needs to be Sprintf'd with the key name. That is, its
	// usage is usually:
	// regex := fmt.Sprintf(tuneRegexFmt, "key_name")
	tuneRegexFmt = "^(\\s*#+?\\s*)?(%s) = (\\S+?)(\\s*(?:#.*|))$"
	// tuneRegexQuotedFmt is similar to the format above but for string parameters
	// that need single quotes around them
	tuneRegexQuotedFmt = "^(\\s*#+?\\s*)?(%s) = '(.+?)'(\\s*(?:#.*|))$"
)

var regexes = make(map[string]*regexp.Regexp)

func init() {
	setup := func(arr []string) {
		for _, k := range arr {
			regexes[k] = keyToRegex(k)
		}
	}

	setup(pgtune.MemoryKeys)
	setup(pgtune.ParallelKeys)
	setup(pgtune.WALKeys)
	setup(pgtune.MiscKeys)
	setup(pgtune.BgwriterKeys)
}

// keyToRegex takes a conf file key/param name and creates the correct regular
// expression.
func keyToRegex(key string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(tuneRegexFmt, regexp.QuoteMeta(key)))
}

// keyToRegexQuoted takes a conf file key/param name and creates the correct
// regular expression, assuming the values need to be single quoted.
func keyToRegexQuoted(key string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(tuneRegexQuotedFmt, regexp.QuoteMeta(key)))
}

// parseWithRegex takes a line and attempts to parse it using a given regular
// expression regex. The regex is expected to produce 5 capture groups:
// 1) the full result, 2) whether the line is preceded by # or not, 3) the
// parameter name/key, 4) the parameter value, and 5) any comments at the end.
// If successful, a tunableParseResult is returned based on the contents of the
// line; otherwise, nil. Panics if the regex parsing returns and unexpected
// result (i.e., too many capture groups).
func parseWithRegex(line string, regex *regexp.Regexp) *tunableParseResult {
	res := regex.FindStringSubmatch(line)
	if len(res) > 0 {
		if len(res) != 5 {
			panic(fmt.Sprintf("unexpected regex parse result: %v (len = %d)", res, len(res)))
		}

		return &tunableParseResult{
			commented: len(res[1]) > 0,
			key:       res[2],
			value:     res[3],
			extra:     res[4],
		}
	}
	return nil
}
