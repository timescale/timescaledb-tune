package tstune

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type testPrinter struct {
	statementCalls uint64
	statements     []string
	promptCalls    uint64
	prompts        []string
	successCalls   uint64
	successes      []string
	errorCalls     uint64
	errors         []string
}

func (p *testPrinter) Statement(format string, args ...interface{}) {
	p.statementCalls++
	p.statements = append(p.statements, fmt.Sprintf(format, args...))
}

func (p *testPrinter) Prompt(format string, args ...interface{}) {
	p.promptCalls++
	p.prompts = append(p.prompts, fmt.Sprintf(format, args...))
}

func (p *testPrinter) Success(format string, args ...interface{}) {
	p.successCalls++
	p.successes = append(p.successes, fmt.Sprintf(format, args...))
}

func (p *testPrinter) Error(label string, format string, args ...interface{}) {
	p.errorCalls++
	p.errors = append(p.errors, fmt.Sprintf(label+": "+format, args...))
}

const whiteBoldSeq = "\x1b[37;1m"
const purpleBoldSeq = "\x1b[35;1m"
const greenBoldSeq = "\x1b[32;1m"
const redBoldSeq = "\x1b[31;1m"
const resetSeq = "\x1b[0m"

func TestColorPrinterStatement(t *testing.T) {
	var buf bytes.Buffer
	p := &colorPrinter{&buf}
	stmt := "This is a statement with %d"
	p.Statement(stmt, 1)
	want := whiteBoldSeq + fmt.Sprintf(stmt+"\n", 1) + resetSeq
	if got := buf.String(); got != want {
		t.Errorf("incorrect statement: got\n%s\nwant\n%s", got, want)
	}
}

func TestColorPrinterPrompt(t *testing.T) {
	var buf bytes.Buffer
	p := &colorPrinter{&buf}
	stmt := "This is a prompt with %d"
	p.Prompt(stmt, 1)
	want := purpleBoldSeq + fmt.Sprintf(stmt, 1) + resetSeq
	if got := buf.String(); got != want {
		t.Errorf("incorrect prompt: got\n%s\nwant\n%s", got, want)
	}
}

func TestColorPrinterSuccess(t *testing.T) {
	var buf bytes.Buffer
	p := &colorPrinter{&buf}
	stmt := "This is a success with %d"
	p.Success(stmt, 1)
	want := greenBoldSeq + successLabel + resetSeq + fmt.Sprintf(stmt+"\n", 1)
	if got := buf.String(); got != want {
		t.Errorf("incorrect success: got\n%s\nwant\n%s", got, want)
	}
}

func TestColorPrinterError(t *testing.T) {
	var buf bytes.Buffer
	p := &colorPrinter{&buf}
	stmt := "This is a error with %d"
	label := "yikes"
	p.Error(label, stmt, 1)
	want := redBoldSeq + label + ": " + resetSeq + fmt.Sprintf(stmt+"\n", 1)
	if got := buf.String(); got != want {
		t.Errorf("incorrect error: got\n%s\nwant\n%s", got, want)
	}
}

func TestNoColorPrinterStatement(t *testing.T) {
	var buf bytes.Buffer
	p := &noColorPrinter{&buf}
	stmt := "This is a statement with %d"
	p.Statement(stmt, 1)
	want := noColorPrefixStatement + fmt.Sprintf(stmt+"\n", 1)
	if got := buf.String(); got != want {
		t.Errorf("incorrect statement: got\n%s\nwant\n%s", got, want)
	}
}

func TestNoColorPrinterPrompt(t *testing.T) {
	var buf bytes.Buffer
	p := &noColorPrinter{&buf}
	stmt := "This is a prompt with %d"
	p.Prompt(stmt, 1)
	want := noColorPrefixPrompt + fmt.Sprintf(stmt, 1)
	if got := buf.String(); got != want {
		t.Errorf("incorrect prompt: got\n%s\nwant\n%s", got, want)
	}
}

func TestNoColorPrinterSuccess(t *testing.T) {
	var buf bytes.Buffer
	p := &noColorPrinter{&buf}
	stmt := "This is a success with %d"
	p.Success(stmt, 1)
	want := strings.ToUpper(successLabel) + fmt.Sprintf(stmt+"\n", 1)
	if got := buf.String(); got != want {
		t.Errorf("incorrect success: got\n%s\nwant\n%s", got, want)
	}
}

func TestNoColorPrinterError(t *testing.T) {
	var buf bytes.Buffer
	p := &noColorPrinter{&buf}
	stmt := "This is a error with %d"
	label := "yikes"
	p.Error(label, stmt, 1)
	want := strings.ToUpper(label) + ": " + fmt.Sprintf(stmt+"\n", 1)
	if got := buf.String(); got != want {
		t.Errorf("incorrect error: got\n%s\nwant\n%s", got, want)
	}
}
