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
)

type printer interface {
	Statement(string, ...any)
	Prompt(string, ...any)
	Success(string, ...any)
	Error(string, string, ...any)
}

type colorPrinter struct {
	w io.Writer
}

func (p *colorPrinter) printWithColor(c *color.Color, format string, args ...any) {
	color.NoColor = false
	c.Fprintf(p.w, format, args...)
}

func (p *colorPrinter) Statement(format string, args ...any) {
	p.printWithColor(statementColor, format+"\n", args...)
}

func (p *colorPrinter) Prompt(format string, args ...any) {
	p.printWithColor(promptColor, format, args...)
}

func (p *colorPrinter) Success(format string, args ...any) {
	p.printWithColor(successColor, successLabel)
	fmt.Fprintf(p.w, format+"\n", args...)
}

func (p *colorPrinter) Error(label string, format string, args ...any) {
	p.printWithColor(errorColor, label+": ")
	fmt.Fprintf(p.w, format+"\n", args...)
}

type noColorPrinter struct {
	w io.Writer
}

func (p *noColorPrinter) Statement(format string, args ...any) {
	fmt.Fprintf(p.w, noColorPrefixStatement+format+"\n", args...)
}

func (p *noColorPrinter) Prompt(format string, args ...any) {
	fmt.Fprintf(p.w, noColorPrefixPrompt+format, args...)
}

func (p *noColorPrinter) Success(format string, args ...any) {
	fmt.Fprintf(p.w, strings.ToUpper(successLabel)+format+"\n", args...)
}

func (p *noColorPrinter) Error(label string, format string, args ...any) {
	fmt.Fprintf(p.w, strings.ToUpper(label)+": "+format+"\n", args...)
}
