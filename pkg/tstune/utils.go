package tstune

import (
	"math"
	"os/exec"
)

// isCloseEnough checks whether a provided value actual is within +/- the
// fudge factor fudge of target.
func isCloseEnough(actual, target, fudge float64) bool {
	return math.Abs((target-actual)/target) <= fudge
}

// isIn checks whether a given string s is inside the []string arr.
func isIn(s string, arr []string) bool {
	for _, x := range arr {
		if s == x {
			return true
		}
	}
	return false
}

func getPGConfigVersion(binPath string) ([]byte, error) {
	return exec.Command(binPath, "--version").Output()
}
