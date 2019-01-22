package tstune

import (
	"fmt"
	"strconv"
)

var (
	errSkip = fmt.Errorf("skip err")
)

func isYes(s string) bool {
	return s == "y" || s == "yes"
}

func isNo(s string) bool {
	return s == "n" || s == "no"
}

func isSkip(s string) bool {
	return s == "s" || s == "skip"
}

func isQuit(s string) bool {
	return s == "q" || s == "quit"
}

type promptChecker interface {
	Check(string) (bool, error)
}

type yesNoChecker struct {
	errMsg string
	args   []interface{}
}

func newYesNoChecker(errMsg string, args ...interface{}) *yesNoChecker {
	return &yesNoChecker{errMsg, args}
}

func (c *yesNoChecker) Check(r string) (bool, error) {
	if isNo(r) {
		return false, fmt.Errorf(c.errMsg, c.args...)
	} else if isYes(r) {
		return true, nil
	}
	return false, nil
}

type skipChecker struct {
	err error
}

func newSkipChecker(errMsg string, args ...interface{}) *skipChecker {
	return &skipChecker{fmt.Errorf(errMsg, args...)}
}

func (c *skipChecker) Check(r string) (bool, error) {
	if isQuit(r) || isNo(r) {
		return false, c.err
	} else if isYes(r) {
		return true, nil
	} else if isSkip(r) {
		return true, errSkip
	}
	return false, nil
}

type numberedListChecker struct {
	limit    int
	err      error
	response int
}

func newNumberedListChecker(limit int, errMsg string, args ...interface{}) *numberedListChecker {
	return &numberedListChecker{limit, fmt.Errorf(errMsg, args...), 0}
}

func (c *numberedListChecker) Check(r string) (bool, error) {
	if isQuit(r) {
		return false, c.err
	}
	num, err := strconv.ParseInt(r, 10, 0)
	if err != nil {
		return false, err
	} else if num < 1 || int(num) > c.limit {
		return false, nil
	}
	c.response = int(num)
	return true, nil
}
