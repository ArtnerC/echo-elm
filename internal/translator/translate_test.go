package translator_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/parser"
	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/translator"
)

func translate(t *testing.T, cql string, opts ...func(*translator.Options)) *translator.Result {
	t.Helper()
	pr, err := parser.ParseBytes([]byte(cql), "test.cql")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	o := translator.DefaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	tr := translator.New(o)
	return tr.Translate(pr.Library, "test.cql")
}

func TestMinimalLibrary(t *testing.T) {
	r := translate(t, "library Minimal version '1.0.0'")
	if r.Library == nil {
		t.Fatal("expected Library, got nil")
	}
	if r.Library.Identifier.ID != "Minimal" {
		t.Errorf("id: want Minimal, got %q", r.Library.Identifier.ID)
	}
	if r.Library.Identifier.Version != "1.0.0" {
		t.Errorf("version: want 1.0.0, got %q", r.Library.Identifier.Version)
	}
	if len(r.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", r.Diagnostics)
	}
}

func TestUsingsAndIncludes(t *testing.T) {
	cql := `library Test version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers`
	r := translate(t, cql)
	if r.Library.Usings == nil {
		t.Fatal("expected Usings")
	}
	// System + FHIR
	if len(r.Library.Usings.Def) != 2 {
		t.Errorf("want 2 using defs, got %d", len(r.Library.Usings.Def))
	}
	if r.Library.Usings.Def[1].LocalIdentifier != "FHIR" {
		t.Errorf("want FHIR using, got %q", r.Library.Usings.Def[1].LocalIdentifier)
	}
	if r.Library.Includes == nil || len(r.Library.Includes.Def) != 1 {
		t.Fatal("expected 1 include")
	}
	inc := r.Library.Includes.Def[0]
	if inc.LocalIdentifier != "FHIRHelpers" {
		t.Errorf("include alias: want FHIRHelpers, got %q", inc.LocalIdentifier)
	}
}

func TestContextSection(t *testing.T) {
	cql := `library TestCtx version '1.0'
using FHIR version '4.0.1'
context Patient
define "InitPop": true`
	r := translate(t, cql)

	if r.Library.Contexts == nil {
		t.Fatal("expected Contexts section")
	}
	if len(r.Library.Contexts.Def) != 1 || r.Library.Contexts.Def[0].Name != "Patient" {
		t.Errorf("context def: %+v", r.Library.Contexts.Def)
	}

	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) < 2 {
		t.Fatalf("expected >=2 statement defs (accessor + InitPop), got %d", len(stmts.Def))
	}
	// First statement should be the implicit Patient accessor
	if stmts.Def[0].Name != "Patient" {
		t.Errorf("first stmt: want Patient, got %q", stmts.Def[0].Name)
	}
}

func TestCQFModeAnnotations(t *testing.T) {
	cql := `library CQFTest version '1.0'
using FHIR version '4.0.1'
context Patient
define "InitPop": true`

	r := translate(t, cql, func(o *translator.Options) { o.CQFMode = true })

	// Library.Annotation is []json.RawMessage; first entry is CqlToElmInfo
	if len(r.Library.Annotation) == 0 {
		t.Fatal("expected library annotation")
	}
	var info struct {
		TranslatorOptions string `json:"translatorOptions"`
	}
	if err := json.Unmarshal(r.Library.Annotation[0], &info); err != nil {
		t.Fatalf("unmarshal info: %v", err)
	}
	if info.TranslatorOptions != "" {
		t.Errorf("CQF mode translatorOptions: want empty, got %q", info.TranslatorOptions)
	}

	// Every UsingDef should have annotation:[]
	for i, ud := range r.Library.Usings.Def {
		if string(ud.Annotation) != "[]" {
			t.Errorf("UsingDef[%d].Annotation: want [], got %s", i, ud.Annotation)
		}
	}

	// ContextDef should have annotation:[]
	cd := r.Library.Contexts.Def[0]
	if string(cd.Annotation) != "[]" {
		t.Errorf("ContextDef.Annotation: want [], got %s", cd.Annotation)
	}

	// StatementDefs should have annotation:[]
	for i, sd := range r.Library.Statements.Def {
		if string(sd.Annotation) != "[]" {
			t.Errorf("StatementDef[%d].Annotation: want [], got %s", i, sd.Annotation)
		}
	}
}

func TestCQFModeVsModernMode(t *testing.T) {
	cql := `library DiffTest version '1.0'
using FHIR version '4.0.1'`

	cqfResult := translate(t, cql, func(o *translator.Options) { o.CQFMode = true })
	modernResult := translate(t, cql)

	// In CQF mode, System UsingDef has annotation:[]
	cqfSystemAnnotation := cqfResult.Library.Usings.Def[0].Annotation
	if string(cqfSystemAnnotation) != "[]" {
		t.Errorf("CQF System annotation: want [], got %s", cqfSystemAnnotation)
	}

	// In modern mode, annotation is nil (omitted)
	modernSystemAnnotation := modernResult.Library.Usings.Def[0].Annotation
	if modernSystemAnnotation != nil {
		t.Errorf("modern System annotation: want nil, got %s", modernSystemAnnotation)
	}

	// CQF mode emits empty translatorOptions; modern emits non-empty options string
	var cqfInfo, modernInfo struct {
		TranslatorOptions string `json:"translatorOptions"`
	}
	if len(cqfResult.Library.Annotation) > 0 {
		_ = json.Unmarshal(cqfResult.Library.Annotation[0], &cqfInfo)
	}
	if len(modernResult.Library.Annotation) > 0 {
		_ = json.Unmarshal(modernResult.Library.Annotation[0], &modernInfo)
	}

	if cqfInfo.TranslatorOptions != "" {
		t.Errorf("CQF translatorOptions: want empty, got %q", cqfInfo.TranslatorOptions)
	}
	if modernInfo.TranslatorOptions == "" {
		t.Errorf("modern translatorOptions: want non-empty")
	}
}

func TestDiagnosticsOnSyntaxError(t *testing.T) {
	cql := `library Bad define broken @@@@`
	pr, _ := parser.ParseBytes([]byte(cql), "bad.cql")
	o := translator.DefaultOptions()
	tr := translator.New(o)
	r := tr.Translate(pr.Library, "bad.cql")

	// Parse errors come through parse result, but translator should still produce a library.
	if r.Library == nil {
		t.Fatal("expected Library even on syntax error")
	}
	// Name from error-recovered parse
	if r.Library.Identifier == nil {
		t.Fatal("expected non-nil Identifier")
	}
}

func TestLibrarySourceInterface(t *testing.T) {
	// Verify MapSource lookup works correctly.
	m := resolver.NewMapSource(map[string][]byte{
		"FHIRHelpers":        []byte("library FHIRHelpers version '4.0.1'"),
		"FHIRHelpers|4.0.1": []byte("library FHIRHelpers version '4.0.1'"),
	})

	src, ok, err := m.GetLibrarySource("FHIRHelpers", "4.0.1")
	if err != nil || !ok {
		t.Fatalf("MapSource versioned lookup failed: ok=%v err=%v", ok, err)
	}
	if !strings.Contains(string(src), "FHIRHelpers") {
		t.Error("unexpected source content")
	}

	src2, ok2, err2 := m.GetLibrarySource("FHIRHelpers", "")
	if err2 != nil || !ok2 {
		t.Fatalf("MapSource unversioned lookup failed: ok=%v err=%v", ok2, err2)
	}
	if !strings.Contains(string(src2), "FHIRHelpers") {
		t.Error("unexpected source content")
	}
}

func TestMultiSource(t *testing.T) {
	first := resolver.NewMapSource(map[string][]byte{
		"LibA": []byte("library LibA"),
	})
	second := resolver.NewMapSource(map[string][]byte{
		"LibB": []byte("library LibB"),
	})
	multi := resolver.NewMultiSource(first, second)

	_, ok, _ := multi.GetLibrarySource("LibA", "")
	if !ok {
		t.Error("MultiSource: expected LibA from first source")
	}
	_, ok, _ = multi.GetLibrarySource("LibB", "")
	if !ok {
		t.Error("MultiSource: expected LibB from second source")
	}
	_, ok, _ = multi.GetLibrarySource("LibC", "")
	if ok {
		t.Error("MultiSource: LibC should not be found")
	}
}
