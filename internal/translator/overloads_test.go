package translator_test

import (
	"strings"
	"testing"
)

// fluentErrors returns the error diagnostics a translation reports for a fluent
// function called with ordinary syntax.
func fluentErrors(t *testing.T, cql string) []string {
	t.Helper()
	var out []string
	for _, d := range translate(t, cql).Diagnostics {
		if strings.EqualFold(d.Severity, "error") && strings.Contains(d.Message, "fluent") {
			out = append(out, d.Message)
		}
	}
	return out
}

// TestFluentFunctionNeedsFluentSyntax pins issues/08 #8: CQF rejects a fluent
// function invoked with ordinary call syntax, with this exact message. The
// harness skips the fixture as an upstream error, so the rule is pinned here.
func TestFluentFunctionNeedsFluentSyntax(t *testing.T) {
	errs := fluentErrors(t, `library L version '1.0'
define fluent function "Doubled"(n Integer): n * 2
define "Plain call": "Doubled"(2)`)
	want := "Operator Doubled with signature (System.Integer) is a fluent function and can only be invoked with fluent syntax."
	if len(errs) != 1 || errs[0] != want {
		t.Fatalf("errors = %q, want [%q]", errs, want)
	}
}

// TestFluentSyntaxAndNonFluentOverloadsAreAccepted pins the rule's limits: dot
// syntax is fine, and so is a plain call that a non-fluent overload of the
// same name can take.
func TestFluentSyntaxAndNonFluentOverloadsAreAccepted(t *testing.T) {
	for name, cql := range map[string]string{
		"fluent syntax": `library L version '1.0'
define fluent function "Doubled"(n Integer): n * 2
define "Fluent call": 2."Doubled"()`,
		"non-fluent overload": `library L version '1.0'
define fluent function "Twice"(n Integer): n * 2
define function "Twice"(s String): s + s
define "Plain call": "Twice"('a')`,
	} {
		if errs := fluentErrors(t, cql); len(errs) != 0 {
			t.Errorf("%s: unexpected errors %q", name, errs)
		}
	}
}

// TestMixedFlattenCompilesStatelessly pins the issues/08 W15(b) decision: a
// mixed-element flatten compiles regardless of earlier Flatten calls in the
// library. CQF 5.0.0 rejects the third definition as ambiguous; echo-elm
// deliberately does not reproduce that state dependence.
func TestMixedFlattenCompilesStatelessly(t *testing.T) {
	r := translate(t, `library L version '1.0'
define "Integer lists": flatten({ {1, 2}, {3} })
define "String lists": flatten({ {'a'}, {'b'} })
define "Mixed lists": flatten({ {1, 2}, {'a'} })`)
	for _, d := range r.Diagnostics {
		if strings.EqualFold(d.Severity, "error") {
			t.Errorf("unexpected error: %s", d.Message)
		}
	}
}
