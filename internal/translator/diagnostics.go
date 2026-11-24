package translator

// CqlToElmError diagnostics recorded in the ELM, as CQF records them.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// overloadWarning is CQF's text for a call to an overloaded function whose
// signature is left out of the ELM.
const overloadWarning = "The function %s.%s has multiple overloads and due to the SignatureLevel setting (%s), " +
	"the overload signature is not being included in the output. This may result in ambiguous function " +
	"resolution at runtime, consider setting the SignatureLevel to Overloads or All to ensure that the " +
	"output includes sufficient information to support correct overload selection at runtime."

// noteOverloadedCall records CQF's signature-omitted warning for a call to an
// overloaded function left without a signature — one per call, including the
// FHIRHelpers conversions the translator inserts. That is every such call at
// SignatureLevel None; at Differing, callers note only the calls they leave
// unsigned. It carries no location, as CQF's does not. (#8)
func (t *Translator) noteOverloadedCall(libraryAlias, name string) {
	level := t.opts.SignatureLevel
	if level == "" {
		level = "None"
	}
	if !t.opts.CQFMode || (level != "None" && level != "Differing") {
		return
	}
	var libName string
	var overloaded bool
	switch {
	case libraryAlias == "":
		libName, overloaded = t.libraryID, t.localFuncCounts[name] > 1
	case t.libFuncCounts[libraryAlias] != nil:
		libName, overloaded = t.includeLibraryNames[libraryAlias], t.libFuncCounts[libraryAlias][name] > 1
	case t.includeLibraryNames[libraryAlias] == "FHIRHelpers":
		// Not resolvable from here; CQF resolves it from its classpath.
		libName, overloaded = "FHIRHelpers", typesystem.FHIRHelpersOverloaded[name]
	}
	if !overloaded || libName == "" {
		return
	}
	t.elmDiagnostics = append(t.elmDiagnostics, &elm.CqlToElmError{
		Message:       fmt.Sprintf(overloadWarning, libName, name, level),
		ErrorType:     "semantic",
		ErrorSeverity: "warning",
	})
}

// isEmptySig reports a signature field that records no operand types.
func isEmptySig(sig json.RawMessage) bool {
	s := strings.TrimSpace(string(sig))
	return s == "" || s == "[]" || s == "null"
}

// noteListDemotion records CQF's warning for a list operand demoted to a
// singleton. CQF emits it while resolving a list-to-list inclusion — it tries
// the point form, demoting the operand in the point position — even though it
// then settles on the list form, so the warning accompanies ELM that contains
// no demotion. Only while list demotion is enabled. (#8)
func (t *Translator) noteListDemotion(loc ast.Interval) {
	if t.opts.DisableListDemotion {
		return
	}
	t.noteDemotion(loc, "List-valued expression was demoted to a singleton.")
}

// noteIntervalDemotion is noteListDemotion for an interval demoted to a point,
// which CQF tries the same way, only while interval demotion is enabled. (#8)
func (t *Translator) noteIntervalDemotion(loc ast.Interval) {
	if !t.opts.EnableIntervalDemotion {
		return
	}
	t.noteDemotion(loc, "Interval-valued expression was demoted to a point.")
}

func (t *Translator) noteDemotion(loc ast.Interval, message string) {
	if !t.opts.CQFMode || loc.Start.Line == 0 {
		return
	}
	t.elmDiagnostics = append(t.elmDiagnostics, &elm.CqlToElmError{
		LibraryID:      t.libraryID,
		LibraryVersion: t.libraryVersion,
		StartLine:      loc.Start.Line,
		StartChar:      loc.Start.Column,
		EndLine:        loc.Stop.Line,
		EndChar:        loc.Stop.Column,
		Message:        message,
		ErrorType:      "semantic",
		ErrorSeverity:  "warning",
	})
}

// noteInclusionDemotion applies noteListDemotion to a list-to-list inclusion,
// and noteIntervalDemotion to an interval-to-interval one: the operand in the
// point position is the one CQF tried to demote.
func (t *Translator) noteInclusionDemotion(op string, left, right ast.Expr, lhs, rhs elm.Expression) {
	lts, rts := t.inferTypeSpec(lhs), t.inferTypeSpec(rhs)
	note := t.noteListDemotion
	switch {
	case isListTS(lts) && isListTS(rts):
	case isIntervalTS(lts) && isIntervalTS(rts):
		note = t.noteIntervalDemotion
	default:
		return
	}
	// Only against a collection whose element type is known: the point form is
	// tried by matching the demoted operand to that element type, so with two
	// untyped lists — `{} includes {}` — CQF never tries it and never warns.
	switch op {
	case "IncludedIn", "ProperIncludedIn":
		if elementTypeOf(t.typeOf(rhs)) != nil {
			note(left.Loc())
		}
	case "Includes", "ProperIncludes":
		if elementTypeOf(t.typeOf(lhs)) != nil {
			note(right.Loc())
		}
	}
}

func isIntervalTS(ts typeSpec) bool {
	_, ok := ts.(intervalTS)
	return ok
}

// noteInclusionPromotion records the second overload warning CQF emits for an
// interval inclusion — `during`, `included in`, `includes` and their proper
// forms — where exactly one operand is an implicit FHIRHelpers conversion.
// With list promotion on, CQF first resolves the point form against the other
// interval promoted to a list, applying the conversion, then discards it and
// resolves the interval form, applying it again. When both operands are
// converted the point form does not resolve, and an explicit FHIRHelpers call
// was resolved once, before the operator. (#8)
func (t *Translator) noteInclusionPromotion(op string, lhs, rhs elm.Expression) {
	if t.opts.DisableListPromotion {
		return
	}
	switch op {
	case "IncludedIn", "ProperIncludedIn", "Includes", "ProperIncludes":
	default:
		return
	}
	if !isIntervalTS(t.inferTypeSpec(lhs)) || !isIntervalTS(t.inferTypeSpec(rhs)) {
		return
	}
	lf, rf := t.implicitFHIRConversion(lhs), t.implicitFHIRConversion(rhs)
	switch {
	case lf != nil && rf == nil:
		t.noteOverloadedCall(t.fhirHelpersLocalName, lf.Name)
	case rf != nil && lf == nil:
		t.noteOverloadedCall(t.fhirHelpersLocalName, rf.Name)
	}
}

// implicitFHIRConversion returns e when it is a FHIRHelpers conversion the
// translator inserted, or nil.
func (t *Translator) implicitFHIRConversion(e elm.Expression) *elm.FunctionRefNode {
	fr, ok := e.(*elm.FunctionRefNode)
	if !ok || !t.converted[e] || t.fhirHelpersLocalName == "" || fr.LibraryName != t.fhirHelpersLocalName {
		return nil
	}
	return fr
}
