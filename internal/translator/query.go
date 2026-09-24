package translator

// Queries: translation, implicit list traversal, and the decimal widening of aggregate sources.

import (
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

func (t *Translator) translateQuery(q *ast.QueryExpression) elm.Expression {
	// Build the alias set for this query scope.
	aliases := make(map[string]bool)
	for _, src := range q.Sources {
		if src.Alias != "" {
			aliases[src.Alias] = true
		}
	}
	for _, rel := range q.Relationship {
		if rel.Source != nil && rel.Source.Alias != "" {
			aliases[rel.Source.Alias] = true
		}
	}
	t.queryAliases = append(t.queryAliases, aliases)
	defer func() { t.queryAliases = t.queryAliases[:len(t.queryAliases)-1] }()

	// Build alias→FHIR-type map for implicit FHIRHelpers coercion, and pre-translate
	// each source expression so we can both register the alias' element type for
	// signature inference and reuse the translated node below.
	aliasTypes := make(map[string]string)
	aliasTypeSpecs := make(map[string]typeSpec)
	translatedSources := make(map[int]elm.Expression, len(q.Sources))
	for i, src := range q.Sources {
		if src.Alias == "" {
			continue
		}
		if t.fhirHelpersLocalName != "" {
			if re, ok := src.Expression.(*ast.RetrieveExpr); ok && re.DataType != "" {
				// DataType format: "FHIR.Encounter" — extract after the last dot.
				typeName := re.DataType
				if i := strings.LastIndex(typeName, "."); i >= 0 {
					typeName = typeName[i+1:]
				}
				aliasTypes[src.Alias] = typeName
			} else if src.Alias != "" {
				// For union/intersect source expressions, find the first Retrieve to determine type.
				if typeName := t.extractRetrieveType(src.Expression); typeName != "" {
					aliasTypes[src.Alias] = typeName
				} else if typeName := t.resolveFHIRSourceType(src.Expression); typeName != "" {
					// A property-valued source, e.g. (Patient.address) A — the alias
					// element type is the property's own FHIR type.
					aliasTypes[src.Alias] = typeName
				}
			}
		}
		srcExpr := t.translateExpr(src.Expression)
		translatedSources[i] = srcExpr
		srcTS := t.inferTypeSpec(srcExpr)
		if known := knownTS(srcTS); known != nil {
			if _, isList := known.(listTS); !isList {
				// A query over a single value ranges over just that value.
				aliasTypeSpecs[src.Alias] = known
			}
		}
		if lt, ok := srcTS.(listTS); ok {
			aliasTypeSpecs[src.Alias] = lt.elem
			// The cases above read the model type out of the source's syntax, so
			// they only fire when a Retrieve is visible in it. A source that is a
			// reference to a define — `"Encounters" E` rather than `[Encounter] E`
			// — has no Retrieve to find, and the alias would carry no FHIR type,
			// which silently suppresses every implicit FHIRHelpers conversion
			// inside the query. The inferred element type knows the answer, so
			// fall back to it. (issues/05 3.2)
			if t.fhirHelpersLocalName != "" && aliasTypes[src.Alias] == "" {
				if name := fhirLocalTypeName(lt.elem); name != "" {
					aliasTypes[src.Alias] = name
				}
			}
		}
	}
	t.queryAliasTypes = append(t.queryAliasTypes, aliasTypes)
	defer func() { t.queryAliasTypes = t.queryAliasTypes[:len(t.queryAliasTypes)-1] }()
	t.queryAliasTypeSpecs = append(t.queryAliasTypeSpecs, aliasTypeSpecs)
	defer func() { t.queryAliasTypeSpecs = t.queryAliasTypeSpecs[:len(t.queryAliasTypeSpecs)-1] }()

	// Build the let-variable set for this query scope (for QueryLetRef resolution).
	letScope := make(map[string]bool)
	for _, let := range q.Let {
		letScope[let.Identifier] = true
	}
	t.queryLetScopes = append(t.queryLetScopes, letScope)
	defer func() { t.queryLetScopes = t.queryLetScopes[:len(t.queryLetScopes)-1] }()
	letTypeSpecs := make(map[string]typeSpec)
	t.queryLetTypeSpecs = append(t.queryLetTypeSpecs, letTypeSpecs)
	defer func() { t.queryLetTypeSpecs = t.queryLetTypeSpecs[:len(t.queryLetTypeSpecs)-1] }()

	qn := &elm.QueryNode{
		Annotation:   t.cqfAnnotation(),
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
	}
	for i, src := range q.Sources {
		expr, ok := translatedSources[i]
		if !ok {
			expr = t.translateExpr(src.Expression)
		}
		aqse := &elm.AliasedQuerySourceELM{
			Annotation: t.cqfAnnotation(),
			Alias:      src.Alias,
			Expression: expr,
		}
		if t.opts.EnableLocators {
			aqse.Locator = locatorStr(src.Loc())
		}
		qn.Source = append(qn.Source, aqse)
	}
	for _, let := range q.Let {
		letExpr := t.translateExpr(let.Expression)
		letTypeSpecs[let.Identifier] = t.inferTypeSpec(letExpr)
		lce := &elm.LetClauseELM{
			Annotation: t.cqfAnnotation(),
			Identifier: let.Identifier,
			Expression: letExpr,
		}
		if t.opts.EnableLocators {
			lce.Locator = locatorStr(let.Loc())
		}
		qn.Let = append(qn.Let, lce)
	}
	for _, rel := range q.Relationship {
		kind := "With"
		if rel.Kind == "without" {
			kind = "Without"
		}
		r := &elm.RelationshipClauseELM{
			Annotation: t.cqfAnnotation(),
			Kind:       kind,
			Alias:      rel.Source.Alias,
			Expression: t.translateExpr(rel.Source.Expression),
		}
		if rel.Source.Alias != "" {
			if lt, ok := t.inferTypeSpec(r.Expression).(listTS); ok {
				aliasTypeSpecs[rel.Source.Alias] = lt.elem
				// The same fallback query sources get (issues/05 3.2). A `with`
				// alias never had its model type recorded at all, so properties
				// navigated off it were never converted — in one comparison the
				// source alias's side was converted and the `with` alias's side
				// was not. (issues/06)
				if t.fhirHelpersLocalName != "" && aliasTypes[rel.Source.Alias] == "" {
					if name := fhirLocalTypeName(lt.elem); name != "" {
						aliasTypes[rel.Source.Alias] = name
					}
				}
			}
		}
		r.SuchThat = t.translateExpr(rel.SuchThat)
		if t.opts.EnableLocators {
			r.Locator = locatorStr(rel.Loc())
		}
		qn.Relationship = append(qn.Relationship, r)
	}
	if q.Where != nil {
		qn.Where = t.translateExpr(q.Where)
		// The clause's outermost node carries the whole `where <expr>` span,
		// keyword included; everything inside keeps its own.
		if t.opts.EnableLocators && q.WhereLoc.Start.Line != 0 {
			setLocator(qn.Where, locatorStr(q.WhereLoc))
		}
	}
	if q.Return == nil && q.Aggregate == nil && len(q.Sources) > 1 {
		// A multi-source query with no return clause returns a tuple of its
		// aliases, distinct — CQF writes that return clause out.
		tuple := &elm.TupleNode{Annotation: t.cqfAnnotation()}
		for _, src := range q.Sources {
			tuple.Element = append(tuple.Element, &elm.TupleElementNode{
				Name:  src.Alias,
				Value: &elm.AliasRefNode{Annotation: t.cqfAnnotation(), Name: src.Alias},
			})
		}
		distinct := true
		qn.Return = &elm.ReturnClauseELM{Annotation: t.cqfAnnotation(), Distinct: &distinct, Expression: tuple}
	}
	if q.Return != nil {
		// Plain 'return' omits the field; explicit 'return all'/'return distinct'
		// emit distinct:false / distinct:true respectively.
		var distinct *bool
		if q.Return.Distinct != nil {
			d := *q.Return.Distinct
			distinct = &d
		}
		rc := &elm.ReturnClauseELM{
			Annotation: t.cqfAnnotation(),
			Distinct:   distinct,
			Expression: t.translateExprUnconsumed(q.Return.Expression),
		}
		if t.opts.EnableLocators {
			rc.Locator = locatorStr(q.Return.Loc())
		}
		qn.Return = rc
	}
	if q.Aggregate != nil {
		var distinct *bool
		if q.Aggregate.Distinct != nil {
			d := *q.Aggregate.Distinct
			distinct = &d
		}
		// The accumulator identifier is in scope inside the aggregate expression
		// and resolves to a QueryLetRef, so register it — with the type of the
		// starting expression — before translating the body.
		letScope[q.Aggregate.Identifier] = true
		starting := t.translateExpr(q.Aggregate.Starting)
		if q.Aggregate.Starting != nil {
			letTypeSpecs[q.Aggregate.Identifier] = t.inferTypeSpec(starting)
		}
		// With no starting value the accumulator starts as null, so it is typed
		// Any — definitely, not merely unknown — and CQF casts it wherever it
		// meets a typed operator.
		if q.Aggregate.Starting == nil {
			if t.anyAccumulators == nil {
				t.anyAccumulators = map[string]bool{}
			}
			prev, had := t.anyAccumulators[q.Aggregate.Identifier]
			t.anyAccumulators[q.Aggregate.Identifier] = true
			defer func() {
				if had {
					t.anyAccumulators[q.Aggregate.Identifier] = prev
				} else {
					delete(t.anyAccumulators, q.Aggregate.Identifier)
				}
			}()
		}
		if q.Aggregate.Starting == nil && t.opts.EnableResultTypes {
			elm.SetResultType(starting, typesystem.TypeAny, nil)
		}
		ac := &elm.AggregateClauseELM{
			Distinct:   distinct,
			Identifier: q.Aggregate.Identifier,
			Expression: t.translateExpr(q.Aggregate.Expression),
			Starting:   starting,
		}
		if t.opts.EnableLocators {
			ac.Locator = locatorStr(q.Aggregate.Loc())
		}
		qn.Aggregate = ac
	}
	if q.Sort != nil {
		// Sort-by expressions are scoped to the query's result element, so record
		// its type for inferTypeSpec to resolve the IdentifierRefs it contains.
		prevSortElem := t.sortElementTS
		if lt, ok := t.inferTypeSpec(qn).(listTS); ok {
			t.sortElementTS = lt.elem
		} else {
			t.sortElementTS = nil
		}
		defer func() { t.sortElementTS = prevSortElem }()

		sortClause := &elm.SortClauseELM{Annotation: t.cqfAnnotation()}
		if t.opts.EnableLocators {
			sortClause.Locator = locatorStr(q.Sort.Loc())
		}
		for _, item := range q.Sort.Items {
			dir := item.DirectionText
			if dir == "" {
				dir = "ascending"
				if item.Direction == ast.SortDesc {
					dir = "descending"
				}
			}
			sortItem := &elm.SortByItemELM{
				Annotation: t.cqfAnnotation(),
				Direction:  dir,
			}
			if t.opts.EnableLocators {
				sortItem.Locator = locatorStr(item.Loc())
			}
			if item.Expression == nil {
				// `sort asc` / `sort desc` — direction only, no by-expression.
			} else if id, ok := item.Expression.(*ast.IdentifierRef); ok && id != nil {
				sortItem.Path = id.Name
			} else {
				prevSortScope := t.inSortScope
				t.inSortScope = true
				sortItem.Expression = t.translateExpr(item.Expression)
				t.inSortScope = prevSortScope
			}
			sortClause.By = append(sortClause.By, sortItem)
		}
		qn.Sort = sortClause
	}
	// The alias and let scopes this query established are popped as soon as it
	// returns, so result types for the query and its clauses have to be resolved
	// here rather than by the caller's stamping pass.
	if t.opts.EnableResultTypes {
		for _, src := range qn.Source {
			src.ResultTypeName, src.ResultTypeSpecifier = resultTypeOf(t.nodeResultTS(src.Expression))
		}
		for _, l := range qn.Let {
			l.ResultTypeName, l.ResultTypeSpecifier = resultTypeOf(t.nodeResultTS(l.Expression))
		}
		for _, rel := range qn.Relationship {
			rel.ResultTypeName, rel.ResultTypeSpecifier = resultTypeOf(t.nodeResultTS(rel.Expression))
		}
		if qn.Aggregate != nil && qn.Aggregate.Expression != nil {
			qn.Aggregate.ResultTypeName, qn.Aggregate.ResultTypeSpecifier =
				resultTypeOf(t.nodeResultTS(qn.Aggregate.Expression))
		}
		if qn.Return != nil && qn.Return.Expression != nil && q.Return != nil {
			// A return clause yields its element type, in a list unless the
			// query ranges over a single value. The tuple return CQF writes for
			// a multi-source query is synthesized and left untyped.
			ret := t.nodeResultTS(qn.Return.Expression)
			if !t.isSingularQuery(qn) {
				ret = listTS{ret}
			}
			qn.Return.ResultTypeName, qn.Return.ResultTypeSpecifier = resultTypeOf(ret)
		}
		if qn.Sort != nil {
			for _, item := range qn.Sort.By {
				switch {
				case item.Expression != nil:
					item.ResultTypeName, item.ResultTypeSpecifier = resultTypeOf(t.nodeResultTS(item.Expression))
				case item.Path != "":
					// ByColumn names a field of the query's result element.
					if tt, ok := t.sortElementTS.(tupleTS); ok {
						if fts, found := tt.field(item.Path); found {
							item.ResultTypeName, item.ResultTypeSpecifier = resultTypeOf(fts)
						}
					}
				}
			}
		}
		t.stampResultType(qn)
	}
	// Record the query's type while its aliases are still in scope. Anything
	// that infers it later — an enclosing aggregate deciding whether its list
	// needs converting, say — would otherwise re-infer once the scope is gone
	// and, with result types off, get nothing. That is why Min over a query of
	// FHIR dateTimes was converted under one option profile and not another.
	// (issues/06 follow-up)
	t.queryTypes[qn] = t.inferTypeSpec(qn)
	return qn
}

// isDecimalListExpr reports whether expr is a list literal whose elements
// are all Decimal literals (or Null). CQF only coerces Avg/Median/etc. sources
// via a ToDecimal query when the element type isn't already Decimal.
func (t *Translator) isDecimalListExpr(expr ast.Expr) bool {
	le, ok := expr.(*ast.ListExpr)
	if !ok || len(le.Elements) == 0 {
		return false
	}
	sawDecimal := false
	for _, e := range le.Elements {
		switch e.(type) {
		case *ast.DecimalLiteral:
			sawDecimal = true
		case *ast.NullLiteral:
			// permitted, but doesn't establish type on its own
		default:
			return false
		}
	}
	return sawDecimal
}

// wrapInDecimalQuery wraps a list expression in a Query that maps each element
// through ToDecimal. CQF always does this for Avg/Median/StdDev/Variance
// regardless of input element type, coercing List<T> → List<Decimal>.
func (t *Translator) wrapInDecimalQuery(src elm.Expression) elm.Expression {
	const alias = "X"
	// Register the alias element type so inferTypeSpec(AliasRef) returns the
	// correct type when computing the inner ToDecimal signature.
	aliasTypeSpecs := map[string]typeSpec{}
	if lt, ok := t.inferTypeSpec(src).(listTS); ok {
		aliasTypeSpecs[alias] = lt.elem
	} else {
		aliasTypeSpecs[alias] = anyTS
	}
	t.queryAliasTypeSpecs = append(t.queryAliasTypeSpecs, aliasTypeSpecs)
	defer func() { t.queryAliasTypeSpecs = t.queryAliasTypeSpecs[:len(t.queryAliasTypeSpecs)-1] }()

	aliasRef := &elm.AliasRefNode{Annotation: t.cqfAnnotation(), Name: alias}
	return &elm.QueryNode{
		Annotation: t.cqfAnnotation(),
		Source: []*elm.AliasedQuerySourceELM{
			{
				Annotation: t.cqfAnnotation(),
				Alias:      alias,
				Expression: src,
			},
		},
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
		Return: &elm.ReturnClauseELM{
			Annotation: t.cqfAnnotation(),
			Distinct:   func() *bool { f := false; return &f }(),
			Expression: &elm.UnaryExpressionNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig("ToDecimal", []elm.Expression{aliasRef}),
				Operator:   "ToDecimal",
				Operand:    aliasRef,
			},
		},
	}
}

// listTraversal is implicit list traversal: navigating `.p` on a list navigates
// into each element. `Patient.name.family` is
//
//	Query($this in Patient.name where $this.family is not null return $this.family)
//
// flattened when p is itself list-valued (`Patient.name.given`). Without it the
// property was taken of the list — a different value entirely. It is on unless
// DisableListTraversal is set, so it applies to the modern `translate` default
// and the CQF CLI default, though not to the measure-bundle profile.
//
// Only the outermost node is sourced from the expression, so it alone takes the
// locator and result type (translateExpr stamps them); everything inside is
// synthesized and bare, as in CQF. Returns nil when the source is not a list of
// a model type with that property.
func (t *Translator) listTraversal(src elm.Expression, path string) elm.Expression {
	if t.opts.DisableListTraversal {
		return nil
	}
	lt, ok := t.inferTypeSpec(src).(listTS)
	if !ok {
		return nil
	}
	local := fhirLocalTypeName(lt.elem)
	if local == "" {
		return nil
	}
	fhirType, ok := typesystem.FHIRPropertyTypeOf(local, path)
	if !ok {
		return nil
	}
	var elemTS typeSpec = namedTS{"{http://hl7.org/fhir}" + fhirType}
	isList := typesystem.IsFHIRListPropertyOf(local, path)
	if isList {
		elemTS = listTS{elemTS}
	}
	prop := func() elm.Expression {
		return &elm.PropertyNode{
			Annotation: t.cqfAnnotation(),
			Path:       path,
			Source:     &elm.AliasRefNode{Annotation: t.cqfAnnotation(), Name: "$this"},
		}
	}
	isNull := &elm.UnaryExpressionNode{Annotation: t.cqfAnnotation(), Operator: "IsNull", Operand: prop()}
	isNull.Signature = t.computeSig("IsNull", []elm.Expression{isNull.Operand})
	notNull := &elm.UnaryExpressionNode{Annotation: t.cqfAnnotation(), Operator: "Not", Operand: isNull}
	notNull.Signature = t.computeSig("Not", []elm.Expression{isNull})
	distinct := false
	q := &elm.QueryNode{
		Annotation: t.cqfAnnotation(),
		Source: []*elm.AliasedQuerySourceELM{{
			Annotation: t.cqfAnnotation(),
			Alias:      "$this",
			Expression: src,
		}},
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
		Where:        notNull,
		Return:       &elm.ReturnClauseELM{Annotation: t.cqfAnnotation(), Distinct: &distinct, Expression: prop()},
	}
	t.queryTypes[q] = listTS{elemTS}
	if !isList {
		return q
	}
	return &elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig("Flatten", []elm.Expression{q}),
		Operator:   "Flatten",
		Operand:    q,
	}
}
