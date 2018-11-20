package main

import (
	"strings"

	"github.com/fatih/color"
)

const successLabel = "success: "

var (
	statementColor = color.New(color.FgWhite, color.Bold)   // color for directions / statements
	promptColor    = color.New(color.FgMagenta, color.Bold) // color for prompt/questions requiring user input
	successColor   = color.New(color.FgGreen, color.Bold)
	errorColor     = color.New(color.FgRed, color.Bold)
)

type printer interface {
	Statement(string, ...interface{})
	Prompt(string, ...interface{})
	Success(string, ...interface{})
	Error(string, string, ...interface{})
}

type colorPrinter struct{}

func printWithColor(c *color.Color, format string, args ...interface{}) {
	c.Printf(format, args...)
}

func (p *colorPrinter) Statement(format string, args ...interface{}) {
	printWithColor(statementColor, format+"\n", args...)
}

func (p *colorPrinter) Prompt(format string, args ...interface{}) {
	printWithColor(promptColor, format, args...)
}

func (p *colorPrinter) Success(format string, args ...interface{}) {
	printWithColor(successColor, successLabel)
	printFn(format+"\n", args...)
}

func (p *colorPrinter) Error(label string, format string, args ...interface{}) {
	printWithColor(errorColor, label+": ")
	printFn(format+"\n", args...)
}

type noColorPrinter struct{}

func (p *noColorPrinter) Statement(format string, args ...interface{}) {
	printFn("== "+format+"\n", args...)
}

func (p *noColorPrinter) Prompt(format string, args ...interface{}) {
	printFn("-- "+format, args...)
}

func (p *noColorPrinter) Success(format string, args ...interface{}) {
	printFn(strings.ToUpper(successLabel)+format+"\n", args...)
}

func (p *noColorPrinter) Error(label string, format string, args ...interface{}) {
	printFn(strings.ToUpper(label)+": "+format+"\n", args...)
}
