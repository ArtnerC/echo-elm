package translator

// FHIR model support: FHIRHelpers conversions, list lifts, age functions and retrieve helpers.

import (
	"encoding/json"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// fhirLocalTypeName returns the local FHIR type name of a type spec — the
// "Encounter" in "{http://hl7.org/fhir}Encounter" — or "" when the spec is not a
// namespaced model type. System types are excluded: they are already CQL values
// and need no conversion.
func fhirLocalTypeName(ts typeSpec) string {
	named, ok := ts.(namedTS)
	if !ok || !strings.HasPrefix(named.name, "{") {
		return ""
	}
	close := strings.Index(named.name, "}")
	if close < 0 {
		return ""
	}
	uri, local := named.name[1:close], named.name[close+1:]
	if uri == typesystem.SystemURI || local == "" {
		return ""
	}
	return local
}

// ageFunctionPrecision maps age function names to their ELM precision string.
var ageFunctionPrecision = map[string]string{
	"AgeInYears": "Year", "AgeInYearsAt": "Year",
	"AgeInMonths": "Month", "AgeInMonthsAt": "Month",
	"AgeInWeeks": "Week", "AgeInWeeksAt": "Week",
	"AgeInDays": "Day", "AgeInDaysAt": "Day",
	"AgeInHours": "Hour", "AgeInHoursAt": "Hour",
	"AgeInMinutes": "Minute", "AgeInMinutesAt": "Minute",
	"AgeInSeconds": "Second", "AgeInSecondsAt": "Second",
}

// expandAgeFunction expands age-related FHIRHelpers function calls to CalculateAge/CalculateAgeAt.
// Returns nil if the name is not a recognized age function.
// CQF expands these during library resolution; we replicate the pattern here.
func (t *Translator) expandAgeFunction(ann, sig json.RawMessage, name string, operands []ast.Expr) elm.Expression {
	precision, ok := ageFunctionPrecision[name]
	if !ok {
		return nil
	}
	isAt := strings.HasSuffix(name, "At")
	contextName := t.currentContextName
	if contextName == "" || contextName == "Unfiltered" {
		contextName = "Patient"
	}

	// Build the birthDate access expression. The access pattern depends on the model:
	// - FHIR: Property(path="birthDate.value", source=ExpressionRef("Patient")) wrapped in ToDateTime
	// - QUICK/other: Property(path="birthDate", source=ExpressionRef("Patient")) wrapped in ToDate
	contextRef := &elm.ExpressionRefNode{
		Annotation: t.cqfAnnotation(),
		Name:       contextName,
	}

	var birthDateExpr elm.Expression
	if t.primaryModelName == "FHIR" {
		propOperand := &elm.PropertyNode{
			Annotation: t.cqfAnnotation(),
			Path:       "birthDate.value",
			Source:     contextRef,
		}
		birthDateExpr = &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.buildSigFromTypes("ToDateTime", []typeSpec{dateTS}),
			Operator:   "ToDateTime",
			Operand:    propOperand,
		}
	} else {
		propOperand := &elm.PropertyNode{
			Annotation: t.cqfAnnotation(),
			Path:       "birthDate",
			Source:     contextRef,
		}
		birthDateExpr = &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.buildSigFromTypes("ToDate", []typeSpec{dateTimeTS}),
			Operator:   "ToDate",
			Operand:    propOperand,
		}
	}

	if isAt && len(operands) > 0 {
		dateArg := t.translateExpr(operands[0])
		// Against a Date the age is computed with the Date overload, so the
		// birth date is used as it is. Converting it to DateTime regardless
		// was the stray ToDateTime issues/07 measured.
		if u, ok := birthDateExpr.(*elm.UnaryExpressionNode); ok && u.Operator == "ToDateTime" && t.isDateTS(dateArg) {
			birthDateExpr = u.Operand
		}
		ops := []elm.Expression{birthDateExpr, dateArg}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("CalculateAgeAt", ops),
			Operator:   "CalculateAgeAt",
			Precision:  precision,
			Operand:    ops,
		}
	}

	return &elm.CalculateAgeNode{
		Annotation: ann,
		Signature:  t.computeSig("CalculateAge", []elm.Expression{birthDateExpr}),
		Precision:  precision,
		Operand:    birthDateExpr,
	}
}

// convertFHIRPrimitiveList rewrites a list-valued FHIR primitive property into
// the implicit per-element conversion query CQF emits:
//
//	Query{ source: [X: <property>], return: { distinct: false, expression: To<T>(X) } }
//
// Wrapping such a list in ToList instead would produce a List<List<T>>, so this
// is checked before list promotion. Returns ok=false when expr is not a
// list-valued FHIR property with a conversion.
func (t *Translator) convertFHIRPrimitiveList(expr ast.Expr, translated elm.Expression) (elm.Expression, bool) {
	if t.fhirHelpersLocalName == "" {
		return nil, false
	}
	pe, ok := expr.(*ast.PropertyExpr)
	if !ok {
		return nil, false
	}
	sourceType := t.resolveFHIRSourceType(pe.Source)
	if sourceType == "" || !typesystem.IsFHIRListPropertyOf(sourceType, pe.Path) {
		return nil, false
	}
	fhirType, _ := typesystem.FHIRPropertyTypeOf(sourceType, pe.Path)
	fhirFunc := typesystem.FHIRCoercionFor(fhirType)
	if fhirFunc == "" {
		return nil, false
	}

	return t.fhirListConversionQuery(translated, fhirFunc, fhirType), true
}

// fhirListConversionQuery is the element-wise conversion CQF lifts a list of
// FHIR values into: Query{ source: [X: list], return: { distinct: false,
// expression: FHIRHelpers.To<T>(X) } }. None of its nodes carry a locator or a
// result type.
func (t *Translator) fhirListConversionQuery(translated elm.Expression, fhirFunc string, elemFHIRType string) elm.Expression {
	t.noteOverloadedCall(t.fhirHelpersLocalName, fhirFunc)
	alias := &elm.AliasRefNode{Annotation: t.cqfAnnotation(), Name: fhirConversionAlias}
	distinct := false
	return t.markConverted(&elm.QueryNode{
		Annotation: t.cqfAnnotation(),
		Source: []*elm.AliasedQuerySourceELM{{
			Annotation: t.cqfAnnotation(),
			Alias:      fhirConversionAlias,
			Expression: translated,
		}},
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
		Return: &elm.ReturnClauseELM{
			Annotation: t.cqfAnnotation(),
			Distinct:   &distinct,
			Expression: &elm.FunctionRefNode{
				Annotation:  t.cqfAnnotation(),
				Signature:   t.buildSigFromTypes("FunctionRef", []typeSpec{namedTS{"{http://hl7.org/fhir}" + elemFHIRType}}),
				Name:        fhirFunc,
				LibraryName: t.fhirHelpersLocalName,
				Operand:     []elm.Expression{alias},
			},
		},
	})
}

// fhirConversionAlias is the query alias CQF generates for an implicit
// per-element FHIR conversion.
const fhirConversionAlias = "X"

// extractRetrieveType returns the FHIR type name from the first Retrieve found in
// expr (e.g., for Union/Intersect source expressions). Returns "" if not found.
func (t *Translator) extractRetrieveType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.RetrieveExpr:
		typeName := e.DataType
		if i := strings.LastIndex(typeName, "."); i >= 0 {
			typeName = typeName[i+1:]
		}
		return typeName
	case *ast.BinaryExpr:
		if typeName := t.extractRetrieveType(e.Left); typeName != "" {
			return typeName
		}
		return t.extractRetrieveType(e.Right)
	case *ast.ListExpr:
		for _, elem := range e.Elements {
			if typeName := t.extractRetrieveType(elem); typeName != "" {
				return typeName
			}
		}
	}
	return ""
}

// isFHIRConceptExpr returns true when expr is a PropertyExpr that resolves to a
// FHIR CodeableConcept type (which gets coerced to CQL Concept via ToConcept).
func (t *Translator) isFHIRConceptExpr(expr ast.Expr) bool {
	if pe, ok := expr.(*ast.PropertyExpr); ok {
		return t.resolveFHIRPropertyType(pe.Source, pe.Path) == "CodeableConcept"
	}
	return false
}

// wrapToConcept wraps an ELM expression in the ELM built-in ToConcept operator.
// This is used for RHS code promotion in Concept ~ Code comparisons.
func (t *Translator) wrapToConcept(expr elm.Expression) elm.Expression {
	return &elm.ToConceptNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
		Operand:    expr,
	}
}

// resolveFHIRSourceType returns the FHIR resource or backbone type name for the// expression, by inspecting query alias type bindings and the current context.
// Returns "" if the type cannot be determined.
func (t *Translator) resolveFHIRSourceType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.IdentifierRef:
		// Check query alias type bindings (innermost scope first).
		for i := len(t.queryAliasTypes) - 1; i >= 0; i-- {
			if typ, ok := t.queryAliasTypes[i][e.Name]; ok {
				return typ
			}
		}
		// A function operand declared with a model type, e.g. (dx FHIR.Claim.Diagnosis).
		// Without this, navigation off an operand had no source type, so the
		// implicit conversion on dx.sequence = 1 was never inserted. (issues/08 W3)
		if t.functionParamScope[e.Name] {
			if local := fhirLocalTypeName(t.operandTypeSpecs[e.Name]); local != "" {
				return local
			}
		}
		// Fall back to the current context singleton (e.g., "Patient").
		if e.Name == t.currentContextName {
			return t.currentContextName
		}
	case *ast.AliasRef:
		for i := len(t.queryAliasTypes) - 1; i >= 0; i-- {
			if typ, ok := t.queryAliasTypes[i][e.Name]; ok {
				return typ
			}
		}
	case *ast.PropertyExpr:
		// For chained access (e.g., E.period.start), resolve the intermediate type.
		sourceType := t.resolveFHIRSourceType(e.Source)
		if sourceType == "" {
			return ""
		}
		typ, _ := typesystem.FHIRPropertyTypeOf(sourceType, e.Path)
		return typ
	}
	return ""
}

// resolveFHIRPropertyType returns the FHIR type of a property access on expr.
// Returns "" if the type cannot be determined or is not in the FHIR property map.
func (t *Translator) resolveFHIRPropertyType(source ast.Expr, path string) string {
	sourceType := t.resolveFHIRSourceType(source)
	if sourceType == "" {
		return ""
	}
	// Walk the base-type chain: id, meta and extension are declared on
	// Resource/Element, so a direct index misses them on every concrete type.
	typ, _ := typesystem.FHIRPropertyTypeOf(sourceType, path)
	return typ
}

// resolveFHIRPropertyCoercion returns the FHIRHelpers function name (e.g. "ToString",
// "ToDateTime") that converts the result of accessing `path` on `source`, or "" if
// no conversion applies.
//
// A list-valued property is left alone: CQF converts a list of FHIR primitives by
// lifting the conversion into a per-element query rather than wrapping the list,
// so emitting a scalar conversion here would produce the wrong tree.
func (t *Translator) resolveFHIRPropertyCoercion(source ast.Expr, path string) string {
	fhirType := t.resolveFHIRPropertyType(source, path)
	if fhirType == "" {
		return ""
	}
	if sourceType := t.resolveFHIRSourceType(source); sourceType != "" {
		if typesystem.IsFHIRListPropertyOf(sourceType, path) {
			return ""
		}
	}
	return typesystem.FHIRCoercionFor(fhirType)
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// buildMedicationReferenceSubQuery builds the reference-resolution sub-query for
// FHIR medication choice-type retrieves. When a retrieve filters by a ValueSet on a
// property like MedicationRequest.medication[x] (which can be CodeableConcept OR
// Reference(Medication)), CQF generates a union of the direct retrieve and a
// sub-query that joins the primary resource with the referenced resource.
//
// The generated sub-query:
//
//	Query {
//	  source: [{ alias: "MR", expr: Retrieve{localType} }],
//	  relationship: [With {
//	    alias: "M",
//	    expr: Retrieve{refTarget},
//	    suchThat: And(
//	      Equal(FHIRHelpers.ToString(M.id), Last(Split(FHIRHelpers.ToString(MR.medication.reference), "/"))),
//	      InValueSet(FHIRHelpers.ToConcept(M.code), vsRef)
//	    )
//	  }]
//	}
//
// localType: e.g. "MedicationRequest", refTarget: e.g. "Medication"
// loc: the locator string from the original Retrieve node (CQF reuses it on all synthetic nodes)
func (t *Translator) buildMedicationReferenceSubQuery(localType, refTarget string, vsRef *elm.ValueSetRefNode, loc string) elm.Expression {
	ann := t.cqfAnnotation()
	sig := t.cqfEmptyArrayField()
	fhirNS := "{http://hl7.org/fhir}"
	fhirBase := "http://hl7.org/fhir/StructureDefinition/"
	mrAlias := "MR"
	mAlias := "M"

	// The property name for the reference: e.g. "medication.reference"
	codeProp := typesystem.FHIRPrimaryCodePath[localType] // e.g. "medication"
	refPropPath := codeProp + ".reference"

	// Retrieve{localType} — unconstrained, no locator (CQF does not put a locator on this synthetic node)
	primaryRetrieve := &elm.RetrieveNode{
		Annotation:  ann,
		Include:     t.cqfEmptyArrayField(),
		CodeFilter:  t.cqfEmptyArrayField(),
		DateFilter:  t.cqfEmptyArrayField(),
		OtherFilter: t.cqfEmptyArrayField(),
		DataType:    fhirNS + localType,
		TemplateID:  fhirBase + localType,
	}

	// Retrieve{refTarget} — unconstrained, no locator (CQF does not put a locator on this synthetic node)
	refRetrieve := &elm.RetrieveNode{
		Annotation:  ann,
		Include:     t.cqfEmptyArrayField(),
		CodeFilter:  t.cqfEmptyArrayField(),
		DateFilter:  t.cqfEmptyArrayField(),
		OtherFilter: t.cqfEmptyArrayField(),
		DataType:    fhirNS + refTarget,
		TemplateID:  fhirBase + refTarget,
	}

	// The Medication resource id, converted to a string.
	mIDProp := &elm.PropertyNode{Annotation: ann, Path: "id", Scope: mAlias}
	mIDStr := &elm.FunctionRefNode{
		Annotation:  ann,
		Signature:   sig,
		Name:        "ToString",
		LibraryName: t.fhirHelpersLocalName,
		Operand:     []elm.Expression{mIDProp},
	}

	// The request medication reference, converted to a string.
	mrRefProp := &elm.PropertyNode{Annotation: ann, Path: refPropPath, Scope: mrAlias}
	mrRefStr := &elm.FunctionRefNode{
		Annotation:  ann,
		Signature:   sig,
		Name:        "ToString",
		LibraryName: t.fhirHelpersLocalName,
		Operand:     []elm.Expression{mrRefProp},
	}

	// That reference split on "/", so its trailing id can be taken.
	sepLit := &elm.LiteralNode{ValueType: "{urn:hl7-org:elm-types:r1}String", Value: "/"}
	splitExpr := &elm.SplitNode{Annotation: ann, Signature: sig, StringToSplit: mrRefStr, Separator: sepLit}

	// Last(Split(...))
	lastExpr := &elm.LastNode{Annotation: ann, Signature: sig, Source: splitExpr}

	// Equal(FHIRHelpers.ToString(M.id), Last(Split(...)))
	eqOperands := []elm.Expression{mIDStr, lastExpr}
	equalExpr := &elm.OperatorExpressionNode{
		Annotation: ann,
		Signature:  sig,
		Operator:   "Equal",
		Operand:    eqOperands,
	}

	// The Medication code, converted to a Concept and tested against the value set.
	mCodeProp := &elm.PropertyNode{Annotation: ann, Path: "code", Scope: mAlias}
	mCodeConcept := &elm.FunctionRefNode{
		Annotation:  ann,
		Signature:   sig,
		Name:        "ToConcept",
		LibraryName: t.fhirHelpersLocalName,
		Operand:     []elm.Expression{mCodeProp},
	}
	// Build a copy of vsRef with preserve=true for the InValueSet valueset field
	tr := true
	inVSExpr := &elm.InValueSetNode{
		Annotation: ann,
		Signature:  sig,
		Code:       mCodeConcept,
		ValueSet: &elm.ValueSetRefNode{
			Locator:     vsRef.Locator,
			Name:        vsRef.Name,
			LibraryName: vsRef.LibraryName,
			Preserve:    &tr,
		},
	}

	// Both conditions together: the reference matches and the code is in the set.
	suchThat := &elm.OperatorExpressionNode{
		Annotation: ann,
		Signature:  sig,
		Operator:   "And",
		Operand:    []elm.Expression{equalExpr, inVSExpr},
	}

	// With clause
	withClause := &elm.RelationshipClauseELM{
		Locator:    loc,
		Kind:       "With",
		Alias:      mAlias,
		Expression: refRetrieve,
		SuchThat:   suchThat,
	}

	// Empty let slice (CQF emits "let": [])
	return &elm.QueryNode{
		Annotation:   ann,
		Locator:      loc,
		Source:       []*elm.AliasedQuerySourceELM{{Locator: loc, Alias: mrAlias, Expression: primaryRetrieve}},
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{withClause},
	}
}

// convertFHIRCast wraps a cast to a convertible FHIR type in its implicit
// FHIRHelpers conversion, or returns nil when the target is not one.
func (t *Translator) convertFHIRCast(asNode *elm.AsNode, ts elm.TypeSpecifier, loc string) elm.Expression {
	if t.fhirHelpersLocalName == "" {
		return nil
	}
	named, ok := ts.(*elm.NamedTypeSpecifier)
	if !ok || !strings.HasPrefix(named.Name, "{http://hl7.org/fhir}") {
		return nil
	}
	fhirType := strings.TrimPrefix(named.Name, "{http://hl7.org/fhir}")
	fn := typesystem.FHIRCoercionFor(fhirType)
	if fn == "" {
		return nil
	}
	return t.fhirConversionWrapper(asNode, fhirType, fn, loc)
}

// aliasRefNode references a query alias, converting it through FHIRHelpers when
// the alias ranges over a convertible FHIR type and the reference is consumed.
// `(E.type) T where T in "VS"` tests ToConcept(T) in CQF; left bare, membership
// was tested against an unconverted CodeableConcept. An alias over a resource
// type has no conversion and is returned as is. (issues/06)
func (t *Translator) aliasRefNode(ann json.RawMessage, name, loc string, noCoerce bool) elm.Expression {
	ref := &elm.AliasRefNode{Annotation: ann, Name: name}
	if noCoerce || t.fhirHelpersLocalName == "" {
		return ref
	}
	fhirType := t.aliasFHIRType(name)
	fn := typesystem.FHIRCoercionFor(fhirType)
	if fn == "" {
		return ref
	}
	return t.fhirConversionWrapper(ref, fhirType, fn, loc)
}

// aliasFHIRType returns the FHIR type a query alias ranges over, or "".
func (t *Translator) aliasFHIRType(name string) string {
	for i := len(t.queryAliasTypes) - 1; i >= 0; i-- {
		if typ, ok := t.queryAliasTypes[i][name]; ok {
			return typ
		}
	}
	return ""
}

// fhirConversionWrapper wraps a FHIR-typed node in its implicit FHIRHelpers
// conversion the way the property path does: the source locator and the FHIR
// result type stay on the inner node, and the synthesized call carries neither.
func (t *Translator) fhirConversionWrapper(inner elm.Expression, fhirType, fn, loc string) elm.Expression {
	if t.opts.EnableLocators && loc != "" {
		setLocator(inner, loc)
		t.skipLocatorStamp = true
	}
	if t.opts.EnableResultTypes {
		elm.SetResultType(inner, "{http://hl7.org/fhir}"+fhirType, nil)
	}
	t.skipResultTypeStamp = true
	return &elm.FunctionRefNode{
		Annotation:  t.cqfAnnotation(),
		Signature:   t.buildSigFromTypes("FunctionRef", []typeSpec{namedTS{"{http://hl7.org/fhir}" + fhirType}}),
		Name:        fn,
		LibraryName: t.fhirHelpersLocalName,
		Operand:     []elm.Expression{inner},
	}
}

// typedAggregateOps are the aggregates whose overloads are per System type —
// Sum(List<Integer>), Min(List<DateTime>), AllTrue(List<Boolean>). Handed a list
// of FHIR values they need every element converted, and CQF lifts that
// conversion into a query over the list. Count, First, Last and Mode are
// generic in the element type and take the list as it is.
var typedAggregateOps = map[string]bool{
	"Sum": true, "Product": true, "Min": true, "Max": true,
	"Avg": true, "Median": true, "StdDev": true, "PopulationStdDev": true,
	"Variance": true, "PopulationVariance": true, "AllTrue": true, "AnyTrue": true,
}

// liftFHIRListOperand wraps an aggregate's list operand in the element-wise
// FHIRHelpers conversion when the list holds convertible FHIR values.
// `Min("Encs" E return E.period.start)` aggregates ToDateTime of each start;
// left alone, it asked for the minimum of FHIR.dateTime values. (issues/06)
func (t *Translator) liftFHIRListOperand(op string, src elm.Expression) (elm.Expression, bool) {
	if !typedAggregateOps[op] || t.fhirHelpersLocalName == "" {
		return nil, false
	}
	lt, ok := t.inferTypeSpec(src).(listTS)
	if !ok {
		return nil, false
	}
	fhirFunc := typesystem.FHIRCoercionFor(fhirLocalTypeName(lt.elem))
	if fhirFunc == "" {
		return nil, false
	}
	return t.fhirListConversionQuery(src, fhirFunc, fhirLocalTypeName(lt.elem)), true
}

// sortFieldCoercion returns the FHIRHelpers conversion for a sort-by identifier
// naming a field of a FHIR-typed query element, and that field's FHIR type.
func (t *Translator) sortFieldCoercion(name string, noCoerce bool) (fn, fhirType string) {
	if t.fhirHelpersLocalName == "" || noCoerce {
		return "", ""
	}
	local := fhirLocalTypeName(t.sortElementTS)
	if local == "" || typesystem.IsFHIRListPropertyOf(local, name) {
		return "", ""
	}
	fhirType, ok := typesystem.FHIRPropertyTypeOf(local, name)
	if !ok {
		return "", ""
	}
	return typesystem.FHIRCoercionFor(fhirType), fhirType
}

// equalityOps are the operators CQF resolves generically over any type, so two
// operands of a common FHIR type need no conversion to System values. Ordering
// comparisons are not here: they need an ordered System type, and CQF converts.
var equalityOps = map[string]bool{
	"Equal": true, "NotEqual": true, "Equivalent": true, "NotEquivalent": true,
}

// sharesFHIRType reports whether both operands of an equality are FHIR-typed
// properties with a common FHIR type — the same type, or one deriving from the
// other through the model's base-type chain.
func (t *Translator) sharesFHIRType(op string, left, right ast.Expr) bool {
	if !equalityOps[op] || t.fhirHelpersLocalName == "" {
		return false
	}
	lt, rt := t.fhirOperandType(left), t.fhirOperandType(right)
	if lt == "" || rt == "" {
		return false
	}
	return fhirDerivesFrom(lt, rt) || fhirDerivesFrom(rt, lt)
}

// fhirOperandType is the FHIR type of a property access, or "" for anything
// else (literals and System-typed expressions take the conversion path).
func (t *Translator) fhirOperandType(e ast.Expr) string {
	p, ok := e.(*ast.PropertyExpr)
	if !ok {
		return ""
	}
	return t.resolveFHIRPropertyType(p.Source, p.Path)
}

// fhirDerivesFrom reports whether FHIR type sub is super or derives from it.
func fhirDerivesFrom(sub, super string) bool {
	for depth := 0; sub != "" && depth < 32; depth++ {
		if sub == super {
			return true
		}
		sub = typesystem.FHIRBaseType[sub]
	}
	return false
}

// calculateAgePrecisions maps the precision word in CalculateAgeIn<P>[At].
var calculateAgePrecisions = map[string]string{
	"Years": "Year", "Months": "Month", "Weeks": "Week", "Days": "Day",
	"Hours": "Hour", "Minutes": "Minute", "Seconds": "Second",
}

// calculateAgeCall maps the explicit-birth-date age functions to their system
// operators: CalculateAgeIn<P>At(birthDate, asOf) is CalculateAgeAt{precision P},
// and CalculateAgeIn<P>(birthDate) is CalculateAge{precision P}. Left as calls
// they named a library function that does not exist, with no libraryName, which
// no engine can evaluate. (issues/08 W7)
//
// Unlike AgeInYearsAt, the birth date is an argument here, so it is translated
// as written — and, being consumed, converted like any FHIR value would be.
func (t *Translator) calculateAgeCall(ann json.RawMessage, name string, args []ast.Expr) elm.Expression {
	rest, ok := strings.CutPrefix(name, "CalculateAgeIn")
	if !ok {
		return nil
	}
	word, isAt := strings.CutSuffix(rest, "At")
	precision, ok := calculateAgePrecisions[word]
	if !ok {
		return nil
	}
	switch {
	case isAt && len(args) == 2:
		ops := []elm.Expression{t.translateExpr(args[0]), t.translateExpr(args[1])}
		return &elm.PrecisionOperatorNode{
			Annotation: ann,
			Signature:  t.computeSig("CalculateAgeAt", ops),
			Operator:   "CalculateAgeAt",
			Precision:  precision,
			Operand:    ops,
		}
	case !isAt && len(args) == 1:
		op := t.translateExpr(args[0])
		return &elm.CalculateAgeNode{
			Annotation: ann,
			Signature:  t.computeSig("CalculateAge", []elm.Expression{op}),
			Precision:  precision,
			Operand:    op,
		}
	}
	return nil
}

// convertFHIRCode converts an already-translated FHIR value used as the code of
// a membership test — a Coding to ToCode, a CodeableConcept to ToConcept. The
// call is synthesized, so it carries no locator or result type of its own.
// A value that is not a convertible FHIR type is returned as it is.
func (t *Translator) convertFHIRCode(e elm.Expression) elm.Expression {
	if t.fhirHelpersLocalName == "" {
		return e
	}
	local := fhirLocalTypeName(t.inferTypeSpec(e))
	fn := typesystem.FHIRCoercionFor(local)
	if local == "" || fn == "" {
		return e
	}
	return t.markConverted(&elm.FunctionRefNode{
		Annotation:  t.cqfAnnotation(),
		Signature:   t.buildSigFromTypes("FunctionRef", []typeSpec{namedTS{"{http://hl7.org/fhir}" + local}}),
		Name:        fn,
		LibraryName: t.fhirHelpersLocalName,
		Operand:     []elm.Expression{e},
	})
}

// codeSystemRef is the code system reference of an inline Code selector,
// stamped as CQF stamps it: annotation, locator and the CodeSystem type, as a
// declared CodeDef's codeSystem already is.
func (t *Translator) codeSystemRef(name, library string, loc ast.Interval) *elm.CodeSystemRef {
	ref := &elm.CodeSystemRef{Annotation: t.cqfAnnotation(), Name: name, LibraryName: library}
	if t.opts.EnableLocators {
		ref.Locator = locatorStr(loc)
	}
	if t.opts.EnableResultTypes {
		ref.ResultTypeName = typesystem.TypeCodeSystem
	}
	return ref
}
