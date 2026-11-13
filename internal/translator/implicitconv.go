package translator

import (
	"encoding/json"

	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// This file applies the implicit conversions CQF inserts when it resolves a
// system operator's overload: an untyped null argument is cast to the resolved
// parameter type, and a numeric argument is converted up the implicit
// conversion chain Integer → Long → Decimal → Quantity to meet the overload.
//
// It runs once per translated node (see translateExpr), after the node's own
// operands have been translated and converted but while any enclosing query's
// alias scope is still live, so operand types are as well known as they will
// ever be. It replaces checks that looked at operand *syntax* — "is this an
// Integer literal?" — which is why Ceiling(1) and 2.0 * Power(10, -8) were
// missed: the operand was an Integer, just not a literal one.
//
// The rules are deliberately limited to operators whose CQF behavior has been
// observed (the HL7 cql-tests conformance suite exercises every one of them).
// An operator not listed here is left exactly as translated.

// numericRank orders the System types on the implicit conversion chain.
var numericRank = map[string]int{
	typesystem.TypeInteger:  1,
	typesystem.TypeLong:     2,
	typesystem.TypeDecimal:  3,
	typesystem.TypeQuantity: 4,
}

var rankType = map[int]string{
	1: typesystem.TypeInteger,
	2: typesystem.TypeLong,
	3: typesystem.TypeDecimal,
	4: typesystem.TypeQuantity,
}

// conversionOperator is the ELM operator CQF inserts to convert up to a type.
var conversionOperator = map[string]string{
	typesystem.TypeLong:     "ToLong",
	typesystem.TypeDecimal:  "ToDecimal",
	typesystem.TypeQuantity: "ToQuantity",
}

// numericOperators resolve to the overload matching their widest numeric
// operand. The value is the narrowest overload that exists, and the widest:
// Divide and Log are Decimal-only at the bottom; Power has no Quantity overload.
var numericOperators = map[string]struct{ min, max int }{
	"Add": {1, 4}, "Subtract": {1, 4}, "Multiply": {1, 4}, "Divide": {3, 4},
	"Modulo": {1, 4}, "TruncatedDivide": {1, 4}, "Power": {1, 3}, "Log": {3, 3},
	"Equal": {1, 4}, "Equivalent": {1, 4},
	"Less": {1, 4}, "LessOrEqual": {1, 4}, "Greater": {1, 4}, "GreaterOrEqual": {1, 4},
}

// decimalOnlyUnary have a single Decimal overload, so an Integer or Long
// argument is converted to Decimal and a null is cast to it.
var decimalOnlyUnary = map[string]bool{
	"Ceiling": true, "Floor": true, "Truncate": true, "Exp": true, "Ln": true,
}

// fixedNullTypes give the parameter type an untyped null is cast to, for
// operators whose every overload agrees on it.
var fixedNullTypes = map[string]string{
	"And": typesystem.TypeBoolean, "Or": typesystem.TypeBoolean,
	"Xor": typesystem.TypeBoolean, "Implies": typesystem.TypeBoolean,
	"Not": typesystem.TypeBoolean, "IsTrue": typesystem.TypeBoolean, "IsFalse": typesystem.TypeBoolean,
	"Upper": typesystem.TypeString, "Lower": typesystem.TypeString,
	"StartsWith": typesystem.TypeString, "EndsWith": typesystem.TypeString,
	"Matches": typesystem.TypeString, "ReplaceMatches": typesystem.TypeString,
	"Concatenate": typesystem.TypeString,
	"Log":         typesystem.TypeDecimal,
}

// namedSlotNullTypes are the same, per named operand slot.
var namedSlotNullTypes = map[string]map[string]typeSpec{
	"Substring":      {"stringToSub": strTS, "startIndex": intTS, "length": intTS},
	"Split":          {"stringToSplit": strTS, "separator": strTS},
	"PositionOf":     {"pattern": strTS, "string": strTS},
	"LastPositionOf": {"pattern": strTS, "string": strTS},
	"Combine":        {"source": listTS{strTS}, "separator": strTS},
	"Round":          {"operand": decTS, "precision": intTS},
	"IndexOf":        {"source": listTS{anyTS}},
	"Slice":          {"source": listTS{anyTS}},
}

// applyImplicitConversions rewrites one node's direct operands. It returns the
// node, which is changed in place.
func (t *Translator) applyImplicitConversions(e elm.Expression) elm.Expression {
	switch n := e.(type) {
	case *elm.OperatorExpressionNode:
		t.convertOperatorOperands(n)
	case *elm.UnaryExpressionNode:
		t.convertUnaryOperand(n)
	case *elm.NamedOperatorExpressionNode:
		t.convertNamedOperands(n)
	case *elm.AggregateExpressionNode:
		if n.Operator == "AllTrue" || n.Operator == "AnyTrue" {
			switch {
			case isBareNull(n.Source):
				n.Source = t.castNull(n.Source, listTS{boolTS})
			case isUntypedList(n.Source):
				n.Source = t.liftListCast(n.Source, boolTS)
			default:
				return e
			}
			n.Signature = t.computeSig(n.Operator, []elm.Expression{n.Source})
		}
	case *elm.DateTimeNode:
		before := presentOperands(n.Year, n.Month, n.Day, n.Hour, n.Minute, n.Second, n.Millisecond, n.TimezoneOffset)
		for _, f := range []*elm.Expression{&n.Year, &n.Month, &n.Day, &n.Hour, &n.Minute, &n.Second, &n.Millisecond} {
			t.castNullField(f, intTS)
		}
		t.castNullField(&n.TimezoneOffset, decTS)
		if r := numericRank[namedName(t.typeOf(n.TimezoneOffset))]; r == 1 || r == 2 {
			n.TimezoneOffset = t.convertTo(n.TimezoneOffset, typesystem.TypeDecimal)
		}
		// Recompute the constructor's signature only when a component changed:
		// a DateTime literal carries none, and must keep carrying none.
		if after := presentOperands(n.Year, n.Month, n.Day, n.Hour, n.Minute, n.Second, n.Millisecond, n.TimezoneOffset); !sameExprs(before, after) {
			n.Signature = t.computeSig("DateTime", after)
		}
	case *elm.DateNode:
		for _, f := range []*elm.Expression{&n.Year, &n.Month, &n.Day} {
			t.castNullField(f, intTS)
		}
	case *elm.TimeNode:
		for _, f := range []*elm.Expression{&n.Hour, &n.Minute, &n.Second, &n.Millisecond} {
			t.castNullField(f, intTS)
		}
	case *elm.ListNode:
		t.castNullListElements(n)
		// Date and DateTime elements share the type DateTime.
		if t.hasDateAndDateTime(n.Element) {
			for i, el := range n.Element {
				if t.isDateTS(el) {
					n.Element[i] = t.wrapToDateTime(el)
				}
			}
		}
	case *elm.IfNode:
		if t.isDateTS(n.Then) && t.isDateTimeTS(n.Else) {
			n.Then = t.wrapToDateTime(n.Then)
		} else if t.isDateTimeTS(n.Then) && t.isDateTS(n.Else) {
			n.Else = t.wrapToDateTime(n.Else)
		}
	case *elm.CaseNode:
		branches := []elm.Expression{n.Else}
		for _, it := range n.CaseItem {
			branches = append(branches, it.Then)
		}
		if t.hasDateAndDateTime(branches) {
			for _, it := range n.CaseItem {
				if t.isDateTS(it.Then) {
					it.Then = t.wrapToDateTime(it.Then)
				}
			}
			if t.isDateTS(n.Else) {
				n.Else = t.wrapToDateTime(n.Else)
			}
		}
	case *elm.PrecisionOperatorNode:
		if len(n.Operand) == 2 {
			n.Operand[0], n.Operand[1] = t.promoteDateIntervalOperands(n.Operand[0], n.Operand[1])
		}
	case *elm.InstanceNode:
		// System.Quantity's value is a Decimal.
		if n.ClassType == typesystem.TypeQuantity {
			for _, el := range n.Element {
				if el.Name == "value" {
					if r := numericRank[namedName(t.typeOf(el.Value))]; r == 1 || r == 2 {
						el.Value = t.convertTo(el.Value, typesystem.TypeDecimal)
					}
				}
			}
		}
	}
	return e
}

// definitelyAny reports an expression whose type is known to be Any, as
// opposed to merely unresolved — CQF records the former as a result type:
// an aggregate accumulator with no starting value; a Coalesce whose first
// alternative is one, or over an untyped list (CQF binds Coalesce's type from
// its first operand); and an element or point taken from a collection whose
// element type is Any.
func (t *Translator) definitelyAny(e elm.Expression) bool {
	switch v := e.(type) {
	case *elm.LetRefNode:
		return t.anyAccumulators[v.Name]
	case *elm.OperatorExpressionNode:
		if len(v.Operand) == 0 {
			return false
		}
		switch v.Operator {
		case "Coalesce":
			if len(v.Operand) == 1 && isUntypedList(v.Operand[0]) {
				return true
			}
			return t.definitelyAny(v.Operand[0])
		case "Indexer":
			return anyCollection(v.Operand[0])
		}
	case *elm.UnaryExpressionNode:
		switch v.Operator {
		case "PointFrom", "Width":
			return anyCollection(v.Operand)
		}
	case *elm.SingletonFromNode:
		return anyCollection(v.Operand)
	case *elm.AggregateExpressionNode:
		if v.Operator == "First" || v.Operator == "Last" {
			return anyCollection(v.Source)
		}
	}
	return false
}

// anyCollection reports a list or interval whose element type is definitely
// Any: an empty or all-null list selector, an interval selector with two null
// bounds, or a cast to List<Any> / Interval<Any>.
func anyCollection(e elm.Expression) bool {
	switch v := e.(type) {
	case *elm.ListNode:
		return isUntypedList(v)
	case *elm.IntervalNode:
		return isBareNull(v.Low) && isBareNull(v.High)
	case *elm.AsNode:
		switch spec := v.AsTypeSpecifier.(type) {
		case *elm.ListTypeSpecifier:
			return isAnySpecifier(spec.ElementType)
		case *elm.IntervalTypeSpecifier:
			return isAnySpecifier(spec.PointType)
		}
	}
	return false
}

func isAnySpecifier(ts elm.TypeSpecifier) bool {
	n, ok := ts.(*elm.NamedTypeSpecifier)
	return ok && n.Name == typesystem.TypeAny
}

// isSingularQuery reports a query whose only source is a single value rather
// than a list: `(4) l` ranges over just 4, and yields a value, not a list.
func (t *Translator) isSingularQuery(q *elm.QueryNode) bool {
	if len(q.Source) != 1 || q.Source[0] == nil {
		return false
	}
	st := knownTS(t.nodeResultTS(q.Source[0].Expression))
	if st == nil {
		return false
	}
	_, isList := st.(listTS)
	return !isList
}

func (t *Translator) convertOperatorOperands(n *elm.OperatorExpressionNode) {
	ops := n.Operand
	changed := false
	demoted := false
	set := func(i int, e elm.Expression) {
		ops[i] = e
		changed = true
	}

	// A null on the "point" side of an inclusion operator makes it a
	// membership test: `X includes null` is Contains, `null included in X` In.
	if len(ops) == 2 {
		switch n.Operator {
		case "Includes", "ProperIncludes":
			if isBareNull(ops[1]) && isCollection(t.typeOf(ops[0])) {
				n.Operator = map[string]string{"Includes": "Contains", "ProperIncludes": "ProperContains"}[n.Operator]
				changed = true
			}
		case "IncludedIn", "ProperIncludedIn":
			if isBareNull(ops[0]) && isCollection(t.typeOf(ops[1])) {
				n.Operator = map[string]string{"IncludedIn": "In", "ProperIncludedIn": "ProperIn"}[n.Operator]
				changed, demoted = true, true
			}
		}
	}

	// An interval selector with two null bounds, as the second operand, takes
	// its point type from the first. CQF binds a generic operator's type
	// parameter from the first operand, so a null-bounded first operand stays
	// untyped (the same asymmetry as Starts / Ends below).
	if len(ops) == 2 {
		if iv, ok := ops[1].(*elm.IntervalNode); ok && isBareNull(iv.Low) && isBareNull(iv.High) {
			first := t.typeOf(ops[0])
			pt := pointTypeOf(first)
			if pt == nil && (n.Operator == "In" || n.Operator == "ProperIn") {
				pt = first
			}
			if pt != nil {
				// CQF rebuilds the selector around the cast bounds, so the
				// interval it emits is synthesized: no locator, no result type.
				iv.Low, iv.High = t.castNull(iv.Low, pt), t.castNull(iv.High, pt)
				iv.Locator = ""
				elm.SetResultType(iv, "", nil)
				changed = true
			}
		}
	}

	// An empty or all-null list selector used with a typed list or element is
	// lifted element-wise to that element type: `{ 1, 2, 3 } union {}` casts
	// each element of {} to Integer.
	if listSetOperators[n.Operator] && len(ops) == 2 {
		for i := range ops {
			// Between two lists the type parameter binds from the first, so
			// only a second-position empty list is lifted:
			// `{ 1, 2, 3 } union {}` lifts, `{} included in { 1, 2, 3 }` does not.
			if !isUntypedList(ops[i]) || (i == 0 && listListOperators[n.Operator]) {
				continue
			}
			other := t.typeOf(ops[1-i])
			elem := elementTypeOf(other)
			if lt, ok := other.(listTS); !ok || lt.elem == nil {
				// The other operand is the element itself (Contains / In).
				if !isCollection(other) {
					elem = other
				}
			}
			if elem != nil {
				set(i, t.liftListCast(ops[i], elem))
			}
		}
	}

	// A Date meets a DateTime: the Date side converts. For a point tested
	// against an Interval<DateTime> (In / Contains) that is the point; for two
	// intervals it is the bounds of a Date interval selector.
	if len(ops) == 2 {
		switch n.Operator {
		case "In", "ProperIn":
			if t.isDateTS(ops[0]) && isDateTimeInterval(t.typeOf(ops[1])) {
				set(0, t.wrapToDateTime(ops[0]))
			}
		case "Contains", "ProperContains":
			if t.isDateTS(ops[1]) && isDateTimeInterval(t.typeOf(ops[0])) {
				set(1, t.wrapToDateTime(ops[1]))
			}
		}
		a, b := t.promoteDateIntervalOperands(ops[0], ops[1])
		if a != ops[0] {
			set(0, a)
		}
		if b != ops[1] {
			set(1, b)
		}
	}

	// Untyped nulls take the resolved parameter type.
	if ft, ok := fixedNullTypes[n.Operator]; ok {
		for i := range ops {
			if isBareNull(ops[i]) {
				set(i, t.castNull(ops[i], namedTS{ft}))
			}
		}
	}
	switch n.Operator {
	case "Indexer", "HighBoundary", "LowBoundary":
		if len(ops) == 2 && isBareNull(ops[1]) {
			set(1, t.castNull(ops[1], intTS))
		}
	case "Coalesce":
		// Recorded from CQF's output rather than derived. With a typed first
		// alternative every null is cast to it: Coalesce('a', null),
		// Coalesce(Now(), null), Coalesce(1, null, null, 4, 5). With a null
		// first alternative CQF casts only for String and list types —
		// Coalesce(null, 'a') and Coalesce(null, null, {'a'}) are cast, while
		// Coalesce(null, 1), Coalesce(null, null, 42) and the DateTime and Time
		// equivalents are left bare.
		var common typeSpec
		for _, o := range ops {
			if ts := t.typeOf(o); ts != nil {
				common = ts
				break
			}
		}
		if common != nil && isBareNull(ops[0]) {
			if _, isList := common.(listTS); !isList && namedName(common) != typesystem.TypeString {
				common = nil
			}
		}
		if common != nil {
			for i := range ops {
				if isBareNull(ops[i]) {
					set(i, t.castNull(ops[i], common))
				}
			}
		}
	case "Contains", "ProperContains":
		if len(ops) == 2 && isBareNull(ops[1]) {
			if et := elementTypeOf(t.typeOf(ops[0])); et != nil {
				set(1, t.castNull(ops[1], et))
			}
		}
	case "In", "ProperIn":
		// Against an interval the null point is cast — `null in
		// Interval[@T05:59, @T10:59]` becomes Time — but against a list only
		// when the membership test came from `included in`: `null in { 1, null }`
		// stays untyped while `null properly included in { 's', null }` does not.
		if len(ops) == 2 && isBareNull(ops[0]) {
			other := t.typeOf(ops[1])
			pt := pointTypeOf(other)
			if pt == nil && demoted {
				pt = elementTypeOf(other)
			}
			if pt != nil {
				set(0, t.castNull(ops[0], pt))
			}
		}
	case "Union", "Intersect", "Except":
		if len(ops) == 2 {
			for i := range ops {
				// A value set where a list is required is its expansion: the
				// implicit ValueSet → List<Code> conversion CQF writes as
				// ExpandValueSet. Without it the union was of two value sets,
				// which is not an operation on lists at all. (issues/08 W6)
				if namedName(t.typeOf(ops[i])) == typesystem.TypeValueSet {
					set(i, t.expandValueSet(ops[i]))
					continue
				}
				if isBareNull(ops[i]) {
					if other := t.typeOf(ops[1-i]); isCollection(other) {
						set(i, t.castNull(ops[i], other))
					}
				}
			}
		}
	case "Starts", "Ends":
		// CQF binds the interval type from the first operand, so a null first
		// operand is cast to Interval<Any> and a null second operand to the
		// first operand's type.
		if len(ops) == 2 {
			if isBareNull(ops[1]) {
				if first := t.typeOf(ops[0]); first != nil {
					set(1, t.castNull(ops[1], first))
				}
			} else if isBareNull(ops[0]) {
				if _, ok := t.typeOf(ops[1]).(intervalTS); ok {
					set(0, t.castNull(ops[0], intervalTS{anyTS}))
				}
			}
		}
	case "Expand":
		if len(ops) >= 1 {
			switch {
			case isBareNull(ops[0]):
				set(0, t.castNull(ops[0], intervalTS{anyTS}))
			case isUntypedList(ops[0]):
				set(0, t.liftListCast(ops[0], intervalTS{anyTS}))
			}
		}
	}

	// Two tuples with different elements have no common tuple type, so
	// comparing them resolves through list promotion: CQF wraps each in ToList
	// and compares List<Any>. Only when list promotion is on.
	if (n.Operator == "Equal" || n.Operator == "Equivalent") && len(ops) == 2 && !t.opts.DisableListPromotion {
		a, aok := t.typeOf(ops[0]).(tupleTS)
		b, bok := t.typeOf(ops[1]).(tupleTS)
		if aok && bok && !sameTupleShape(a, b) {
			for i := range ops {
				set(i, t.markConverted(&elm.UnaryExpressionNode{
					Annotation: t.cqfAnnotation(),
					Signature:  t.computeSig("ToList", []elm.Expression{ops[i]}),
					Operator:   "ToList",
					Operand:    ops[i],
				}))
			}
		}
	}

	// Numeric operands convert up to the resolved overload. A null operand —
	// bare, or already cast to a sibling's type — is cast straight to the
	// resolved type rather than cast and then converted.
	if bounds, ok := numericOperators[n.Operator]; ok && len(ops) >= 2 {
		target := bounds.min
		all, typed := true, 0
		for _, o := range ops {
			if isNullish(o) || t.definitelyAny(o) {
				continue
			}
			r := numericRank[namedName(t.typeOf(o))]
			if r == 0 {
				all = false
				break
			}
			typed++
			if r > target {
				target = r
			}
		}
		if all && typed > 0 && target <= bounds.max {
			to := rankType[target]
			for i, o := range ops {
				switch {
				case isNullish(o):
					if namedName(t.typeOf(o)) != to {
						set(i, t.castNull(nullOf(o), namedTS{to}))
					}
				case t.definitelyAny(o):
					set(i, t.castNull(o, namedTS{to}))
				case numericRank[namedName(t.typeOf(o))] < target:
					set(i, t.convertTo(o, to))
				}
			}
		}
	}

	if changed {
		n.Signature = t.computeSig(n.Operator, ops)
	}
}

func (t *Translator) convertUnaryOperand(n *elm.UnaryExpressionNode) {
	before := n.Operand
	if isBareNull(n.Operand) {
		switch {
		case fixedNullTypes[n.Operator] != "":
			n.Operand = t.castNull(n.Operand, namedTS{fixedNullTypes[n.Operator]})
		case decimalOnlyUnary[n.Operator]:
			n.Operand = t.castNull(n.Operand, decTS)
		case n.Operator == "Exists":
			n.Operand = t.castNull(n.Operand, listTS{anyTS})
		}
	} else if decimalOnlyUnary[n.Operator] {
		if r := numericRank[namedName(t.typeOf(n.Operand))]; r == 1 || r == 2 {
			n.Operand = t.convertTo(n.Operand, typesystem.TypeDecimal)
		}
	}
	if n.Operand != before {
		n.Signature = t.computeSig(n.Operator, []elm.Expression{n.Operand})
	}
}

func (t *Translator) convertNamedOperands(n *elm.NamedOperatorExpressionNode) {
	changed := false
	slots := namedSlotNullTypes[n.Operator]
	for i := range n.Operands {
		op := &n.Operands[i]
		if isBareNull(op.Value) {
			if ts, ok := slots[op.Name]; ok {
				op.Value = t.castNull(op.Value, ts)
				changed = true
			} else if n.Operator == "IndexOf" && op.Name == "element" {
				if et := elementTypeOf(t.typeOf(namedOperand(n, "source"))); et != nil {
					op.Value = t.castNull(op.Value, et)
					changed = true
				}
			}
			continue
		}
		if n.Operator == "Combine" && op.Name == "source" && isUntypedList(op.Value) {
			op.Value = t.liftListCast(op.Value, strTS)
			changed = true
			continue
		}
		if n.Operator == "Round" && op.Name == "operand" {
			if r := numericRank[namedName(t.typeOf(op.Value))]; r == 1 || r == 2 {
				op.Value = t.convertTo(op.Value, typesystem.TypeDecimal)
				changed = true
			}
		}
	}
	if changed {
		vals := make([]elm.Expression, 0, len(n.Operands))
		for _, op := range n.Operands {
			vals = append(vals, op.Value)
		}
		n.Signature = t.computeSig(n.Operator, vals)
	}
}

// castNullListElements casts the null elements of a list selector to the type
// its other elements share.
func (t *Translator) castNullListElements(n *elm.ListNode) {
	var common typeSpec
	for _, el := range n.Element {
		if isBareNull(el) {
			continue
		}
		ts := t.typeOf(el)
		if ts == nil {
			return
		}
		if common == nil {
			common = ts
		} else if !typeSpecEqual(common, ts) {
			return
		}
	}
	if common == nil {
		return
	}
	for i, el := range n.Element {
		if isBareNull(el) {
			n.Element[i] = t.castNull(el, common)
		}
	}
}

func (t *Translator) castNullField(f *elm.Expression, ts typeSpec) {
	if isBareNull(*f) {
		*f = t.castNull(*f, ts)
	}
}

// castNull wraps an untyped null in the As CQF emits for it. The As is
// synthesized, so like CQF's it carries no locator and no result type.
func (t *Translator) castNull(e elm.Expression, ts typeSpec) elm.Expression {
	return t.castNullTo(e, ts)
}

// convertTo wraps a numeric operand in the conversion up to a wider type.
func (t *Translator) convertTo(e elm.Expression, to string) elm.Expression {
	op := conversionOperator[to]
	return t.markConverted(&elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig(op, []elm.Expression{e}),
		Operator:   op,
		Operand:    e,
	})
}

// typeOf is an operand's known type, or nil when it is unknown or Any.
func (t *Translator) typeOf(e elm.Expression) typeSpec {
	if e == nil || isBareNull(e) {
		return nil
	}
	return knownTS(t.inferTypeSpec(e))
}

func isBareNull(e elm.Expression) bool {
	_, ok := e.(*elm.NullNode)
	return ok
}

func namedName(ts typeSpec) string {
	if n, ok := ts.(namedTS); ok {
		return n.name
	}
	return ""
}

func isCollection(ts typeSpec) bool {
	switch ts.(type) {
	case listTS, intervalTS:
		return true
	}
	return false
}

func elementTypeOf(ts typeSpec) typeSpec {
	switch v := ts.(type) {
	case listTS:
		return knownTS(v.elem)
	case intervalTS:
		return knownTS(v.point)
	}
	return nil
}

func pointTypeOf(ts typeSpec) typeSpec {
	if v, ok := ts.(intervalTS); ok {
		return knownTS(v.point)
	}
	return nil
}

func namedOperand(n *elm.NamedOperatorExpressionNode, name string) elm.Expression {
	for _, op := range n.Operands {
		if op.Name == name {
			return op.Value
		}
	}
	return nil
}

func typeSpecEqual(a, b typeSpec) bool {
	if a == nil || b == nil {
		return a == b
	}
	av, bv := a.toJSONValue(), b.toJSONValue()
	return fmtJSON(av) == fmtJSON(bv)
}

func fmtJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// isNullish reports a null operand: bare, or a synthesized cast of one.
func isNullish(e elm.Expression) bool {
	return nullOf(e) != nil
}

// nullOf returns the null inside a bare or synthesized-cast null, or nil.
func nullOf(e elm.Expression) elm.Expression {
	if isBareNull(e) {
		return e
	}
	if as, ok := e.(*elm.AsNode); ok && as.Strict == nil && isBareNull(as.Operand) {
		return as.Operand
	}
	return nil
}

// isUntypedList reports a list selector with no element that has a type: an
// empty list, or one holding only nulls.
func isUntypedList(e elm.Expression) bool {
	l, ok := e.(*elm.ListNode)
	if !ok {
		return false
	}
	for _, el := range l.Element {
		if !isBareNull(el) {
			return false
		}
	}
	return true
}

// liftListCast is how CQF passes an untyped list where a typed one is
// required: it casts each element, `X return X as T`, rather than the list.
func (t *Translator) liftListCast(list elm.Expression, elem typeSpec) elm.Expression {
	distinct := false
	return t.markConverted(&elm.QueryNode{
		Annotation: t.cqfAnnotation(),
		Source: []*elm.AliasedQuerySourceELM{{
			Annotation: t.cqfAnnotation(),
			Alias:      fhirConversionAlias,
			Expression: list,
		}},
		Let:          []*elm.LetClauseELM{},
		Relationship: []*elm.RelationshipClauseELM{},
		Return: &elm.ReturnClauseELM{
			Annotation: t.cqfAnnotation(),
			Distinct:   &distinct,
			Expression: t.castNullTo(&elm.AliasRefNode{Annotation: t.cqfAnnotation(), Name: fhirConversionAlias}, elem),
		},
	})
}

// isQuantityList reports List<Quantity>, which the statistical aggregates take
// as it is: they have a Quantity overload, so there is nothing to widen.
func isQuantityList(ts typeSpec) bool {
	lt, ok := ts.(listTS)
	return ok && namedName(lt.elem) == typesystem.TypeQuantity
}

// listSetOperators take a list operand whose element type an empty list must
// be lifted to.
var listSetOperators = map[string]bool{
	"Union": true, "Intersect": true, "Except": true,
	"Includes": true, "ProperIncludes": true, "IncludedIn": true, "ProperIncludedIn": true,
	"Contains": true, "ProperContains": true, "In": true, "ProperIn": true,
}

// listListOperators compare two lists, binding their type parameter from the
// first operand.
var listListOperators = map[string]bool{
	"Union": true, "Intersect": true, "Except": true,
	"Includes": true, "ProperIncludes": true, "IncludedIn": true, "ProperIncludedIn": true,
}

// sameTupleShape reports whether two tuple types have the same element names.
func sameTupleShape(a, b tupleTS) bool {
	if len(a.fields) != len(b.fields) {
		return false
	}
	names := make(map[string]bool, len(a.fields))
	for _, f := range a.fields {
		names[f.name] = true
	}
	for _, f := range b.fields {
		if !names[f.name] {
			return false
		}
	}
	return true
}

// typedNull is a synthesized null that CQF types under result types — the
// endIndex of the Slice that Skip and Tail lower to is an Integer.
func (t *Translator) typedNull(ts typeSpec) elm.Expression {
	n := &elm.NullNode{Annotation: t.cqfAnnotation()}
	if t.opts.EnableResultTypes {
		name, spec := resultTypeOf(ts)
		elm.SetResultType(n, name, spec)
	}
	return n
}

// inferSystemOperator gives the result type of system operators whose type
// follows from their operands in a way the generic inference does not cover.
// It returns nil for anything it does not handle.
func (t *Translator) inferSystemOperator(e elm.Expression) typeSpec {
	switch v := e.(type) {
	case *elm.MinMaxValueNode:
		if v.ValueType != "" {
			return namedTS{v.ValueType}
		}
	case *elm.NamedOperatorExpressionNode:
		switch v.Operator {
		case "Message", "Slice":
			return t.inferTypeSpec(namedOperand(v, "source"))
		case "Descendents", "Children":
			return listTS{anyTS}
		}
	case *elm.OperatorExpressionNode:
		if len(v.Operand) == 0 {
			return nil
		}
		first := t.inferTypeSpec(v.Operand[0])
		switch v.Operator {
		case "HighBoundary", "LowBoundary":
			return first
		case "Indexer":
			if namedName(first) == typesystem.TypeString {
				return strTS
			}
		case "Expand":
			// Expanding an interval yields its points; expanding a list of
			// intervals yields intervals.
			switch ft := first.(type) {
			case intervalTS:
				return listTS{ft.point}
			case listTS:
				return ft
			}
		case "Coalesce":
			// Coalesce over a single list picks one of its elements.
			if len(v.Operand) == 1 {
				if lt, ok := first.(listTS); ok {
					return lt.elem
				}
			}
		}
	case *elm.UnaryExpressionNode:
		switch v.Operator {
		case "Precision":
			return intTS
		case "ToConcept":
			return namedTS{typesystem.TypeConcept}
		case "PointFrom":
			if iv, ok := t.inferTypeSpec(v.Operand).(intervalTS); ok {
				return iv.point
			}
		}
	}
	return nil
}

// castChoiceArgument casts an argument to a function's declared choice
// parameter. FHIR choice elements are wide — Condition.onset is one of five
// types — and a helper usually declares the few it handles, so CQF casts the
// argument: As(C.onset, Choice<Period, dateTime>). The cast is synthesized: no
// locator and no result type. (issues/07)
func (t *Translator) castChoiceArgument(arg elm.Expression, declared typeSpec) elm.Expression {
	want, ok := declared.(choiceTS)
	if !ok {
		return arg
	}
	// A choice argument already contained in the parameter's choice passes as
	// it is; anything else — a narrower choice's superset, or a single-typed
	// argument such as "Sys"('a') — is cast to the declared choice.
	argTS := knownTS(t.inferTypeSpec(arg))
	if argTS == nil {
		return arg
	}
	if got, ok := argTS.(choiceTS); ok && choiceSubset(got, want) {
		return arg
	}
	spec := toELMTypeSpecifier(want)
	if spec == nil {
		return arg
	}
	return t.markConverted(&elm.AsNode{
		Annotation:      t.cqfAnnotation(),
		Signature:       t.cqfEmptyArrayField(),
		Operand:         arg,
		AsTypeSpecifier: spec,
	})
}

// promoteDateIntervalOperands converts a Date interval selector's bounds to
// DateTime when the other operand is an Interval<DateTime>. The rebuilt
// interval is synthesized (promoteIntervalBounds drops its locator and type).
func (t *Translator) promoteDateIntervalOperands(a, b elm.Expression) (elm.Expression, elm.Expression) {
	if isDateTimeInterval(t.typeOf(b)) {
		a = t.promoteIntervalBounds(a)
	}
	if isDateTimeInterval(t.typeOf(a)) {
		b = t.promoteIntervalBounds(b)
	}
	return a, b
}

func isDateTimeInterval(ts typeSpec) bool {
	iv, ok := ts.(intervalTS)
	return ok && namedName(iv.point) == typesystem.TypeDateTime
}

func (t *Translator) hasDateAndDateTime(es []elm.Expression) bool {
	date, dateTime := false, false
	for _, e := range es {
		if e == nil {
			continue
		}
		date = date || t.isDateTS(e)
		dateTime = dateTime || t.isDateTimeTS(e)
	}
	return date && dateTime
}

// choiceSubset reports whether every alternative of a is one of b's.
func choiceSubset(a, b choiceTS) bool {
	for _, x := range a.alts {
		found := false
		for _, y := range b.alts {
			if typeSpecEqual(x, y) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isListTS(ts typeSpec) bool {
	_, ok := ts.(listTS)
	return ok
}

// presentOperands lists a constructor's supplied components, in order.
func presentOperands(es ...elm.Expression) []elm.Expression {
	out := make([]elm.Expression, 0, len(es))
	for _, e := range es {
		if e != nil {
			out = append(out, e)
		}
	}
	return out
}

func sameExprs(a, b []elm.Expression) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// expandValueSet wraps a value set in the ExpandValueSet CQF inserts when a
// value set is used as a list of codes. The node is synthesized: no locator.
func (t *Translator) expandValueSet(vs elm.Expression) elm.Expression {
	return t.markConverted(&elm.UnaryExpressionNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.computeSig("ExpandValueSet", []elm.Expression{vs}),
		Operator:   "ExpandValueSet",
		Operand:    vs,
	})
}
