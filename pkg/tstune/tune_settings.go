package tstune

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

// tuneRegexFmt is a regular expression that is used to match a line in the
// conf file that just needs to be Sprintf'd with the key name. That is, its
// usage is usually:
// regex := fmt.Sprintf(tuneRegexFmt, "key_name")
const tuneRegexFmt = "^(\\s*#+?\\s*)?(%s) = (\\S+?)(\\s*(?:#.*|))$"

var regexes = make(map[string]*regexp.Regexp)

type floatParser interface {
	ParseFloat(string) (float64, error)
}

type bytesFloatParser struct{}

func (v *bytesFloatParser) ParseFloat(s string) (float64, error) {
	return parse.PGFormatToBytes(s)
}

type numericFloatParser struct{}

func (v *numericFloatParser) ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// keyToRegex takes a conf file key/param name and creates the correct regular
// expression.
func keyToRegex(key string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(tuneRegexFmt, key))
}

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
}

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
