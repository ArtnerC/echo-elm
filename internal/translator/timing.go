package translator

// Timing and membership: timing phrases, point/interval promotion and demotion, Date/DateTime promotion.

import (
	"reflect"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// promoteIntervalDateBoundsToDateTime wraps Date-typed bounds of an IntervalNode in
// ToDateTime conversion nodes when the declared parameter type is Interval<DateTime>.
// This replicates CQF's implicit Date→DateTime promotion for parameter defaults.
func (t *Translator) promoteIntervalDateBoundsToDateTime(declaredType ast.TypeSpecifier, expr elm.Expression) elm.Expression {
	its, ok := declaredType.(*ast.IntervalTypeSpecifier)
	if !ok {
		return expr
	}
	nts, ok := its.PointType.(*ast.NamedTypeSpecifier)
	if !ok || nts.Name != "DateTime" {
		return expr
	}
	ivl, ok := expr.(*elm.IntervalNode)
	if !ok {
		return expr
	}
	ivl.Low = t.wrapDateInToDateTime(ivl.Low)
	ivl.High = t.wrapDateInToDateTime(ivl.High)
	return ivl
}

// wrapDateInToDateTime wraps a Date expression in a ToDateTime conversion node.
// If the expression is not a *elm.DateNode it is returned unchanged.
func (t *Translator) wrapDateInToDateTime(expr elm.Expression) elm.Expression {
	if expr == nil {
		return nil
	}
	if _, isDate := expr.(*elm.DateNode); !isDate {
		return expr
	}
	return &elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.buildSigFromTypes("ToDateTime", []typeSpec{dateTS}),
		Operator:   "ToDateTime",
		Operand:    expr,
	}
}

// dateTimePromotionOps are the operators for which CQF applies the implicit
// Date→DateTime conversion when comparing a Date against a DateTime. Membership
// operators are excluded: CQF leaves `dateTime in Interval<Date>` alone.
var dateTimePromotionOps = map[string]bool{
	"Equal": true, "NotEqual": true, "Equivalent": true, "NotEquivalent": true,
	"Less": true, "LessOrEqual": true, "Greater": true, "GreaterOrEqual": true,
	"Before": true, "After": true, "SameAs": true,
	"SameOrBefore": true, "SameOrAfter": true,
}

// promoteDateToDateTime applies CQL's implicit Date→DateTime conversion when one
// operand of a comparison resolves to DateTime and the other to Date. Two Dates
// are compared as Dates and are left untouched.
//
// A Date-bounded interval literal is promoted through its bounds rather than as a
// whole, so `dateTime in Interval[@2020-01-01, @2020-12-31]` converts each bound —
// this applies to membership operators too, which take no scalar promotion.
func (t *Translator) promoteDateToDateTime(op string, lhs, rhs elm.Expression) (elm.Expression, elm.Expression) {
	if t.isDateTimeTS(lhs) {
		rhs = t.promoteIntervalBounds(rhs)
	}
	if t.isDateTimeTS(rhs) {
		lhs = t.promoteIntervalBounds(lhs)
	}
	if !dateTimePromotionOps[op] {
		return lhs, rhs
	}
	switch {
	case t.isDateTimeTS(lhs) && t.isDateTS(rhs):
		return lhs, t.wrapToDateTime(rhs)
	case t.isDateTS(lhs) && t.isDateTimeTS(rhs):
		return t.wrapToDateTime(lhs), rhs
	}
	return lhs, rhs
}

func (t *Translator) isDateTS(e elm.Expression) bool {
	n, ok := t.inferTypeSpec(e).(namedTS)
	return ok && n.name == typesystem.TypeDate
}

func (t *Translator) isDateTimeTS(e elm.Expression) bool {
	n, ok := t.inferTypeSpec(e).(namedTS)
	return ok && n.name == typesystem.TypeDateTime
}

// promoteIntervalBounds converts the bounds of a Date-typed interval literal to
// DateTime. Only a literal Interval is rewritten: a reference to an interval-valued
// definition keeps whatever point type that definition declares.
func (t *Translator) promoteIntervalBounds(e elm.Expression) elm.Expression {
	iv, ok := e.(*elm.IntervalNode)
	if !ok || iv.Low == nil || iv.High == nil {
		return e
	}
	if !t.isDateTS(iv.Low) || !t.isDateTS(iv.High) {
		return e
	}
	// The promoted interval is synthetic — CQF stamps no locator on it, only on
	// the source expressions its bounds still carry.
	promoted := *iv
	promoted.Locator = ""
	promoted.LocalID = ""
	promoted.ResultTypeName = ""
	promoted.ResultTypeSpecifier = nil
	promoted.Low = t.wrapToDateTime(iv.Low)
	promoted.High = t.wrapToDateTime(iv.High)
	return &promoted
}

func (t *Translator) wrapToDateTime(e elm.Expression) elm.Expression {
	return t.markConverted(&elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig("ToDateTime", []elm.Expression{e}),
		Operator:   "ToDateTime",
		Operand:    e,
	})
}

// pointInclusionOps maps the interval-inclusion operators to their
// point-in-interval counterparts.
var pointInclusionOps = map[string]string{
	"IncludedIn":       "In",
	"ProperIncludedIn": "ProperIn",
}

// isListOrIntervalTS reports whether ts is a list or interval type specifier.
func isListOrIntervalTS(ts typeSpec) bool {
	switch ts.(type) {
	case listTS, intervalTS:
		return true
	}
	return false
}

// tsAssignable reports whether a value of type from can satisfy an operand
// declared as to. Any on either side is treated as compatible because the
// inference is incomplete, not because the types are known to differ.
func tsAssignable(to, from typeSpec) bool {
	if to == nil || from == nil {
		return false
	}
	if n, ok := to.(namedTS); ok && n.name == anyTS.name {
		return true
	}
	if n, ok := from.(namedTS); ok && n.name == anyTS.name {
		return true
	}
	return reflect.DeepEqual(to, from)
}

// rhsSatisfiesIn reports whether rhs already matches an ELM In overload for a
// left operand of type lhs — In(T, List<T>) or In(T, Interval<T>). When it does
// not, CQF promotes the right operand (ToList, or an interval construction).
// Note that `Interval[a,b] in P` where P is Interval<T> does NOT match: T is
// itself Interval<T'>, so CQF promotes P to List<Interval<T'>> via ToList.
func rhsSatisfiesIn(lhs, rhs typeSpec) bool {
	switch r := rhs.(type) {
	case listTS:
		return tsAssignable(lhs, r.elem)
	case intervalTS:
		// An implicit conversion also satisfies it: a Date point against an
		// Interval<DateTime>, or a narrower number against a wider one, is
		// converted rather than list-promoted. (issues/07)
		return tsAssignable(lhs, r.point) || implicitlyConvertible(lhs, r.point)
	}
	return false
}

// implicitlyConvertible reports a System type that converts implicitly to
// another: Date to DateTime, and up the numeric chain.
func implicitlyConvertible(from, to typeSpec) bool {
	f, t := namedName(from), namedName(to)
	if f == typesystem.TypeDate && t == typesystem.TypeDateTime {
		return true
	}
	rf, rt := numericRank[f], numericRank[t]
	return rf > 0 && rt > rf
}

// demotePointInclusionOp maps `included in` / `properly included in` (and their
// `during` synonyms) to the point-in-interval operators `In` / `ProperIn` when
// the left operand is a point rather than an interval or list. CQF selects the
// operator by resolved operand type, so `X during P` yields In for a DateTime X
// but IncludedIn for an Interval<DateTime> X.
func (t *Translator) demotePointInclusionOp(op string, leftAST ast.Expr, left elm.Expression) string {
	pointOp, ok := pointInclusionOps[op]
	if !ok {
		return op
	}
	// A FHIR dateTime/instant property coerces to a DateTime point via FHIRHelpers.
	if t.fhirHelpersLocalName != "" {
		if pe, ok := leftAST.(*ast.PropertyExpr); ok {
			if fhirType := t.resolveFHIRPropertyType(pe.Source, pe.Path); typesystem.IsFHIRDateTimeType(fhirType) {
				return pointOp
			}
		}
	}
	// Start and End yield a point whatever their operand is inferred to be.
	// Relying on inference alone made `start of E.period during P` depend on
	// knowing the FHIRHelpers.ToInterval return type, and it fell back to
	// IncludedIn whenever that was unknown. (issues/06 4.2)
	if u, ok := left.(*elm.UnaryExpressionNode); ok && (u.Operator == "Start" || u.Operator == "End") {
		return pointOp
	}
	// A FHIR Period or Range converts to an interval, so it is never a point:
	// `(P.performed as FHIR.Period) during P` stays IncludedIn. (issues/06)
	if typesystem.FHIRComplexCoercion[fhirLocalTypeName(t.inferTypeSpec(left))] == "ToInterval" {
		return op
	}
	ts := t.inferTypeSpec(left)
	if isListOrIntervalTS(ts) {
		return op
	}
	// An unresolved type is left alone: demoting on Any would change the
	// operator for expressions whose interval-ness we simply could not infer.
	if n, isNamed := ts.(namedTS); isNamed && n.name == anyTS.name {
		return op
	}
	return pointOp
}

func (t *Translator) translateTimingExpr(v *ast.TimingExpr) elm.Expression {
	lhs := t.boundary(t.translateExpr(v.Left), v.LeftBoundary, locatorStr(v.LeftBoundaryLoc))
	if v.Offset != nil && v.Offset.Quantity != nil {
		return t.translateOffsetTiming(v, lhs)
	}
	op := t.demotePointInclusionOp(v.Op, v.Left, lhs)

	rhs := t.boundary(t.translateExpr(v.Right), v.RightBoundary, locatorStr(v.RightBoundaryLoc))
	t.noteInclusionDemotion(op, v.Left, v.Right, lhs, rhs)
	op = t.demotePointContainsOp(op, rhs)
	promotedLHS, promotedRHS := t.promoteDateToDateTime(op, lhs, rhs)
	promotedLHS, promotedRHS = t.promotePointToInterval(op, v, promotedLHS, promotedRHS)
	operands := []elm.Expression{promotedLHS, promotedRHS}
	sig := t.computeSig(op, operands)
	if v.Precision != "" {
		return &elm.PrecisionOperatorNode{
			Annotation: t.cqfAnnotation(),
			Signature:  sig,
			Operator:   op,
			Precision:  v.Precision,
			Operand:    operands,
		}
	}
	return &elm.OperatorExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  sig,
		Operator:   op,
		Operand:    operands,
	}
}

// boundary selects the start or end of an interval operand, as a timing
// phrase's `starts`/`ends` or trailing `start`/`end` asks for. Built directly
// rather than through translateExpr: CQF leaves these synthesized boundaries
// without a result type.
func (t *Translator) boundary(e elm.Expression, which, loc string) elm.Expression {
	var op string
	switch which {
	case "start":
		op = "Start"
	case "end":
		op = "End"
	default:
		return e
	}
	node := &elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig(op, []elm.Expression{e}),
		Operator:   op,
		Operand:    e,
	}
	// CQF attributes the boundary to its `starts` / `ends` keyword.
	if t.opts.EnableLocators && loc != "" {
		node.Locator = loc
	}
	return node
}

// translateOffsetTiming lowers a before/after phrase that carries a quantity
// offset, the way CQF does. With X the left operand, Y the right, q the offset,
// and Y∓q meaning Y−q for before and Y+q for after:
//
//	X q before Y              SameAs(X, Y−q)
//	X q or more before Y      SameOrBefore(X, Y−q)
//	X more than q before Y    Before(X, Y−q)
//	X less than q before Y    In(X, Interval(Y−q, Y))
//	X q or less before Y      And(In(X, Interval[Y−q, Y)), Not(IsNull(Y)))
//
// and symmetrically for after. An `on or` relationship closes the bound at Y.
// A precision rides on the comparison. Ignoring the offset — the previous
// behavior — compared X against Y itself, so `ends 1 day or less on or before
// end of P` became a bare Before. (issues/06)
func (t *Translator) translateOffsetTiming(v *ast.TimingExpr, x elm.Expression) elm.Expression {
	before := v.Op == "Before" || v.Op == "SameOrBefore"
	inclusive := v.Op == "SameOrBefore" || v.Op == "SameOrAfter"

	// Y appears more than once, and CQF emits a separate node each time.
	y := func() elm.Expression {
		return t.boundary(t.translateExpr(v.Right), v.RightBoundary, locatorStr(v.RightBoundaryLoc))
	}
	shifted := func() elm.Expression {
		op := "Add"
		if before {
			op = "Subtract"
		}
		operands := []elm.Expression{y(), t.translateExpr(v.Offset.Quantity)}
		node := &elm.OperatorExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig(op, operands),
			Operator:   op,
			Operand:    operands,
		}
		// The shifted anchor is attributed to the anchor operand itself.
		if t.opts.EnableLocators {
			node.Locator = locatorStr(v.Right.Loc())
		}
		return node
	}
	compare := func(op string, l, r elm.Expression) elm.Expression {
		operands := []elm.Expression{l, r}
		if v.Precision != "" {
			return &elm.PrecisionOperatorNode{
				Annotation: t.cqfAnnotation(),
				Signature:  t.computeSig(op, operands),
				Operator:   op,
				Precision:  v.Precision,
				Operand:    operands,
			}
		}
		return &elm.OperatorExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig(op, operands),
			Operator:   op,
			Operand:    operands,
		}
	}
	pick := func(ifBefore, ifAfter string) string {
		if before {
			return ifBefore
		}
		return ifAfter
	}

	switch v.Offset.Kind {
	case "within", "properlyWithin":
		// `within q of Y` is the window [Y - q, Y + q]; `properly within`
		// opens it and needs no null guard. Unlike before/after offsets, CQF
		// attributes every synthesized node here — both bounds' arithmetic
		// included — to the quantity, and leaves the nested In unattributed.
		withinLoc := ""
		if t.opts.EnableLocators {
			withinLoc = locatorStr(v.Offset.Loc)
		}
		anchor := y()
		bound := func(op string, a elm.Expression) elm.Expression {
			operands := []elm.Expression{a, t.translateExpr(v.Offset.Quantity)}
			return &elm.OperatorExpressionNode{
				Annotation: t.cqfAnnotation(),
				Locator:    withinLoc,
				Signature:  t.computeSig(op, operands),
				Operator:   op,
				Operand:    operands,
			}
		}
		properly := v.Offset.Kind == "properlyWithin"
		iv := &elm.IntervalNode{
			Annotation: t.cqfAnnotation(),
			Locator:    withinLoc,
			Low:        bound("Subtract", anchor),
			High:       bound("Add", y()),
			LowClosed:  !properly,
			HighClosed: !properly,
		}
		if t.opts.EnableResultTypes {
			if pt := knownTS(t.inferTypeSpec(anchor)); pt != nil {
				_, spec := resultTypeOf(intervalTS{pt})
				elm.SetResultType(iv, "", spec)
			}
		}
		in := compare("In", x, iv)
		if properly {
			return in
		}
		isNull := &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Locator:    withinLoc,
			Operator:   "IsNull",
			Operand:    y(),
		}
		isNull.Signature = t.computeSig("IsNull", []elm.Expression{isNull.Operand})
		notNull := &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Locator:    withinLoc,
			Signature:  t.computeSig("Not", []elm.Expression{isNull}),
			Operator:   "Not",
			Operand:    isNull,
		}
		operands := []elm.Expression{in, notNull}
		return &elm.OperatorExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig("And", operands),
			Operator:   "And",
			Operand:    operands,
		}
	case "orMore":
		return compare(pick("SameOrBefore", "SameOrAfter"), x, shifted())
	case "moreThan":
		return compare(pick("Before", "After"), x, shifted())
	case "orLess", "lessThan":
		farClosed := v.Offset.Kind == "orLess"
		anchor := y()
		// The nodes the offset synthesizes are attributed to the offset phrase
		// ("3 days or less"), as CQF does.
		offsetLoc := ""
		if t.opts.EnableLocators {
			offsetLoc = locatorStr(v.Offset.Loc)
		}
		iv := &elm.IntervalNode{Annotation: t.cqfAnnotation(), Locator: offsetLoc}
		if before {
			iv.Low, iv.High = shifted(), anchor
			iv.LowClosed, iv.HighClosed = farClosed, inclusive
		} else {
			iv.Low, iv.High = anchor, shifted()
			iv.LowClosed, iv.HighClosed = inclusive, farClosed
		}
		if t.opts.EnableResultTypes {
			if pt := knownTS(t.inferTypeSpec(anchor)); pt != nil {
				_, spec := resultTypeOf(intervalTS{pt})
				elm.SetResultType(iv, "", spec)
			}
		}
		in := compare("In", x, iv)
		if v.Offset.Kind == "lessThan" {
			// Top-level: it carries the whole expression's locator instead.
			return in
		}
		if offsetLoc != "" {
			setLocator(in, offsetLoc)
		}
		// `or less` must not hold vacuously when Y is null, which would leave
		// the interval unbounded on one side; CQF guards it explicitly.
		isNull := &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Locator:    offsetLoc,
			Operator:   "IsNull",
			Operand:    y(),
		}
		isNull.Signature = t.computeSig("IsNull", []elm.Expression{isNull.Operand})
		notNull := &elm.UnaryExpressionNode{
			Annotation: t.cqfAnnotation(),
			Locator:    offsetLoc,
			Signature:  t.computeSig("Not", []elm.Expression{isNull}),
			Operator:   "Not",
			Operand:    isNull,
		}
		operands := []elm.Expression{in, notNull}
		return &elm.OperatorExpressionNode{
			Annotation: t.cqfAnnotation(),
			Signature:  t.computeSig("And", operands),
			Operator:   "And",
			Operand:    operands,
		}
	}
	return compare("SameAs", x, shifted())
}

// pointContainsOps map interval inclusion to point membership, used when the
// right operand of `includes` is a point: `P includes start Q` is
// Contains(P, Start(Q)), the mirror of `included in` becoming In.
var pointContainsOps = map[string]string{
	"Includes":       "Contains",
	"ProperIncludes": "ProperContains",
}

func (t *Translator) demotePointContainsOp(op string, right elm.Expression) string {
	pointOp, ok := pointContainsOps[op]
	if !ok {
		return op
	}
	if u, ok := right.(*elm.UnaryExpressionNode); ok && (u.Operator == "Start" || u.Operator == "End") {
		return pointOp
	}
	if isPointTS(t.inferTypeSpec(right)) {
		return pointOp
	}
	return op
}

// isPointTS reports whether a type is known and is a single value rather than a
// collection, interval, choice or tuple.
func isPointTS(ts typeSpec) bool {
	switch knownTS(ts).(type) {
	case nil, listTS, intervalTS, choiceTS, tupleTS:
		return false
	}
	return true
}

// pointPromotingOps compare two intervals or two points. Given one of each, CQF
// promotes the point to a unit interval rather than rejecting the comparison —
// on whichever side the point is. SameAs and Meets have no such promotion; CQF
// rejects them outright.
var pointPromotingOps = map[string]bool{
	"Before": true, "After": true, "SameOrBefore": true, "SameOrAfter": true,
}

// promotePointToInterval applies that promotion. The point side is translated
// again for each copy, as CQF emits separate nodes. (issues/06 follow-up)
func (t *Translator) promotePointToInterval(op string, v *ast.TimingExpr, lhs, rhs elm.Expression) (elm.Expression, elm.Expression) {
	if !pointPromotingOps[op] {
		return lhs, rhs
	}
	lt, rt := t.inferTypeSpec(lhs), t.inferTypeSpec(rhs)
	_, leftInterval := lt.(intervalTS)
	_, rightInterval := rt.(intervalTS)
	switch {
	case leftInterval && isPointTS(rt):
		again := func() elm.Expression {
			return t.boundary(t.translateExpr(v.Right), v.RightBoundary, locatorStr(v.RightBoundaryLoc))
		}
		return lhs, t.unitInterval(rhs, again, rt)
	case rightInterval && isPointTS(lt):
		again := func() elm.Expression {
			return t.boundary(t.translateExpr(v.Left), v.LeftBoundary, locatorStr(v.LeftBoundaryLoc))
		}
		return t.unitInterval(lhs, again, lt), rhs
	}
	return lhs, rhs
}

// unitInterval is the promotion CQF builds for a point X:
// if X is null then null else Interval[X, X]. Only the null branch carries a
// type — the interval type — and none of the synthesized nodes a locator.
func (t *Translator) unitInterval(first elm.Expression, again func() elm.Expression, pt typeSpec) elm.Expression {
	isNull := &elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig("IsNull", []elm.Expression{first}),
		Operator:   "IsNull",
		Operand:    first,
	}
	null := &elm.NullNode{Annotation: t.cqfAnnotation()}
	if t.opts.EnableResultTypes {
		_, spec := resultTypeOf(intervalTS{pt})
		elm.SetResultType(null, "", spec)
	}
	return t.markConverted(&elm.IfNode{
		Annotation: t.cqfAnnotation(),
		Condition:  isNull,
		Then:       null,
		Else: &elm.IntervalNode{
			Annotation: t.cqfAnnotation(),
			Low:        again(),
			High:       again(),
			LowClosed:  true,
			HighClosed: true,
		},
	})
}

// defaultNeedsDateTimePromotion reports a parameter default that is an interval
// selector with a Date bound, declared as Interval<DateTime>.
func (t *Translator) defaultNeedsDateTimePromotion(declared ast.TypeSpecifier, expr elm.Expression) bool {
	its, ok := declared.(*ast.IntervalTypeSpecifier)
	if !ok {
		return false
	}
	if nts, ok := its.PointType.(*ast.NamedTypeSpecifier); !ok || nts.Name != "DateTime" {
		return false
	}
	ivl, ok := expr.(*elm.IntervalNode)
	if !ok {
		return false
	}
	_, lowDate := ivl.Low.(*elm.DateNode)
	_, highDate := ivl.High.(*elm.DateNode)
	return lowDate || highDate
}
