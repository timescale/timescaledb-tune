package main

import (
	"fmt"
	"math"
	"regexp"
	"runtime"
	"strconv"
)

const (
	regexFmt  = "^(\\s*#+?\\s*)?(%s) = (\\S+?)(\\s*(?:#.*|))$"
	osWindows = "windows"
)

var (
	regexes = make(map[string]*regexp.Regexp)
	parsers = make(map[string]parseFn)
)

type tunableParseResult struct {
	idx       int
	commented bool
	missing   bool
	key       string
	value     string
	extra     string
}

type recommender interface {
	Recommend(string) string
}

func keyToRegex(key string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(regexFmt, key))
}

func isIn(key string, arr []string) bool {
	for _, s := range arr {
		if key == s {
			return true
		}
	}
	return false
}

type parseFn func(string) (float64, error)

func keyToParseFn(key string) parseFn {
	if isIn(key, memoryKeys) || isIn(key, walKeys) {
		return parsePGStringToBytes
	}

	return func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}
}

func init() {
	setup := func(arr []string) {
		for _, k := range arr {
			regexes[k] = keyToRegex(k)
			parsers[k] = keyToParseFn(k)
		}
	}
	if runtime.GOOS == osWindows {
		otherKeys = otherKeys[:len(otherKeys)-1]
	}

	setup(memoryKeys)
	setup(parallelKeys)
	setup(walKeys)
	setup(otherKeys)
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

func isCloseEnough(actual, target, fudge float64) bool {
	return math.Abs((target-actual)/target) <= fudge
}
