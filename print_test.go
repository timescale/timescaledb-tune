package main

import "fmt"

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

func (p *testPrinter) Error(_ string, format string, args ...interface{}) {
	p.errorCalls++
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}
