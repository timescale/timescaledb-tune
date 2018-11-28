package tstune

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

const (
	successLabel           = "success: "
	noColorPrefixStatement = "== "
	noColorPrefixPrompt    = "-- "
)

var (
	statementColor = color.New(color.FgWhite, color.Bold)   // color for directions / statements
	promptColor    = color.New(color.FgMagenta, color.Bold) // color for prompt/questions requiring user input
	successColor   = color.New(color.FgGreen, color.Bold)
	errorColor     = color.New(color.FgRed, color.Bold)

	printFn = fmt.Fprintf
)

type printer interface {
	Statement(string, ...interface{})
	Prompt(string, ...interface{})
	Success(string, ...interface{})
	Error(string, string, ...interface{})
}

type colorPrinter struct {
	w io.Writer
}

func (p *colorPrinter) printWithColor(c *color.Color, format string, args ...interface{}) {
	color.NoColor = false
	c.Fprintf(p.w, format, args...)
}

func (p *colorPrinter) Statement(format string, args ...interface{}) {
	p.printWithColor(statementColor, format+"\n", args...)
}

func (p *colorPrinter) Prompt(format string, args ...interface{}) {
	p.printWithColor(promptColor, format, args...)
}

func (p *colorPrinter) Success(format string, args ...interface{}) {
	p.printWithColor(successColor, successLabel)
	printFn(p.w, format+"\n", args...)
}

func (p *colorPrinter) Error(label string, format string, args ...interface{}) {
	p.printWithColor(errorColor, label+": ")
	printFn(p.w, format+"\n", args...)
}

type noColorPrinter struct {
	w io.Writer
}

func (p *noColorPrinter) Statement(format string, args ...interface{}) {
	printFn(p.w, noColorPrefixStatement+format+"\n", args...)
}

func (p *noColorPrinter) Prompt(format string, args ...interface{}) {
	printFn(p.w, noColorPrefixPrompt+format, args...)
}

func (p *noColorPrinter) Success(format string, args ...interface{}) {
	printFn(p.w, strings.ToUpper(successLabel)+format+"\n", args...)
}

func (p *noColorPrinter) Error(label string, format string, args ...interface{}) {
	printFn(p.w, strings.ToUpper(label)+": "+format+"\n", args...)
}
