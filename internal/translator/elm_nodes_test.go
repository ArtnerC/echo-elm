package translator_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/translator"
)

// marshalDef returns the serialized ELM JSON for the named statement definition.
func marshalDef(t *testing.T, r *translator.Result, name string) map[string]interface{} {
	t.Helper()
	if r.Library == nil || r.Library.Statements == nil {
		t.Fatal("expected statements")
	}
	for _, def := range r.Library.Statements.Def {
		if def.Name != name {
			continue
		}
		b, err := json.Marshal(def)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("unmarshal %s: %v", name, err)
		}
		return m
	}
	t.Fatalf("no statement named %q", name)
	return nil
}

// exprOf returns the "expression" object of a marshalled statement definition.
func exprOf(t *testing.T, r *translator.Result, name string) map[string]interface{} {
	t.Helper()
	def := marshalDef(t, r, name)
	e, ok := def["expression"].(map[string]interface{})
	if !ok {
		t.Fatalf("statement %q has no expression object", name)
	}
	return e
}

func nodeType(t *testing.T, m map[string]interface{}) string {
	t.Helper()
	s, _ := m["type"].(string)
	return s
}

// TestDateTimeComponentOperators pins that `date from` / `time from` /
// `timezoneoffset from` are their own ELM operators rather than
// DateTimeComponentFrom precisions.
func TestDateTimeComponentOperators(t *testing.T) {
	r := translate(t, `library T version '1.0'
define D: date from @2025-01-15T10:30:00
define Tm: time from @2025-01-15T10:30:00
define TZ: timezoneoffset from @2025-01-15T10:30:00
define Y: year from @2025-01-15T10:30:00
define Ms: millisecond from @2025-01-15T10:30:00`)

	cases := []struct {
		def, wantType, wantPrecision string
	}{
		{"D", "DateFrom", ""},
		{"Tm", "TimeFrom", ""},
		{"TZ", "TimezoneOffsetFrom", ""},
		{"Y", "DateTimeComponentFrom", "Year"},
		{"Ms", "DateTimeComponentFrom", "Millisecond"},
	}
	for _, c := range cases {
		e := exprOf(t, r, c.def)
		if got := nodeType(t, e); got != c.wantType {
			t.Errorf("%s: type = %q, want %q", c.def, got, c.wantType)
		}
		prec, _ := e["precision"].(string)
		if prec != c.wantPrecision {
			t.Errorf("%s: precision = %q, want %q", c.def, prec, c.wantPrecision)
		}
		// All five are UnaryExpression in ELM: "operand" is an object, not an array.
		if _, isArray := e["operand"].([]interface{}); isArray {
			t.Errorf("%s: operand is an array, want a single object", c.def)
		}
	}
}

// TestDateFromDateOperandConverts pins the implicit Date→DateTime conversion for
// the DateTime-only component operators. DateTimeComponentFrom has a Date
// overload and takes its operand unchanged.
func TestDateFromDateOperandConverts(t *testing.T) {
	r := translate(t, `library T version '1.0'
define D: date from @2025-01-15
define Y: year from @2025-01-15`)

	inner, ok := exprOf(t, r, "D")["operand"].(map[string]interface{})
	if !ok {
		t.Fatal("D: expected operand object")
	}
	if got := nodeType(t, inner); got != "ToDateTime" {
		t.Errorf("D: operand type = %q, want ToDateTime", got)
	}

	inner, ok = exprOf(t, r, "Y")["operand"].(map[string]interface{})
	if !ok {
		t.Fatal("Y: expected operand object")
	}
	if got := nodeType(t, inner); got != "Date" {
		t.Errorf("Y: operand type = %q, want Date (no conversion)", got)
	}
}

// TestNegatedEqualityLowersToNot pins that != and !~ have no ELM operator of
// their own — CQF wraps Equal / Equivalent in Not.
func TestNegatedEqualityLowersToNot(t *testing.T) {
	r := translate(t, `library T version '1.0'
define NE: 1 != 2
define NEquiv: 1 !~ 2
define Eq: 1 = 2`)

	for _, c := range []struct{ def, wantInner string }{
		{"NE", "Equal"},
		{"NEquiv", "Equivalent"},
	} {
		e := exprOf(t, r, c.def)
		if got := nodeType(t, e); got != "Not" {
			t.Fatalf("%s: type = %q, want Not", c.def, got)
		}
		inner, ok := e["operand"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: expected single Not operand", c.def)
		}
		if got := nodeType(t, inner); got != c.wantInner {
			t.Errorf("%s: inner type = %q, want %q", c.def, got, c.wantInner)
		}
	}
	if got := nodeType(t, exprOf(t, r, "Eq")); got != "Equal" {
		t.Errorf("Eq: type = %q, want Equal", got)
	}
}

// TestInPromotionUsesInferredType pins that list promotion is decided from the
// operand types, not the syntactic form: a reference to a list-valued define
// already satisfies In(T, List<T>) and must not be wrapped in ToList.
func TestInPromotionUsesInferredType(t *testing.T) {
	r := translate(t, `library T version '1.0'
define Numbers: { 1, 2, 3 }
define Period: Interval[@2024-01-01, @2024-12-31]
define Day: @2024-06-01
define InListRef: 2 in Numbers
define InIntervalRef: Day in Period
define InIntervalOfInterval: Interval[@2024-02-01, @2024-03-01] in Period`)

	for _, def := range []string{"InListRef", "InIntervalRef"} {
		operands, ok := exprOf(t, r, def)["operand"].([]interface{})
		if !ok || len(operands) != 2 {
			t.Fatalf("%s: expected two operands", def)
		}
		rhs, _ := operands[1].(map[string]interface{})
		if got := nodeType(t, rhs); got == "ToList" {
			t.Errorf("%s: right operand was promoted with ToList, want the reference unchanged", def)
		}
	}

	// An interval left operand resolves to In(T, List<T>) with T = Interval<T'>,
	// so here the right operand IS promoted.
	operands, ok := exprOf(t, r, "InIntervalOfInterval")["operand"].([]interface{})
	if !ok || len(operands) != 2 {
		t.Fatal("InIntervalOfInterval: expected two operands")
	}
	rhs, _ := operands[1].(map[string]interface{})
	if got := nodeType(t, rhs); got != "ToList" {
		t.Errorf("InIntervalOfInterval: right operand type = %q, want ToList", got)
	}
}

// TestPointInclusionDemotion pins that `included in` / `during` select the
// point-in-interval operators when the left operand is not itself an interval.
func TestPointInclusionDemotion(t *testing.T) {
	r := translate(t, `library T version '1.0'
define Period: Interval[@2024-01-01, @2024-12-31]
define Day: @2024-06-01
define PointIncludedIn: Day included in Period
define PointDuring: Day during Period
define PointProperlyIncludedIn: Day properly included in Period
define IntervalIncludedIn: Interval[@2024-02-01, @2024-03-01] included in Period
define IntervalProperlyIncludedIn: Interval[@2024-02-01, @2024-03-01] properly included in Period`)

	for _, c := range []struct{ def, want string }{
		{"PointIncludedIn", "In"},
		{"PointDuring", "In"},
		{"PointProperlyIncludedIn", "ProperIn"},
		{"IntervalIncludedIn", "IncludedIn"},
		{"IntervalProperlyIncludedIn", "ProperIncludedIn"},
	} {
		if got := nodeType(t, exprOf(t, r, c.def)); got != c.want {
			t.Errorf("%s: type = %q, want %q", c.def, got, c.want)
		}
	}
}

// TestQuotedIdentifierQuerySource pins that a quoted identifier used as a query
// source is unquoted like every other reference position.
func TestQuotedIdentifierQuerySource(t *testing.T) {
	r := translate(t, `library T version '1.0'
define "My Items": { 1, 2, 3 }
define "InQuery": "My Items" I return I`)

	sources, ok := exprOf(t, r, "InQuery")["source"].([]interface{})
	if !ok || len(sources) != 1 {
		t.Fatal("expected one query source")
	}
	src, _ := sources[0].(map[string]interface{})
	expr, _ := src["expression"].(map[string]interface{})
	name, _ := expr["name"].(string)
	if name != "My Items" {
		t.Errorf("query source name = %q, want %q", name, "My Items")
	}
}

// TestAggregateClause pins the starting expression (which the grammar allows as
// a bare simpleLiteral), the accumulator's QueryLetRef scope, and the distinct
// flag for the all / distinct / plain forms.
func TestAggregateClause(t *testing.T) {
	r := translate(t, `library T version '1.0'
define Items: { 1, 2, 3 }
define Plain: Items I aggregate Total starting 0: Total + I
define All: Items I aggregate all Acc starting 1: Acc * I
define Distinct: Items I aggregate distinct Acc starting 1: Acc * I`)

	agg, ok := exprOf(t, r, "Plain")["aggregate"].(map[string]interface{})
	if !ok {
		t.Fatal("Plain: expected aggregate clause")
	}
	starting, _ := agg["starting"].(map[string]interface{})
	if got := nodeType(t, starting); got != "Literal" {
		t.Errorf("Plain: starting type = %q, want Literal", got)
	}
	if v, _ := starting["value"].(string); v != "0" {
		t.Errorf("Plain: starting value = %q, want 0", v)
	}
	if _, present := agg["distinct"]; present {
		t.Error("Plain: distinct should be omitted for a plain aggregate")
	}
	// The accumulator is in scope inside the body and resolves to a QueryLetRef.
	body, _ := agg["expression"].(map[string]interface{})
	operands, _ := body["operand"].([]interface{})
	if len(operands) != 2 {
		t.Fatalf("Plain: expected two body operands, got %d", len(operands))
	}
	acc, _ := operands[0].(map[string]interface{})
	if got := nodeType(t, acc); got != "QueryLetRef" {
		t.Errorf("Plain: accumulator type = %q, want QueryLetRef", got)
	}

	for _, c := range []struct {
		def  string
		want bool
	}{
		{"All", false},
		{"Distinct", true},
	} {
		agg, ok := exprOf(t, r, c.def)["aggregate"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: expected aggregate clause", c.def)
		}
		got, present := agg["distinct"].(bool)
		if !present {
			t.Errorf("%s: distinct missing, want %v", c.def, c.want)
			continue
		}
		if got != c.want {
			t.Errorf("%s: distinct = %v, want %v", c.def, got, c.want)
		}
	}
}

// TestReturnDistinct pins that an explicit return all / return distinct emits
// the flag, while a plain return omits it.
func TestReturnDistinct(t *testing.T) {
	r := translate(t, `library T version '1.0'
define Items: { 1, 2, 3 }
define Plain: Items I return I
define All: Items I return all I
define Distinct: Items I return distinct I`)

	for _, c := range []struct {
		def     string
		present bool
		want    bool
	}{
		{"Plain", false, false},
		{"All", true, false},
		{"Distinct", true, true},
	} {
		ret, ok := exprOf(t, r, c.def)["return"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: expected return clause", c.def)
		}
		got, present := ret["distinct"].(bool)
		if present != c.present {
			t.Errorf("%s: distinct present = %v, want %v", c.def, present, c.present)
			continue
		}
		if present && got != c.want {
			t.Errorf("%s: distinct = %v, want %v", c.def, got, c.want)
		}
	}
}

// TestSortByExpressionUsesIdentifierRef pins that identifiers in a sort-by
// expression stay unresolved: their scope is the query's result element, which
// the engine resolves at evaluation time.
func TestSortByExpressionUsesIdentifierRef(t *testing.T) {
	r := translate(t, `library T version '1.0'
define Items: { Tuple { p: Interval[@2024-01-01, @2024-02-01] } }
define Sorted: Items I sort by start of p`)

	sort, ok := exprOf(t, r, "Sorted")["sort"].(map[string]interface{})
	if !ok {
		t.Fatal("expected sort clause")
	}
	by, _ := sort["by"].([]interface{})
	if len(by) != 1 {
		t.Fatalf("expected one sort item, got %d", len(by))
	}
	item, _ := by[0].(map[string]interface{})
	expr, _ := item["expression"].(map[string]interface{})
	operand, _ := expr["operand"].(map[string]interface{})
	if got := nodeType(t, operand); got != "IdentifierRef" {
		t.Errorf("sort-by identifier type = %q, want IdentifierRef", got)
	}
	if name, _ := operand["name"].(string); name != "p" {
		t.Errorf("sort-by identifier name = %q, want p", name)
	}
}

// TestConceptDeclaration pins that a concept's code list holds bare unquoted
// names with no "type" discriminator, and that the display clause is captured.
func TestConceptDeclaration(t *testing.T) {
	r := translate(t, `library T version '1.0'
codesystem "SNOMED": 'http://snomed.info/sct'
code "Diabetes": '73211009' from "SNOMED" display 'Diabetes mellitus'
concept "Diabetes Concept": { "Diabetes" } display 'Diabetes concept'`)

	if r.Library.Concepts == nil || len(r.Library.Concepts.Def) != 1 {
		t.Fatal("expected one concept def")
	}
	def := r.Library.Concepts.Def[0]
	if def.Display != "Diabetes concept" {
		t.Errorf("display = %q, want %q", def.Display, "Diabetes concept")
	}
	if len(def.Code) != 1 {
		t.Fatalf("expected one code ref, got %d", len(def.Code))
	}
	if def.Code[0].Name != "Diabetes" {
		t.Errorf("code name = %q, want Diabetes (unquoted)", def.Code[0].Name)
	}
	b, err := json.Marshal(def.Code[0])
	if err != nil {
		t.Fatalf("marshal code ref: %v", err)
	}
	if strings.Contains(string(b), `"type"`) {
		t.Errorf("concept code ref carries a type discriminator: %s", b)
	}
}

// TestValueSetRefPreserve pins that every ValueSetRef expression is stamped
// preserve=true under CQL 1.5, and that compatibility level 1.4 suppresses it.
func TestValueSetRefPreserve(t *testing.T) {
	cql := `library T version '1.0'
valueset "VS": 'urn:oid:1.2.3'
define BareRef: "VS"`

	e := exprOf(t, translate(t, cql), "BareRef")
	if nodeType(t, e) != "ValueSetRef" {
		t.Fatalf("type = %q, want ValueSetRef", nodeType(t, e))
	}
	if preserve, _ := e["preserve"].(bool); !preserve {
		t.Error("preserve = false, want true at CQL 1.5")
	}

	e = exprOf(t, translate(t, cql, func(o *translator.Options) {
		o.CompatibilityLevel = "1.4"
	}), "BareRef")
	if _, present := e["preserve"]; present {
		t.Error("preserve emitted at compatibility level 1.4")
	}
}

// TestFHIRPropertyCoverage pins that the generated ModelInfo tables resolve
// properties the previously hand-maintained subset omitted, including binding
// types and properties reached through a query alias over a property source.
func TestFHIRPropertyCoverage(t *testing.T) {
	r := translate(t, `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define "LotNumber": exists ([Immunization] I where I.lotNumber = 'X')
define "Gender": Patient.gender = 'female'
define "City": exists ((Patient.address) A where A.city = 'Boston')`)

	for _, c := range []struct{ def, wantFn string }{
		{"LotNumber", "ToString"},
		{"Gender", "ToString"},
		{"City", "ToString"},
	} {
		found := false
		var walk func(interface{})
		walk = func(o interface{}) {
			switch v := o.(type) {
			case map[string]interface{}:
				if v["type"] == "FunctionRef" && v["name"] == c.wantFn && v["libraryName"] == "FHIRHelpers" {
					found = true
				}
				for _, child := range v {
					walk(child)
				}
			case []interface{}:
				for _, child := range v {
					walk(child)
				}
			}
		}
		walk(exprOf(t, r, c.def))
		if !found {
			t.Errorf("%s: expected a FHIRHelpers.%s conversion", c.def, c.wantFn)
		}
	}
}

// TestFHIRListPropertyNotCoerced pins that a list-valued FHIR primitive is left
// alone. CQF lifts the conversion into a per-element query, so wrapping the list
// itself would be wrong.
func TestFHIRListPropertyNotCoerced(t *testing.T) {
	r := translate(t, `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define "Lines": (Patient.address) A return A.line`)

	ret, ok := exprOf(t, r, "Lines")["return"].(map[string]interface{})
	if !ok {
		t.Fatal("expected return clause")
	}
	expr, _ := ret["expression"].(map[string]interface{})
	if got := nodeType(t, expr); got != "Property" {
		t.Errorf("list-valued property type = %q, want an uncoerced Property", got)
	}
}

// TestDateToDateTimePromotion pins CQL's implicit Date→DateTime conversion in a
// comparison, and that two Dates are compared as Dates.
func TestDateToDateTimePromotion(t *testing.T) {
	r := translate(t, `library T version '1.0'
define DT: @2020-06-01T10:00:00
define D: @2020-01-01
define "MixedLeft": DT before D
define "MixedRight": D before DT
define "BothDates": D before D
define "InDateInterval": DT in Interval[@2020-01-01, @2020-12-31]`)

	operands, _ := exprOf(t, r, "MixedLeft")["operand"].([]interface{})
	rhs, _ := operands[1].(map[string]interface{})
	if got := nodeType(t, rhs); got != "ToDateTime" {
		t.Errorf("MixedLeft: right operand = %q, want ToDateTime", got)
	}

	operands, _ = exprOf(t, r, "MixedRight")["operand"].([]interface{})
	lhs, _ := operands[0].(map[string]interface{})
	if got := nodeType(t, lhs); got != "ToDateTime" {
		t.Errorf("MixedRight: left operand = %q, want ToDateTime", got)
	}

	operands, _ = exprOf(t, r, "BothDates")["operand"].([]interface{})
	for i, o := range operands {
		m, _ := o.(map[string]interface{})
		if got := nodeType(t, m); got == "ToDateTime" {
			t.Errorf("BothDates: operand %d was promoted; two Dates compare as Dates", i)
		}
	}

	// The interval literal is promoted through its bounds, not as a whole.
	operands, _ = exprOf(t, r, "InDateInterval")["operand"].([]interface{})
	iv, _ := operands[1].(map[string]interface{})
	if got := nodeType(t, iv); got != "Interval" {
		t.Fatalf("InDateInterval: right operand = %q, want Interval", got)
	}
	for _, bound := range []string{"low", "high"} {
		m, _ := iv[bound].(map[string]interface{})
		if got := nodeType(t, m); got != "ToDateTime" {
			t.Errorf("InDateInterval: %s bound = %q, want ToDateTime", bound, got)
		}
	}
}

// TestTranslatorOptionsReporting pins the CqlToElmInfo contract: the option
// names and their order mirror what CQF serializes, and compatibilityLevel is
// not reported at all.
func TestTranslatorOptionsReporting(t *testing.T) {
	r := translate(t, "library T version '1.0'", func(o *translator.Options) {
		o.CQFMode = true
		o.EnableAnnotations = true
		o.EnableLocators = true
		o.DisableListTraversal = true
		o.DisableListDemotion = true
		o.RequireFromKeyword = true
		o.CompatibilityLevel = "1.4"
	})
	if len(r.Library.Annotation) == 0 {
		t.Fatal("expected library annotation")
	}
	var info map[string]interface{}
	if err := json.Unmarshal(r.Library.Annotation[0], &info); err != nil {
		t.Fatalf("unmarshal CqlToElmInfo: %v", err)
	}
	want := "EnableAnnotations,EnableLocators,DisableListTraversal,DisableListDemotion,RequireFromKeyword"
	if got, _ := info["translatorOptions"].(string); got != want {
		t.Errorf("translatorOptions:\n got %q\nwant %q", got, want)
	}
	if _, present := info["compatibilityLevel"]; present {
		t.Error("compatibilityLevel should not be reported in CqlToElmInfo")
	}
}

// TestStatementEmissionOrder pins that a define is emitted before the defines
// that reference it, matching CQF's lazy resolution, even when the source
// declares it later.
func TestStatementEmissionOrder(t *testing.T) {
	r := translate(t, `library T version '1.0'
define "Uses Later": "Declared Later" + 1
define "Declared Later": 41
define "Independent": 0`)

	order := make([]string, 0, len(r.Library.Statements.Def))
	for _, def := range r.Library.Statements.Def {
		order = append(order, def.Name)
	}
	want := []string{"Declared Later", "Uses Later", "Independent"}
	if len(order) != len(want) {
		t.Fatalf("statement count = %d, want %d (%v)", len(order), len(want), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("emission order = %v, want %v", order, want)
		}
	}
}

// TestFHIRPrimitiveListConversion pins that a list-valued FHIR primitive
// consumed by `in` is converted element by element through an implicit query.
// Promoting it with ToList would yield a List<List<T>>.
func TestFHIRPrimitiveListConversion(t *testing.T) {
	r := translate(t, `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define "LineIn": exists ((Patient.address) A where 'Main St' in A.line)`)

	operands, _ := exprOf(t, r, "LineIn")["operand"].(map[string]interface{})["where"].(map[string]interface{})["operand"].([]interface{})
	if len(operands) != 2 {
		t.Fatalf("expected two In operands, got %d", len(operands))
	}
	rhs, _ := operands[1].(map[string]interface{})
	if got := nodeType(t, rhs); got != "Query" {
		t.Fatalf("right operand = %q, want an implicit conversion Query", got)
	}
	sources, _ := rhs["source"].([]interface{})
	if len(sources) != 1 {
		t.Fatalf("expected one query source, got %d", len(sources))
	}
	if alias, _ := sources[0].(map[string]interface{})["alias"].(string); alias != "X" {
		t.Errorf("conversion query alias = %q, want X", alias)
	}
	ret, _ := rhs["return"].(map[string]interface{})
	if distinct, ok := ret["distinct"].(bool); !ok || distinct {
		t.Errorf("conversion query return distinct = %v, want false", ret["distinct"])
	}
	fn, _ := ret["expression"].(map[string]interface{})
	if nodeType(t, fn) != "FunctionRef" || fn["name"] != "ToString" {
		t.Errorf("conversion expression = %v, want FHIRHelpers.ToString", fn)
	}
}

// TestResultTypesGatedOnOption pins that result types appear only under
// --result-types, and that a declaration carries the system type CQF records.
func TestResultTypesGatedOnOption(t *testing.T) {
	cql := `library T version '1.0'
codesystem "CS": 'http://example.org'
valueset "VS": 'urn:oid:1.2.3'
define "N": 1 + 2`

	off := marshalDef(t, translate(t, cql), "N")
	if _, present := off["resultTypeName"]; present {
		t.Error("resultTypeName emitted without --result-types")
	}

	r := translate(t, cql, func(o *translator.Options) { o.EnableResultTypes = true })
	on := marshalDef(t, r, "N")
	if got, _ := on["resultTypeName"].(string); got != "{urn:hl7-org:elm-types:r1}Integer" {
		t.Errorf("definition resultTypeName = %q, want Integer", got)
	}
	expr, _ := on["expression"].(map[string]interface{})
	if got, _ := expr["resultTypeName"].(string); got != "{urn:hl7-org:elm-types:r1}Integer" {
		t.Errorf("expression resultTypeName = %q, want Integer", got)
	}
	if r.Library.CodeSystems.Def[0].ResultTypeName != "{urn:hl7-org:elm-types:r1}CodeSystem" {
		t.Errorf("codesystem resultTypeName = %q", r.Library.CodeSystems.Def[0].ResultTypeName)
	}
	if r.Library.ValueSets.Def[0].ResultTypeName != "{urn:hl7-org:elm-types:r1}ValueSet" {
		t.Errorf("valueset resultTypeName = %q", r.Library.ValueSets.Def[0].ResultTypeName)
	}
}

// TestResultTypesSkipSynthesizedNodes pins that CQF's model is followed: a node
// the translator synthesizes carries no result type, while the source node it
// wraps does. A FHIR property reports its FHIR type, not the converted one.
func TestResultTypesSkipSynthesizedNodes(t *testing.T) {
	r := translate(t, `library T version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient
define "G": Patient.gender = 'female'`, func(o *translator.Options) {
		o.EnableResultTypes = true
	})

	operands, _ := exprOf(t, r, "G")["operand"].([]interface{})
	conv, _ := operands[0].(map[string]interface{})
	if nodeType(t, conv) != "FunctionRef" {
		t.Fatalf("expected a FHIRHelpers conversion, got %q", nodeType(t, conv))
	}
	if _, present := conv["resultTypeName"]; present {
		t.Error("the synthesized conversion should carry no result type")
	}
	inner, _ := conv["operand"].([]interface{})
	prop, _ := inner[0].(map[string]interface{})
	if got, _ := prop["resultTypeName"].(string); got != "{http://hl7.org/fhir}AdministrativeGender" {
		t.Errorf("property resultTypeName = %q, want the declared FHIR type", got)
	}

	// The implicit context accessor is synthesized end to end.
	patient := marshalDef(t, r, "Patient")
	if _, present := patient["resultTypeName"]; present {
		t.Error("the implicit context accessor should carry no result type")
	}
}

// TestResultTypeInference covers the inference rules that the debug-profile
// parity run exercises, so a regression shows up here as well as in the goldens.
func TestResultTypeInference(t *testing.T) {
	rt := func(t *testing.T, r *translator.Result, def string) string {
		t.Helper()
		m := marshalDef(t, r, def)
		if name, ok := m["resultTypeName"].(string); ok && name != "" {
			return strings.TrimPrefix(name, "{urn:hl7-org:elm-types:r1}")
		}
		b, _ := json.Marshal(m["resultTypeSpecifier"])
		return string(b)
	}
	withRT := func(o *translator.Options) { o.EnableResultTypes = true }

	t.Run("operators", func(t *testing.T) {
		r := translate(t, `library T version '1.0'
define "Nums": { 1, 2, 3 }
define "MinOf": Min("Nums")
define "AvgOf": Avg("Nums")
define "CountOf": Count("Nums")
define "FloorOf": Floor(1.7)
define "UpperOf": Upper('a')
define "IndexOfIt": IndexOf("Nums", 2)
define "WidthOf": width of Interval[1, 5]
define "IsIt": 1 is Integer`, withRT)

		for _, c := range []struct{ def, want string }{
			{"MinOf", "Integer"}, {"AvgOf", "Decimal"}, {"CountOf", "Integer"},
			{"FloorOf", "Integer"}, {"UpperOf", "String"}, {"IndexOfIt", "Integer"},
			{"WidthOf", "Integer"}, {"IsIt", "Boolean"},
		} {
			if got := rt(t, r, c.def); got != c.want {
				t.Errorf("%s: result type = %s, want %s", c.def, got, c.want)
			}
		}
	})

	t.Run("null typing", func(t *testing.T) {
		r := translate(t, `library T version '1.0'
define "CoalesceAny": Coalesce(null, null, 42)
define "PlainNull": null`, withRT)

		// An untyped null among the alternatives leaves no common type but Any.
		if got := rt(t, r, "CoalesceAny"); got != "Any" {
			t.Errorf("CoalesceAny: result type = %s, want Any", got)
		}
		if got := rt(t, r, "PlainNull"); got != "Any" {
			t.Errorf("PlainNull: result type = %s, want Any", got)
		}
	})

	t.Run("context scoped reference", func(t *testing.T) {
		// A definition declared in a retrieve context yields one value per
		// context member when referenced from Unfiltered.
		r := translate(t, `library T version '1.0'
using QDM version '5.3'
context Patient
define "InPop": true
context Unfiltered
define "AllPops": "InPop"`, withRT)

		if got := rt(t, r, "InPop"); got != "Boolean" {
			t.Errorf("InPop: result type = %s, want Boolean", got)
		}
		if got := rt(t, r, "AllPops"); !strings.Contains(got, "ListTypeSpecifier") {
			t.Errorf("AllPops: result type = %s, want a List specifier", got)
		}
	})

	t.Run("query clauses", func(t *testing.T) {
		r := translate(t, `library T version '1.0'
define "Items": { Tuple { id: 1, score: 10 } }
define "Scores": "Items" I return I.score
define "Total": "Items" I aggregate Sum starting 0: Sum + I.score`, withRT)

		if got := rt(t, r, "Scores"); !strings.Contains(got, "Integer") {
			t.Errorf("Scores: result type = %s, want List<Integer>", got)
		}
		// An aggregate clause reduces the query to a scalar.
		if got := rt(t, r, "Total"); got != "Integer" {
			t.Errorf("Total: result type = %s, want Integer", got)
		}
	})
}
