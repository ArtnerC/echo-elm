package translator

// CqlToElmError diagnostics recorded in the ELM, as CQF records them.

import (
	"fmt"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// overloadWarning is CQF's text for a call to an overloaded function whose
// signature is left out of the ELM.
const overloadWarning = "The function %s.%s has multiple overloads and due to the SignatureLevel setting (None), " +
	"the overload signature is not being included in the output. This may result in ambiguous function " +
	"resolution at runtime, consider setting the SignatureLevel to Overloads or All to ensure that the " +
	"output includes sufficient information to support correct overload selection at runtime."

// noteOverloadedCall records CQF's signature-omitted warning for a call to an
// overloaded function at SignatureLevel None — one per call, including the
// FHIRHelpers conversions the translator inserts. It carries no location, as
// CQF's does not. (#8)
func (t *Translator) noteOverloadedCall(libraryAlias, name string) {
	if !t.opts.CQFMode || (t.opts.SignatureLevel != "" && t.opts.SignatureLevel != "None") {
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
		Message:       fmt.Sprintf(overloadWarning, libName, name),
		ErrorType:     "semantic",
		ErrorSeverity: "warning",
	})
}

// noteListDemotion records CQF's warning for a list operand demoted to a
// singleton. CQF emits it while resolving a list-to-list inclusion — it tries
// the point form, demoting the operand in the point position — even though it
// then settles on the list form, so the warning accompanies ELM that contains
// no demotion. Only while list demotion is enabled. (#8)
func (t *Translator) noteListDemotion(loc ast.Interval) {
	if !t.opts.CQFMode || t.opts.DisableListDemotion || loc.Start.Line == 0 {
		return
	}
	t.elmDiagnostics = append(t.elmDiagnostics, &elm.CqlToElmError{
		LibraryID:      t.libraryID,
		LibraryVersion: t.libraryVersion,
		StartLine:      loc.Start.Line,
		StartChar:      loc.Start.Column,
		EndLine:        loc.Stop.Line,
		EndChar:        loc.Stop.Column,
		Message:        "List-valued expression was demoted to a singleton.",
		ErrorType:      "semantic",
		ErrorSeverity:  "warning",
	})
}

// noteInclusionDemotion applies noteListDemotion to a list-to-list inclusion:
// the operand in the point position is the one CQF tried to demote.
func (t *Translator) noteInclusionDemotion(op string, left, right ast.Expr, lhs, rhs elm.Expression) {
	if !isListTS(t.inferTypeSpec(lhs)) || !isListTS(t.inferTypeSpec(rhs)) {
		return
	}
	// Only against a collection whose element type is known: the point form is
	// tried by matching the demoted operand to that element type, so with two
	// untyped lists — `{} includes {}` — CQF never tries it and never warns.
	switch op {
	case "IncludedIn", "ProperIncludedIn":
		if elementTypeOf(t.typeOf(rhs)) != nil {
			t.noteListDemotion(left.Loc())
		}
	case "Includes", "ProperIncludes":
		if elementTypeOf(t.typeOf(lhs)) != nil {
			t.noteListDemotion(right.Loc())
		}
	}
}
