package parser_test

import (
	"testing"

	"github.com/artnerc/echo-elm/internal/parser"
)

const simpleLibrary = `
library TestLib version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

codesystem "LOINC": 'http://loinc.org'

valueset "Blood Pressure VS": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1033'

parameter "Measurement Period" Interval<DateTime>

context Patient

define "Has BP Measurement":
  [Observation: "Blood Pressure VS"] BPObs
    where BPObs.status in {'final', 'amended'}
`

func TestParseSimpleLibrary(t *testing.T) {
	result, err := parser.ParseString(simpleLibrary, "TestLib.cql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Logf("diagnostic: %s", d)
		}
		t.Fatal("parse produced errors")
	}
	if result.Library == nil {
		t.Fatal("library is nil")
	}

	lib := result.Library
	if lib.Name == nil || lib.Name.Name != "TestLib" {
		t.Errorf("expected library name TestLib, got %v", lib.Name)
	}
	if lib.Name.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", lib.Name.Version)
	}
	if len(lib.Usings) != 1 {
		t.Errorf("expected 1 using, got %d", len(lib.Usings))
	}
	if lib.Usings[0].ModelName != "FHIR" {
		t.Errorf("expected FHIR model, got %s", lib.Usings[0].ModelName)
	}
	if lib.Usings[0].Version != "4.0.1" {
		t.Errorf("expected FHIR version 4.0.1, got %s", lib.Usings[0].Version)
	}
	if len(lib.Includes) != 1 {
		t.Errorf("expected 1 include, got %d", len(lib.Includes))
	}
	if lib.Includes[0].LocalName != "FHIRHelpers" {
		t.Errorf("expected FHIRHelpers include alias, got %s", lib.Includes[0].LocalName)
	}
	if len(lib.Codesystems) != 1 {
		t.Errorf("expected 1 codesystem, got %d", len(lib.Codesystems))
	}
	if lib.Codesystems[0].Name != "LOINC" {
		t.Errorf("expected LOINC codesystem, got %s", lib.Codesystems[0].Name)
	}
	if len(lib.Valuesets) != 1 {
		t.Errorf("expected 1 valueset, got %d", len(lib.Valuesets))
	}
	if len(lib.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(lib.Parameters))
	}
	if lib.Parameters[0].Name != "Measurement Period" {
		t.Errorf("expected 'Measurement Period' parameter, got %q", lib.Parameters[0].Name)
	}
	if lib.Context == nil || lib.Context.Name != "Patient" {
		t.Errorf("expected Patient context, got %v", lib.Context)
	}
	if len(lib.Statements) != 1 {
		t.Errorf("expected 1 statement, got %d", len(lib.Statements))
	}
	if lib.Statements[0].Name != "Has BP Measurement" {
		t.Errorf("expected 'Has BP Measurement' statement, got %q", lib.Statements[0].Name)
	}
}

func TestParseSyntaxError(t *testing.T) {
	result, err := parser.ParseString("library @@@ invalid !! syntax", "bad.cql")
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.HasErrors() {
		t.Error("expected parse errors for invalid syntax, got none")
	}
}

func TestParseMinimalLibrary(t *testing.T) {
	src := `library Minimal`
	result, err := parser.ParseString(src, "Minimal.cql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Library == nil {
		t.Fatal("library is nil")
	}
	if result.Library.Name == nil || result.Library.Name.Name != "Minimal" {
		t.Errorf("expected library name Minimal, got %v", result.Library.Name)
	}
}

func TestParseFunctionDefinition(t *testing.T) {
	src := `library FuncLib version '1.0.0'

define function Age(birthDate DateTime, asOf DateTime) returns Integer:
  duration in years between birthDate and asOf
`
	result, err := parser.ParseString(src, "FuncLib.cql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Logf("diagnostic: %s", d)
		}
		t.Fatal("unexpected parse errors")
	}
	if len(result.Library.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Library.Statements))
	}
	fn := result.Library.Statements[0]
	if !fn.IsFunction {
		t.Error("expected IsFunction=true")
	}
	if fn.Name != "Age" {
		t.Errorf("expected function name Age, got %q", fn.Name)
	}
	if len(fn.Operands) != 2 {
		t.Errorf("expected 2 operands, got %d", len(fn.Operands))
	}
}
