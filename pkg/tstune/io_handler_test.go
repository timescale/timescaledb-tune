package tstune

import (
	"fmt"
	"testing"
)

func TestIOHandlerExit(t *testing.T) {
	p := &testPrinter{}

	oldExitFn := exitFn
	exitCalls := 0
	exitLastCode := -1
	exitFn = func(code int) {
		exitCalls++
		exitLastCode = code
	}

	handler := &ioHandler{p, nil, nil, nil}
	for i := 0; i < 100; i++ {
		handler.exit(i*2, "bye")
		if got := p.errorCalls; got != uint64(i+1) {
			t.Errorf("incorrect number of error calls: got %d want %d", got, i+1)
		}
		if got := exitCalls; got != i+1 {
			t.Errorf("incorrect number of exit calls: got %d want %d", got, i+1)
		}
		if got := exitLastCode; got != i*2 {
			t.Errorf("incorrect last code: got %d want %d", got, i*2)
		}

		want := fmt.Sprintf(exitLabel + ": bye")
		if got := p.errors[len(p.errors)-1]; got != want {
			t.Errorf("incorrect error: got %v want %v", got, want)
		}
	}

	exitFn = oldExitFn
}

func TestIOHandlerErrorExit(t *testing.T) {
	p := &testPrinter{}

	oldExitFn := exitFn
	exitCalls := 0
	exitLastCode := -1
	exitFn = func(code int) {
		exitCalls++
		exitLastCode = code
	}

	handler := &ioHandler{p, nil, nil, nil}
	for i := 0; i < 100; i++ {
		handler.errorExit(fmt.Errorf("error %d", i*3))
		if got := p.errorCalls; got != uint64(i+1) {
			t.Errorf("incorrect number of error calls: got %d want %d", got, i+1)
		}
		if got := exitCalls; got != i+1 {
			t.Errorf("incorrect number of exit calls: got %d want %d", got, i+1)
		}
		if got := exitLastCode; got != 1 {
			t.Errorf("incorrect last code: got %d want %d", got, 1)
		}
		want := fmt.Sprintf(exitLabel+": error %d", i*3)
		if got := p.errors[len(p.errors)-1]; got != want {
			t.Errorf("incorrect error: got %v want %v", got, want)
		}
	}

	exitFn = oldExitFn
}
