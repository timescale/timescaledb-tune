package tstune

import (
	"bufio"
	"io"
	"os"
)

const exitLabel = "exit"

var exitFn = os.Exit

// ioHandler manages the reading and writing for a Tuner
type ioHandler struct {
	p      printer       // handles output
	br     *bufio.Reader // handles input
	out    io.Writer
	outErr io.Writer
}

func (h *ioHandler) exit(errCode int, format string, args ...interface{}) {
	h.p.Error(exitLabel, format, args...)
	exitFn(errCode)
}

func (h *ioHandler) errorExit(err error) {
	h.exit(1, err.Error())
}
