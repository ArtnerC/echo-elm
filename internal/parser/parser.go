// Package parser wraps the ANTLR4-generated CQL grammar to produce
// an echo-elm AST and a collection of diagnostics.
//
// Usage:
//
//	result, err := parser.ParseString(src, "MyLibrary.cql")
//	if err != nil {
//	    // syntax error - result may still be partially populated
//	}
//	for _, d := range result.Diagnostics {
//	    fmt.Println(d)
//	}
package parser

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/parser/cqlparser"
)

// Diagnostic represents a parsing error or warning with location.
type Diagnostic struct {
	Severity DiagnosticSeverity
	Message  string
	Loc      ast.Interval
}

func (d Diagnostic) String() string {
	sev := "error"
	if d.Severity == Warning {
		sev = "warning"
	}
	return "[" + sev + "] " + d.Loc.String() + ": " + d.Message
}

// DiagnosticSeverity is the severity of a diagnostic.
type DiagnosticSeverity int

const (
	Error   DiagnosticSeverity = iota
	Warning DiagnosticSeverity = iota
)

// Result is the output of a parse operation.
type Result struct {
	Library     *ast.Library  // nil on fatal error
	Diagnostics []Diagnostic
}

// HasErrors returns true if any diagnostics have Error severity.
func (r *Result) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// ParseString parses a CQL source string and returns the AST and diagnostics.
// sourceName is used in diagnostic messages (e.g. "MyLibrary.cql").
func ParseString(src string, sourceName string) (*Result, error) {
	input := antlr.NewInputStream(src)
	return parseInput(input, sourceName)
}

// ParseBytes parses a CQL source as a byte slice.
func ParseBytes(src []byte, sourceName string) (*Result, error) {
	input := antlr.NewInputStream(string(src))
	return parseInput(input, sourceName)
}

func parseInput(input antlr.CharStream, sourceName string) (*Result, error) {
	result := &Result{}

	errListener := &diagnosticListener{sourceName: sourceName, result: result}

	lexer := cqlparser.NewcqlLexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errListener)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := cqlparser.NewcqlParser(stream)
	p.RemoveErrorListeners()
	p.AddErrorListener(errListener)
	p.SetErrorHandler(antlr.NewBailErrorStrategy())

	// Recover from bail-out panic
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(*antlr.ParseCancellationException); ok {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: Error,
					Message:  "parse cancelled: " + pe.GetMessage(),
					Loc:      ast.Interval{SourceName: sourceName},
				})
			}
			// Any other panic: let it propagate
		}
	}()

	// Replace BailErrorStrategy with a tolerant one after adding listeners
	p.SetErrorHandler(antlr.NewDefaultErrorStrategy())

	tree := p.Library()
	if tree == nil {
		return result, nil
	}

	// Build AST from parse tree
	builder := newASTBuilder(sourceName)
	result.Library = builder.buildLibrary(tree)
	result.Diagnostics = append(result.Diagnostics, errListener.diagnostics...)

	return result, nil
}

// -----------------------------------------------------------------------
// Error listener
// -----------------------------------------------------------------------

type diagnosticListener struct {
	*antlr.DefaultErrorListener
	sourceName  string
	result      *Result
	diagnostics []Diagnostic
}

func (d *diagnosticListener) SyntaxError(
	_ antlr.Recognizer,
	_ interface{},
	line, col int,
	msg string,
	_ antlr.RecognitionException,
) {
	d.diagnostics = append(d.diagnostics, Diagnostic{
		Severity: Error,
		Message:  msg,
		Loc: ast.Interval{
			SourceName: d.sourceName,
			Start:      ast.Position{Line: line, Column: col},
			Stop:       ast.Position{Line: line, Column: col},
		},
	})
}
