package translator

import (
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"

	"github.com/artnerc/echo-elm/internal/typesystem"
)

// funcOverload is one declaration a function call might resolve to.
//
// The earlier bookkeeping kept operand and return types for unambiguous names
// only, on the reasoning that picking between overloads needs argument types.
// That is right, but it meant the cases that most need a type — an overloaded
// helper, or one fluent name declared by several includes — got none. CQF
// resolves them by argument type, so this records every overload and selectOverload
// does the same. (issues/08 W4, #13)
type funcOverload struct {
	// operands are the declared operand types; a nil entry is undeclared.
	operands []typeSpec
	// returns is the declared return type, or else the type inferred from the
	// body once the definition has been translated. Nil until then.
	returns typeSpec
	fluent  bool
	// library is the include alias that declares it, "" for this library.
	library string
}

// recordLocalOverload registers a function definition of this library.
func (t *Translator) recordLocalOverload(name string, ov *funcOverload) {
	if t.localOverloads == nil {
		t.localOverloads = map[string][]*funcOverload{}
	}
	t.localOverloads[name] = append(t.localOverloads[name], ov)
}

// callCandidates returns the overloads a call to lib.name may resolve to.
func (t *Translator) callCandidates(lib, name string) []*funcOverload {
	if lib == "" {
		return t.localOverloads[name]
	}
	return t.libOverloads[lib][name]
}

// overloadForCall resolves lib.name applied to args.
func (t *Translator) overloadForCall(lib, name string, args []typeSpec) *funcOverload {
	cands := t.callCandidates(lib, name)
	switch len(cands) {
	case 0:
		return nil
	case 1:
		// A single declaration is the target whatever the arguments look like;
		// the argument types here are approximate and must not veto it.
		return cands[0]
	}
	return selectOverload(cands, args)
}

// fluentLibraryByArgs resolves a fluent name declared by more than one include,
// choosing the library whose overload best fits the arguments. The receiver is
// args[0]. Returns "" when no single overload wins.
func (t *Translator) fluentLibraryByArgs(name string, args []typeSpec) string {
	var cands []*funcOverload
	for _, lib := range t.libFluentFuncs[name] {
		for _, ov := range t.libOverloads[lib][name] {
			if ov.fluent {
				cands = append(cands, ov)
			}
		}
	}
	if ov := selectOverload(cands, args); ov != nil {
		return ov.library
	}
	return ""
}

// selectOverload returns the candidate whose operands best fit args, or nil on
// no fit or a tie. A tie is left unresolved rather than broken arbitrarily: a
// confidently wrong target is worse than an unresolved one.
func selectOverload(cands []*funcOverload, args []typeSpec) *funcOverload {
	var best *funcOverload
	bestScore, tie := -1, false
	for _, c := range cands {
		if len(c.operands) != len(args) {
			continue
		}
		score, ok := overloadScore(c.operands, args)
		if !ok {
			continue
		}
		switch {
		case score > bestScore:
			best, bestScore, tie = c, score, false
		case score == bestScore:
			tie = true
		}
	}
	if tie {
		return nil
	}
	return best
}

func overloadScore(declared, args []typeSpec) (int, bool) {
	total := 0
	for i := range declared {
		s := operandFit(declared[i], args[i])
		if s < 0 {
			return 0, false
		}
		total += s
	}
	return total, true
}

// Operand fit, best first. An unknown argument fits weakly rather than not at
// all: it cannot rule an overload out, but it must not beat a known match.
const (
	fitExact   = 4
	fitSubtype = 3
	fitConvert = 2
	fitWeak    = 1
	fitNone    = -1
)

// operandFit scores how well an argument of type arg satisfies an operand
// declared as decl.
func operandFit(decl, arg typeSpec) int {
	if decl == nil {
		return fitWeak
	}
	if knownTS(arg) == nil {
		return fitWeak
	}
	if typeSpecEqual(decl, arg) {
		return fitExact
	}
	if knownTS(decl) == nil {
		return fitWeak // declared Any accepts anything, least specifically
	}
	switch d := decl.(type) {
	case listTS:
		a, ok := arg.(listTS)
		if !ok {
			return fitNone
		}
		return min(operandFit(d.elem, a.elem), fitSubtype)
	case intervalTS:
		a, ok := arg.(intervalTS)
		if !ok {
			return fitNone
		}
		return min(operandFit(d.point, a.point), fitSubtype)
	case namedTS:
		a, ok := arg.(namedTS)
		if !ok {
			return fitNone
		}
		if dl, al := fhirLocalTypeName(d), fhirLocalTypeName(a); dl != "" && al != "" {
			if fhirDerivesFrom(al, dl) {
				return fitSubtype
			}
			return fitNone
		}
		if implicitlyConvertible(a, d) {
			return fitConvert
		}
		// A FHIR primitive argument converts to its System type through
		// FHIRHelpers, as CQF does when matching a System-typed operand.
		if al := fhirLocalTypeName(a); al != "" {
			if sys := fhirPrimitiveSystemType(al); sys != "" && sys == d.name {
				return fitConvert
			}
		}
		return fitNone
	}
	return fitWeak
}

// fhirPrimitiveSystemType is the System type a FHIR primitive converts to, e.g.
// dateTime → {urn:hl7-org:elm-types:r1}DateTime, or "" for anything else.
func fhirPrimitiveSystemType(local string) string {
	sys := typesystem.FHIRPrimitiveValueType[local]
	if sys == "" {
		return ""
	}
	if strings.HasPrefix(sys, "{") {
		return sys
	}
	return typesystem.SystemELMName + sys
}

// localFunctionFits reports whether a function this library defines under name
// accepts the arguments' FHIR types. Only FHIR-typed arguments are considered:
// that is where a local definition and a System operator of the same name part
// ways, since System operators take no FHIR model types.
func (t *Translator) localFunctionFits(name string, args []ast.Expr) bool {
	if t.localFuncCounts[name] == 0 {
		return false
	}
	types := make([]typeSpec, len(args))
	for i, a := range args {
		types[i] = t.astFHIRArgType(a)
		if types[i] == nil {
			return false
		}
	}
	return selectOverload(t.localOverloads[name], types) != nil
}

// astFHIRArgType is the FHIR model type of an untranslated argument, when it can
// be read off the syntax: a FHIR property, a typed function operand, or a query
// alias. Nil otherwise.
func (t *Translator) astFHIRArgType(e ast.Expr) typeSpec {
	switch v := e.(type) {
	case *ast.PropertyExpr:
		typ := t.resolveFHIRPropertyType(v.Source, v.Path)
		if typ == "" {
			return nil
		}
		var ts typeSpec = namedTS{"{http://hl7.org/fhir}" + typ}
		if src := t.resolveFHIRSourceType(v.Source); src != "" && typesystem.IsFHIRListPropertyOf(src, v.Path) {
			ts = listTS{ts}
		}
		return ts
	case *ast.IdentifierRef:
		if t.functionParamScope[v.Name] {
			if fhirLocalTypeName(t.operandTypeSpecs[v.Name]) != "" {
				return t.operandTypeSpecs[v.Name]
			}
			return nil
		}
		if typ := t.resolveFHIRSourceType(v); typ != "" {
			return namedTS{"{http://hl7.org/fhir}" + typ}
		}
	case *ast.AliasRef:
		if typ := t.resolveFHIRSourceType(v); typ != "" {
			return namedTS{"{http://hl7.org/fhir}" + typ}
		}
	case *ast.TypeAsExpr:
		// `value as Quantity` has exactly the type it names.
		if v.TypeSpec != nil {
			if ts := t.declaredTypeSpec(v.TypeSpec); fhirLocalTypeName(ts) != "" {
				return ts
			}
		}
	}
	return nil
}
