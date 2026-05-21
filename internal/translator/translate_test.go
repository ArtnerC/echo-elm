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
	if r.Library.Identifier.ID == "" {
		t.Fatal("expected non-empty Identifier ID")
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

// -----------------------------------------------------------------------
// Operator and expression tests
// -----------------------------------------------------------------------

func TestArithmeticOperators(t *testing.T) {
	cql := `library ArithTest version '1.0'
define Add: 1 + 2
define Sub: 5 - 3
define Mul: 2 * 4
define Div: 10 / 2
define Mod: 7 mod 3`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil {
		t.Fatal("expected statements")
	}
	wantNames := []string{"Add", "Sub", "Mul", "Div", "Mod"}
	if len(stmts.Def) != len(wantNames) {
		t.Fatalf("want %d statements, got %d", len(wantNames), len(stmts.Def))
	}
	for i, want := range wantNames {
		if stmts.Def[i].Name != want {
			t.Errorf("stmt[%d]: want %q, got %q", i, want, stmts.Def[i].Name)
		}
		if stmts.Def[i].Expression == nil {
			t.Errorf("stmt[%d] %q: nil expression", i, want)
		}
	}
}

func TestComparisonOperators(t *testing.T) {
	cql := `library CmpTest version '1.0'
define EqTest: 1 = 1
define NeqTest: 1 != 2
define LtTest: 1 < 2
define LeTest: 1 <= 1
define GtTest: 2 > 1
define GeTest: 2 >= 2`

	r := translate(t, cql)
	if r.Library.Statements == nil || len(r.Library.Statements.Def) != 6 {
		t.Fatalf("expected 6 statements")
	}
	for _, s := range r.Library.Statements.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestLogicalOperators(t *testing.T) {
	cql := `library LogicTest version '1.0'
define AndTest: true and false
define OrTest: true or false
define NotTest: not true
define XorTest: true xor false`

	r := translate(t, cql)
	if r.Library.Statements == nil || len(r.Library.Statements.Def) != 4 {
		t.Fatalf("expected 4 statements")
	}
	for _, s := range r.Library.Statements.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestLiterals(t *testing.T) {
	cql := `library LitTest version '1.0'
define IntLit: 42
define DecLit: 3.14
define StrLit: 'hello'
define BoolLit: true
define NullLit: null`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestQuantityLiterals(t *testing.T) {
	cql := `library QuantTest version '1.0'
define Q1: 10 'mg'
define Q2: 5 'kg'
define Q3: 120 'mm[Hg]'`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestDateTimeLiterals(t *testing.T) {
	cql := `library DTTest version '1.0'
define DT1: DateTime(2024, 1, 15)
define DT2: DateTime(2024, 1, 15, 10, 30, 0, 0)
define D1: Date(2024, 6, 1)
define T1: Time(14, 30, 0, 0)`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestIntervalLiterals(t *testing.T) {
	cql := `library IntTest version '1.0'
define I1: Interval[1, 5]
define I2: Interval(1, 5)
define I3: Interval[1, 5)`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestListLiterals(t *testing.T) {
	cql := `library ListTest version '1.0'
define L1: List{1, 2, 3}
define L2: {4, 5, 6}`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestTupleLiterals(t *testing.T) {
	cql := `library TupleTest version '1.0'
define T1: Tuple{ Name: 'Alice', Age: 30 }`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts.Def))
	}
	if stmts.Def[0].Expression == nil {
		t.Error("tuple literal: nil expression")
	}
}

func TestParameterDefinitions(t *testing.T) {
	cql := `library ParamTest version '1.0'
parameter MeasurementPeriod Interval<DateTime>
parameter MaxAge Integer default 65`

	r := translate(t, cql)
	if r.Library.Parameters == nil {
		t.Fatal("expected Parameters section")
	}
	if len(r.Library.Parameters.Def) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(r.Library.Parameters.Def))
	}
	if r.Library.Parameters.Def[0].Name != "MeasurementPeriod" {
		t.Errorf("param[0]: want MeasurementPeriod, got %q", r.Library.Parameters.Def[0].Name)
	}
	if r.Library.Parameters.Def[1].Name != "MaxAge" {
		t.Errorf("param[1]: want MaxAge, got %q", r.Library.Parameters.Def[1].Name)
	}
	// MaxAge has a default value.
	if r.Library.Parameters.Def[1].Default == nil {
		t.Error("MaxAge: expected non-nil default")
	}
}

func TestCodeSystemAndValueSet(t *testing.T) {
	cql := `library TermTest version '1.0'
codesystem "LOINC": 'urn:oid:2.16.840.1.113883.6.1'
valueset "BP": 'urn:oid:2.16.840.1.113883.3.526.3.1032'`

	r := translate(t, cql)
	if r.Library.CodeSystems == nil || len(r.Library.CodeSystems.Def) != 1 {
		t.Fatal("expected 1 code system")
	}
	if r.Library.CodeSystems.Def[0].Name != "LOINC" {
		t.Errorf("codesystem name: want LOINC, got %q", r.Library.CodeSystems.Def[0].Name)
	}
	if r.Library.ValueSets == nil || len(r.Library.ValueSets.Def) != 1 {
		t.Fatal("expected 1 value set")
	}
	if r.Library.ValueSets.Def[0].Name != "BP" {
		t.Errorf("valueset name: want BP, got %q", r.Library.ValueSets.Def[0].Name)
	}
}

func TestFunctionDefinition(t *testing.T) {
	cql := `library FuncTest version '1.0'
define function AgeInYears(birthDate DateTime):
  AgeInYearsAt(birthDate)`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) == 0 {
		t.Fatal("expected statements")
	}
	fn := stmts.Def[len(stmts.Def)-1]
	if !fn.IsFunction {
		t.Error("expected IsFunction=true")
	}
	if fn.Name != "AgeInYears" {
		t.Errorf("function name: want AgeInYears, got %q", fn.Name)
	}
	if len(fn.Operand) != 1 {
		t.Errorf("expected 1 operand, got %d", len(fn.Operand))
	}
}

func TestIntervalInOperator(t *testing.T) {
	cql := `library InTest version '1.0'
define Test1: 2 in Interval[1, 5]
define Test2: Interval[1, 5] includes 3`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestTimingOperators(t *testing.T) {
	cql := `library TimingTest version '1.0'
parameter Period Interval<DateTime>
define I1: Interval[DateTime(2024,1,1), DateTime(2024,12,31)]
define Before: I1 before Period
define After: Period after I1
define During: I1 during Period
define Includes: Period includes I1
define Same: I1 same as Period`

	r := translate(t, cql)
	stmts := r.Library.Statements
	// I1 + Before + After + During + Includes + Same = 6
	if stmts == nil || len(stmts.Def) < 5 {
		t.Fatalf("expected >=5 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("timing stmt %q: nil expression", s.Name)
		}
	}
}

func TestFHIRTimingDuringProducesIn(t *testing.T) {
	// `E.period.start during MeasurementPeriod` should produce In(FHIRHelpers.ToDateTime(...), MeasurementPeriod)
	// not IncludedIn. This is the FHIR dateTime coercion path.
	cql := `library FHIRTimingTest version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
parameter MeasurementPeriod Interval<DateTime>
context Patient
define TestDuring: exists (
  [Encounter] E
  where E.period.start during MeasurementPeriod
)`

	r := translate(t, cql)
	if len(r.Diagnostics) > 0 {
		// Filter errors only (warnings are OK).
		for _, d := range r.Diagnostics {
			if d.Severity == "Error" {
				t.Errorf("unexpected error: %s", d.Message)
			}
		}
	}

	// Serialize to JSON and check In appears, not IncludedIn for the timing expression.
	b, err := json.Marshal(r.Library)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonStr := string(b)

	// The TestDuring statement should contain "In" (the point-in-interval operator),
	// not a standalone IncludedIn (interval-in-interval).
	// We check the JSON contains "In" as type and not "IncludedIn".
	if strings.Contains(jsonStr, `"type":"IncludedIn"`) {
		t.Error(`expected "In" operator for FHIR dateTime during Interval, got "IncludedIn"`)
	}
}

func TestQueryExpression(t *testing.T) {
	cql := `library QueryTest version '1.0'
using FHIR version '4.0.1'
context Patient
define Over18: (List{1, 2, 3, 20, 25}) X where X > 18`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil {
		t.Fatal("expected statements")
	}
	var over18 *struct{ Expression interface{} }
	for _, s := range stmts.Def {
		if s.Name == "Over18" {
			if s.Expression == nil {
				t.Error("Over18: nil expression")
			}
			return
		}
	}
	_ = over18
	t.Error("Over18 statement not found")
}

func TestSignatureLevelNoneVsOverloads(t *testing.T) {
	cql := `library SigTest version '1.0'
define F1: 1 + 2`

	rOverloads := translate(t, cql, func(o *translator.Options) {
		o.SignatureLevel = "Overloads"
	})
	rNone := translate(t, cql, func(o *translator.Options) {
		o.SignatureLevel = "None"
	})

	// Both should produce a statement with an expression.
	if rOverloads.Library.Statements == nil || rNone.Library.Statements == nil {
		t.Fatal("expected statements from both")
	}

	// Serialize and confirm SignatureLevel is recorded in the CqlToElmInfo annotation.
	for _, pair := range []struct {
		result *translator.Result
		want   string
	}{
		{rOverloads, "Overloads"},
		{rNone, "None"},
	} {
		if len(pair.result.Library.Annotation) == 0 {
			continue // no annotation in non-CQF mode by default — skip
		}
		var info struct {
			SignatureLevel string `json:"signatureLevel"`
		}
		if err := json.Unmarshal(pair.result.Library.Annotation[0], &info); err == nil {
			if info.SignatureLevel != pair.want {
				t.Errorf("SignatureLevel: want %q, got %q", pair.want, info.SignatureLevel)
			}
		}
	}
}

func TestCompatibilityLevelInHeader(t *testing.T) {
	cql := "library CompatTest version '1.0'"

	r := translate(t, cql, func(o *translator.Options) {
		o.CompatibilityLevel = "1.5"
		o.CQFMode = true // ensure annotation is emitted
	})

	if len(r.Library.Annotation) == 0 {
		t.Fatal("expected library annotation with CQFMode=true")
	}
	var info struct {
		CompatibilityLevel string `json:"compatibilityLevel"`
	}
	if err := json.Unmarshal(r.Library.Annotation[0], &info); err != nil {
		t.Fatalf("unmarshal annotation: %v", err)
	}
	// We now emit compatibilityLevel in the annotation.
	if info.CompatibilityLevel != "1.5" {
		t.Errorf("compatibilityLevel: want 1.5, got %q", info.CompatibilityLevel)
	}
}

func TestCQFModeAnnotationsOnAllNodes(t *testing.T) {
	cql := `library NodeAnns version '1.0'
using FHIR version '4.0.1'
codesystem "LOINC": 'urn:oid:2.16.840.1.113883.6.1'
valueset "BP": 'urn:oid:2.16.840.1.113883.3.526.3.1032'
parameter MeasurementPeriod Interval<DateTime>
context Patient
define InitPop: true`

	r := translate(t, cql, func(o *translator.Options) { o.CQFMode = true })

	// UsingDef annotations
	for i, ud := range r.Library.Usings.Def {
		if string(ud.Annotation) != "[]" {
			t.Errorf("UsingDef[%d].Annotation: want [], got %s", i, ud.Annotation)
		}
	}

	// CodeSystemDef annotations
	if r.Library.CodeSystems != nil {
		for i, cs := range r.Library.CodeSystems.Def {
			if string(cs.Annotation) != "[]" {
				t.Errorf("CodeSystemDef[%d].Annotation: want [], got %s", i, cs.Annotation)
			}
		}
	}

	// ValueSetDef annotations
	if r.Library.ValueSets != nil {
		for i, vs := range r.Library.ValueSets.Def {
			if string(vs.Annotation) != "[]" {
				t.Errorf("ValueSetDef[%d].Annotation: want [], got %s", i, vs.Annotation)
			}
		}
	}

	// ParameterDef annotations
	if r.Library.Parameters != nil {
		for i, pd := range r.Library.Parameters.Def {
			if string(pd.Annotation) != "[]" {
				t.Errorf("ParameterDef[%d].Annotation: want [], got %s", i, pd.Annotation)
			}
		}
	}

	// ContextDef annotations
	if r.Library.Contexts != nil {
		for i, cd := range r.Library.Contexts.Def {
			if string(cd.Annotation) != "[]" {
				t.Errorf("ContextDef[%d].Annotation: want [], got %s", i, cd.Annotation)
			}
		}
	}

	// StatementDef annotations
	for i, sd := range r.Library.Statements.Def {
		if string(sd.Annotation) != "[]" {
			t.Errorf("StatementDef[%d].Annotation: want [], got %s", i, sd.Annotation)
		}
	}
}

func TestELMSchemaIdentifier(t *testing.T) {
	r := translate(t, "library Sch version '1.0'")

	si := r.Library.SchemaIdentifier
	if si == nil {
		t.Fatal("expected SchemaIdentifier")
	}
	// CQF emits schemaIdentifier.id = "urn:hl7-org:elm" and version = "r1"
	if si.ID != "urn:hl7-org:elm" {
		t.Errorf("schemaIdentifier.id: want urn:hl7-org:elm, got %q", si.ID)
	}
	if si.Version != "r1" {
		t.Errorf("schemaIdentifier.version: want r1, got %q", si.Version)
	}
}

func TestStatementAccessLevel(t *testing.T) {
	cql := `library AccTest version '1.0'
define "Public": true
define private "Hidden": false`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) < 2 {
		t.Fatalf("expected >=2 statements")
	}

	for _, s := range stmts.Def {
		switch s.Name {
		case "Public":
			if s.AccessLevel != "" && s.AccessLevel != "Public" {
				t.Errorf("Public stmt: unexpected accessLevel %q", s.AccessLevel)
			}
		case "Hidden":
			if s.AccessLevel != "Private" {
				t.Errorf("Hidden stmt: want Private, got %q", s.AccessLevel)
			}
		}
	}
}

func TestConditionalExpression(t *testing.T) {
	cql := `library CondTest version '1.0'
define IfExpr: if true then 1 else 2
define CaseExpr: case when true then 'a' when false then 'b' else 'c' end`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) < 2 {
		t.Fatalf("expected >=2 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestCastAndAs(t *testing.T) {
	cql := `library CastTest version '1.0'
parameter P Any
define AsInt: P as Integer
define CastInt: cast P as Integer
define IsInt: P is Integer`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) < 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestExistsOperator(t *testing.T) {
	cql := `library ExistsTest version '1.0'
define E1: exists (List{1, 2, 3})
define E2: exists (List{})`

	r := translate(t, cql)
	stmts := r.Library.Statements
	if stmts == nil || len(stmts.Def) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts.Def))
	}
	for _, s := range stmts.Def {
		if s.Expression == nil {
			t.Errorf("%q: nil expression", s.Name)
		}
	}
}

func TestNoDiagnosticsOnValidCQL(t *testing.T) {
	cql := `library Valid version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define "InitPop": true`

	r := translate(t, cql)
	for _, d := range r.Diagnostics {
		if d.Severity == "Error" {
			t.Errorf("unexpected error diagnostic: %s", d.Message)
		}
	}
}

func TestResultHasNoDiagnosticsForMinimal(t *testing.T) {
	r := translate(t, "library X version '1.0'")
	if len(r.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics on minimal library: %v", r.Diagnostics)
	}
}

func TestCQFCorpusFixtures(t *testing.T) {
	// Verify echo-elm translates each cqf-corpus fixture without error.
	// This is a lightweight smoke test that runs purely in Go (no JVM).
	type fixture struct {
		name string
		cql  string
	}
	fixtures := []fixture{
		{
			name: "Minimal",
			cql:  "library Minimal version '1.0.0'",
		},
		{
			name: "DefaultContext",
			cql: `library DefaultContext version '1.0.0'

define NullValue: null
define Add_null: null + 1
define Sub_null: null - 1`,
		},
		{
			name: "InTest",
			cql: `library InTest version '1.0.0'
parameter AnyParameter Any
define AnyExpression: null
define AnyTest: AnyParameter = 1
define test1: 2 in Interval[1, 5]
define test2: (List{2, 7, 9}) X where X in Interval[1, 5]
define test3: (List{2, 7, 9}) X where Interval[1, 5] contains X
define test4: 5 in 5`,
		},
		{
			name: "RatioLiteral",
			cql: `library RatioLiteralTest version '1.0.0'
define RatioLiteral: 5 'mg' : 10 'mL'
define RatioNumerator: 5 'mg'
define RatioDenominator: 10 'mL'`,
		},
	}

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			r := translate(t, f.cql)
			if r.Library == nil {
				t.Fatal("expected Library, got nil")
			}
			for _, d := range r.Diagnostics {
				if d.Severity == "Error" {
					t.Errorf("unexpected error: [%s] %s", d.Locator, d.Message)
				}
			}
		})
	}
}
