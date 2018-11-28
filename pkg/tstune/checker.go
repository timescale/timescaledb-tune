package tstune

import (
	"fmt"
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
