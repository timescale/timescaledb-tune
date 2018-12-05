package tstune

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestIsIn(t *testing.T) {
	limit := 1000
	arr := []string{}
	for i := 0; i < limit; i++ {
		arr = append(arr, fmt.Sprintf("str%d", i))
	}

	// Should always be in the arr
	for i := 0; i < limit*10; i++ {
		s := fmt.Sprintf("str%d", rand.Intn(limit))
		if !isIn(s, arr) {
			t.Errorf("should be in the arr: %s", s)
		}
	}

	// Should never be in the arr
	for i := 0; i < limit*10; i++ {
		s := fmt.Sprintf("str%d", limit+rand.Intn(limit))
		if isIn(s, arr) {
			t.Errorf("should not be in the arr: %s", s)
		}
	}
}
