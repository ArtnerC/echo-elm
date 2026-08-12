package translator

import (
	"encoding/json"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// typeSpec represents a CQL type specifier used in ELM signature output.
type typeSpec interface {
	toJSONValue() interface{}
}

// namedTS is a NamedTypeSpecifier: {"type":"NamedTypeSpecifier","name":"..."}
type namedTS struct{ name string }

func (n namedTS) toJSONValue() interface{} {
	return map[string]string{"type": "NamedTypeSpecifier", "name": n.name}
}

// intervalTS is an IntervalTypeSpecifier: {"type":"IntervalTypeSpecifier","pointType":{...}}
type intervalTS struct{ point typeSpec }

func (i intervalTS) toJSONValue() interface{} {
	return map[string]interface{}{
		"type":      "IntervalTypeSpecifier",
		"pointType": i.point.toJSONValue(),
	}
}

// listTS is a ListTypeSpecifier: {"type":"ListTypeSpecifier","elementType":{...}}
type listTS struct{ elem typeSpec }

func (l listTS) toJSONValue() interface{} {
	return map[string]interface{}{
		"type":        "ListTypeSpecifier",
		"elementType": l.elem.toJSONValue(),
	}
}

// tupleTS is a TupleTypeSpecifier with named element types.
type tupleTS struct {
	// fields preserves declaration order for stable JSON output.
	fields []tupleField
}

type tupleField struct {
	name string
	ts   typeSpec
}

func (t tupleTS) toJSONValue() interface{} {
	els := make([]interface{}, 0, len(t.fields))
	for _, f := range t.fields {
		els = append(els, map[string]interface{}{
			"name":        f.name,
			"elementType": f.ts.toJSONValue(),
		})
	}
	return map[string]interface{}{
		"type":    "TupleTypeSpecifier",
		"element": els,
	}
}

func (t tupleTS) field(name string) (typeSpec, bool) {
	for _, f := range t.fields {
		if f.name == name {
			return f.ts, true
		}
	}
	return nil, false
}

// Pre-built type spec constants.
var (
	anyTS      = namedTS{typesystem.TypeAny}
	boolTS     = namedTS{typesystem.TypeBoolean}
	intTS      = namedTS{typesystem.TypeInteger}
	decTS      = namedTS{typesystem.TypeDecimal}
	longTS     = namedTS{typesystem.TypeLong}
	strTS      = namedTS{typesystem.TypeString}
	dateTS     = namedTS{typesystem.TypeDate}
	dateTimeTS = namedTS{typesystem.TypeDateTime}
	timeTS     = namedTS{typesystem.TypeTime}
	quantityTS = namedTS{typesystem.TypeQuantity}
)

// overloadedOps lists operators that always emit an inferred signature in
// Overloads/Differing mode (regardless of operand shape).
// "In" is NOT listed here — it is handled by computeInSig with smart null-bound detection.
// Collection-polymorphic ops (Union/Intersect/Except/Count/Exists/Distinct/Flatten/First/Last)
// are listed in collectionPolymorphicOps and use operand-opaqueness rules instead.
var overloadedOps = map[string]bool{
	// Multi-type arithmetic
	"Add": true, "Subtract": true, "Multiply": true, "Divide": true,
	"Modulo": true, "Power": true, "TruncatedDivide": true,
	// Multi-type unary arithmetic
	"Negate": true, "Abs": true,
	// Type conversion (multiple input types → result type, or vice versa)
	"ToInteger": true, "ToDecimal": true, "ToString": true, "ToBoolean": true,
	"ToDate": true, "ToDateTime": true, "ToLong": true,
	"ToQuantity": true, "ToRatio": true, "ToConcept": true, "ToCode": true,
	// Polymorphic comparison
	"Equal": true, "NotEqual": true, "Equivalent": true,
	"Less": true, "LessOrEqual": true, "Greater": true, "GreaterOrEqual": true,
	// Duration/Difference between datetimes (multiple precision overloads)
	"DurationBetween": true, "DifferenceBetween": true,
	// Component extraction has Date and DateTime overloads. DateFrom, TimeFrom
	// and TimezoneOffsetFrom are DateTime-only, so they are not listed here.
	"DateTimeComponentFrom": true,
	// Age (polymorphic by FHIR vs QDM patient type)
	"CalculateAge": true, "CalculateAgeAt": true,
	// Aggregate (polymorphic by list element type)
	"Min": true, "Max": true, "Sum": true, "Avg": true,
	"Median": true, "StdDev": true, "PopulationStdDev": true,
	"Variance": true, "PopulationVariance": true,
	"GeometricMean": true, "Product": true,
	// String (polymorphic by input type: String vs List)
	"Length": true,
	// Collection indexer
	"Indexer": true,
	// Before/After are also unconditionally emitted in Overloads mode.
	"Before": true, "After": true,
}

// intervalSetOps require special handling: in Overloads mode CQF emits a
// signature whenever any operand is an Interval (to disambiguate from the
// list-shape overload), and uses normal opacity rules otherwise.
var intervalSetOps = map[string]bool{
	"Union": true, "Intersect": true, "Except": true,
	"Includes": true, "IncludedIn": true,
	"ProperIncludes": true, "ProperIncludedIn": true,
}

// collectionPolymorphicOps are operators with a single overload template
// parameterized by element/point type. In Overloads mode CQF emits a
// signature only when at least one operand is opaque (i.e. its type cannot
// be derived from inline literal structure).
var collectionPolymorphicOps = map[string]bool{
	// List-shape polymorphic
	"Count": true, "Exists": true,
	"Distinct": true, "Flatten": true,
	"First": true, "Last": true, "SingletonFrom": true,
	"Contains": true, "ProperContains": true,
	// Interval-shape polymorphic (point type parameter)
	"Start": true, "End": true, "Width": true, "Size": true, "Expand": true,
	"Successor": true, "Predecessor": true,
	"Meets": true, "MeetsBefore": true, "MeetsAfter": true,
	"Overlaps": true, "OverlapsBefore": true, "OverlapsAfter": true,
	"During": true, "SameAs": true, "SameOrBefore": true, "SameOrAfter": true,
	"Within": true, "Starts": true, "Ends": true,
	// Single-overload string conversion
	"ToTime": true,
}

// fixedFormalTypes provides pre-determined formal parameter types for specific operators
// when the signature level is All. The caller uses these directly rather than inferring
// from the actual operand expressions.
var fixedFormalTypes = map[string][]typeSpec{
	"IsNull":  {anyTS},
	"IsTrue":  {boolTS},
	"IsFalse": {boolTS},
	"Not":     {boolTS},
	"And":     {boolTS, boolTS},
	"Or":      {boolTS, boolTS},
	"Xor":     {boolTS, boolTS},
	"Implies": {boolTS, boolTS},
}

// neverSigOps lists operators that always emit an empty signature array,
// regardless of the signature level setting. These are operators where
// CQF unconditionally omits signature inference.
var neverSigOps = map[string]bool{
	// Binary string concatenation via `+` is folded to Concatenate, which
	// CQF always serialises with signature: [] even in All mode.
	"Concatenate": true,
}

// fixedResultTypes gives the result type of every operator whose type does not
// depend on its operands. Operators whose result follows an operand (arithmetic,
// set operations, Coalesce, …) are resolved from the operand instead and are not
// listed here.
var fixedResultTypes = map[string]typeSpec{
	// Boolean-valued
	"Not": boolTS, "IsNull": boolTS, "IsTrue": boolTS, "IsFalse": boolTS,
	"Exists": boolTS, "IsIncludedIn": boolTS,
	"ProperIn": boolTS, "ProperContains": boolTS,
	"ProperIncludes": boolTS, "ProperIncludedIn": boolTS,
	"Starts": boolTS, "Ends": boolTS, "During": boolTS, "Within": boolTS,
	"Equivalent": boolTS, "NotEquivalent": boolTS,
	"AllTrue": boolTS, "AnyTrue": boolTS, "InValueSet": boolTS, "InCodeSystem": boolTS,
	"AnyInValueSet": boolTS, "AnyInCodeSystem": boolTS,
	"CanConvertQuantity": boolTS, "SameOrOverlaps": boolTS,
	"ConvertsToBoolean": boolTS, "ConvertsToDate": boolTS, "ConvertsToDateTime": boolTS,
	"ConvertsToDecimal": boolTS, "ConvertsToInteger": boolTS, "ConvertsToLong": boolTS,
	"ConvertsToQuantity": boolTS, "ConvertsToRatio": boolTS, "ConvertsToString": boolTS,
	"ConvertsToTime": boolTS,

	// String-valued
	"Upper": strTS, "Lower": strTS, "Substring": strTS, "Combine": strTS,
	"ReplaceMatches": strTS, "Concatenate": strTS, "ToString": strTS,

	// Integer-valued
	"IndexOf": intTS, "PositionOf": intTS, "LastPositionOf": intTS,
	"Length": intTS, "Count": intTS,
	"ToInteger": intTS, "Truncate": intTS, "Floor": intTS, "Ceiling": intTS,
	"TimezoneOffsetFrom": decTS,

	// Decimal-valued
	"ToDecimal": decTS, "Ln": decTS, "Log": decTS, "Exp": decTS, "Sqrt": decTS,

	// Date/time-valued
	"ToDate": dateTS, "DateFrom": dateTS,
	"ToDateTime": dateTimeTS, "Now": dateTimeTS,
	"ToTime": timeTS, "TimeFrom": timeTS, "TimeOfDay": timeTS,
	"Today": dateTS,

	// Other scalar
	"ToQuantity": quantityTS, "ToLong": longTS, "ToBoolean": boolTS,
	"Message": anyTS,

	// Collection-valued
	"Split": listTS{strTS}, "SplitOnMatches": listTS{strTS},
}

// computeSig builds the ELM signature JSON for an operator and its translated operands.
// Returns an empty JSON array ("[]") for operators that never emit signatures in the
// current signature level.
func (t *Translator) computeSig(op string, operands []elm.Expression) json.RawMessage {
	// Operators in neverSigOps always get [].
	if neverSigOps[op] {
		return t.cqfEmptyArrayField()
	}

	level := t.opts.SignatureLevel
	if level == "" || level == "None" {
		return t.cqfEmptyArrayField()
	}

	// FunctionRef (user-defined function call) always gets an inferred signature
	// in Overloads mode and above.
	if op == "FunctionRef" {
		return t.buildInferredSig(operands)
	}

	switch level {
	case "Overloads", "Differing":
		if overloadedOps[op] {
			return t.buildInferredSig(operands)
		}
		if intervalSetOps[op] {
			if hasIntervalOperand(operands) || t.anyOperandOpaque(operands) {
				return t.buildInferredSig(operands)
			}
			return t.cqfEmptyArrayField()
		}
		if collectionPolymorphicOps[op] {
			if t.anyOperandOpaque(operands) {
				return t.buildInferredSig(operands)
			}
			return t.cqfEmptyArrayField()
		}
		return t.cqfEmptyArrayField()

	case "All":
		// Fixed formal types take precedence.
		if specs, ok := fixedFormalTypes[op]; ok {
			return t.buildFixedSig(specs)
		}
		// Decimal aggregate ops always report List<Decimal> as their formal parameter type.
		if decimalAggregateOps[op] {
			return t.buildFixedSig([]typeSpec{listTS{decTS}})
		}
		// Coalesce: variadic. CQF emits the inferred type repeated N times when all
		// operands have a concrete inferred type; otherwise variadic Any.
		if op == "Coalesce" {
			var firstTS typeSpec = anyTS
			allConcrete := len(operands) > 0
			for i, o := range operands {
				its := t.inferTypeSpec(o)
				if its == nil {
					allConcrete = false
					break
				}
				if n, ok := its.(namedTS); ok && n.name == anyTS.name {
					allConcrete = false
					break
				}
				if i == 0 {
					firstTS = its
				}
			}
			if allConcrete {
				return t.buildVariadicSig(len(operands), firstTS)
			}
			return t.buildVariadicSig(len(operands), anyTS)
		}
		// Collapse/Expand: 2nd operand (precision) has formal type Quantity.
		if (op == "Collapse" || op == "Expand") && len(operands) == 2 {
			return t.buildFixedSigOverride(operands, map[int]typeSpec{1: quantityTS})
		}
		// All other operators report inferred types.
		return t.buildInferredSig(operands)
	}

	return t.cqfEmptyArrayField()
}

// buildFixedSigOverride emits an inferred signature, but overrides individual
// positions with explicit formal types from `overrides` (position → typeSpec).
func (t *Translator) buildFixedSigOverride(operands []elm.Expression, overrides map[int]typeSpec) json.RawMessage {
	specs := make([]typeSpec, len(operands))
	for i, op := range operands {
		if ts, ok := overrides[i]; ok {
			specs[i] = ts
		} else {
			specs[i] = t.inferTypeSpec(op)
		}
	}
	return t.buildFixedSig(specs)
}

// computeInSig handles the "In" operator signature, which follows special rules
// in Overloads mode: only emit a signature when list-promotion was applied or
// when the interval RHS has a null bound (requiring null coercion at runtime).
// computeFunctionRefSig builds the signature for a FunctionRef. At
// SignatureLevel=Overloads the signature only disambiguates overloaded calls, so
// a function defined exactly once in this library gets none. Calls into included
// libraries keep the signature: their definitions are not fully resolved here,
// and the common case (FHIRHelpers.ToString, ToDateTime, …) is overloaded.
func (t *Translator) computeFunctionRefSig(libraryName, name string, operands []elm.Expression) json.RawMessage {
	level := t.opts.SignatureLevel
	if level == "Overloads" || level == "Differing" {
		if libraryName == "" && t.localFuncCounts[name] == 1 {
			return t.cqfEmptyArrayField()
		}
	}
	return t.computeSig("FunctionRef", operands)
}

// computeInSig builds the signature for an ELM In node. demoted reports that
// the node came from rewriting `included in` / `during` into In rather than from
// a literal `in`.
//
// At Overloads/Differing, CQF records the resolved signature when a coercion was
// needed to reach the chosen overload.
//
// CQF also emits a signature for some In expressions where no coercion happened
// — `2 in {1,2,3}` and `"Day" included in "Period"` both carry one while
// `5 in Interval[1,10]` and `"Day" in "Period"` do not. The discriminator is not
// yet understood; see the sig-overloads note on elm-nodes/MembershipOperators.cql.
func (t *Translator) computeInSig(lhs, rhs elm.Expression, listPromoted bool) json.RawMessage {
	level := t.opts.SignatureLevel
	if level == "" || level == "None" {
		return t.cqfEmptyArrayField()
	}
	if level == "All" {
		return t.buildInferredSig([]elm.Expression{lhs, rhs})
	}
	if listPromoted || hasNullIntervalBound(rhs) {
		return t.buildInferredSig([]elm.Expression{lhs, rhs})
	}
	return t.cqfEmptyArrayField()
}

// hasNullIntervalBound reports whether an ELM expression is an IntervalNode
// whose low or high bound is absent (nil), a NullNode, or an AsNode wrapping
// a NullNode.
func hasNullIntervalBound(e elm.Expression) bool {
	iv, ok := e.(*elm.IntervalNode)
	if !ok {
		return false
	}
	if iv.Low == nil || iv.High == nil {
		return true
	}
	if isNullLike(iv.Low) || isNullLike(iv.High) {
		return true
	}
	return false
}

func isNullLike(e elm.Expression) bool {
	switch v := e.(type) {
	case *elm.NullNode:
		return true
	case *elm.AsNode:
		return isNullLike(v.Operand)
	}
	return false
}

// buildSigFromTypes returns a signature array using known formal types when
// the current signature level requires one, and an empty array otherwise.
// Use this for synthetic wrappers where the operand types are known a priori
// but inferTypeSpec on the translated operand cannot recover them.
func (t *Translator) buildSigFromTypes(op string, specs []typeSpec) json.RawMessage {
	if neverSigOps[op] {
		return t.cqfEmptyArrayField()
	}
	level := t.opts.SignatureLevel
	if level == "" || level == "None" {
		return t.cqfEmptyArrayField()
	}
	if level == "Overloads" || level == "Differing" {
		if !overloadedOps[op] && op != "FunctionRef" {
			return t.cqfEmptyArrayField()
		}
	}
	return t.buildFixedSig(specs)
}

// anyOperandOpaque returns true if any operand is opaque. For single-arg
// operators (e.g. Start, End, Width, Size, Successor, Predecessor), an
// IntervalNode whose bounds are all literals is treated as transparent
// (its point type is visible inline). For multi-arg operators (e.g. Union,
// Intersect, Except, IncludedIn, Includes, Contains, Before, After),
// IntervalNode operands are always opaque because CQF emits a signature
// to distinguish the list-shape overload from the interval-shape overload.
func (t *Translator) anyOperandOpaque(operands []elm.Expression) bool {
	unary := len(operands) == 1
	for _, op := range operands {
		if unary {
			if t.operandIsOpaqueUnary(op) {
				return true
			}
		} else {
			if t.operandIsOpaqueMulti(op) {
				return true
			}
		}
	}
	return false
}

// operandIsOpaqueUnary applies the unary opacity rule: IntervalNode with
// fully-literal bounds is transparent.
func (t *Translator) operandIsOpaqueUnary(e elm.Expression) bool {
	switch v := e.(type) {
	case *elm.ListNode:
		if v.TypeSpec != nil {
			return true
		}
		if _, ok := t.listNodeTypes[v]; ok {
			return true
		}
		return len(v.Element) == 0
	case *elm.IntervalNode:
		if v.Low != nil {
			if _, ok := v.Low.(*elm.LiteralNode); !ok {
				return true
			}
		}
		if v.High != nil {
			if _, ok := v.High.(*elm.LiteralNode); !ok {
				return true
			}
		}
		return false
	}
	return false
}

// operandIsOpaqueMulti applies the multi-arg opacity rule. ListNode uses the
// typed/empty-element rule; IntervalNode is transparent iff both bounds are
// literal expressions (point type visible inline).
func (t *Translator) operandIsOpaqueMulti(e elm.Expression) bool {
	switch v := e.(type) {
	case *elm.ListNode:
		if v.TypeSpec != nil {
			return true
		}
		if _, ok := t.listNodeTypes[v]; ok {
			return true
		}
		return len(v.Element) == 0
	case *elm.IntervalNode:
		if v.Low != nil {
			if _, ok := v.Low.(*elm.LiteralNode); !ok {
				return true
			}
		}
		if v.High != nil {
			if _, ok := v.High.(*elm.LiteralNode); !ok {
				return true
			}
		}
		return false
	}
	return false
}

// hasIntervalOperand reports whether any operand is an IntervalNode literal
// (a fully-syntactic Interval[low, high] expression).
func hasIntervalOperand(operands []elm.Expression) bool {
	for _, op := range operands {
		if _, ok := op.(*elm.IntervalNode); ok {
			return true
		}
	}
	return false
}

// buildInferredSig builds a signature array by inferring the type of each operand.
func (t *Translator) buildInferredSig(operands []elm.Expression) json.RawMessage {
	specs := make([]interface{}, 0, len(operands))
	for _, op := range operands {
		specs = append(specs, t.inferTypeSpec(op).toJSONValue())
	}
	b, _ := json.Marshal(specs)
	return json.RawMessage(b)
}

// buildFixedSig builds a signature array from pre-determined formal types.
func (t *Translator) buildFixedSig(specs []typeSpec) json.RawMessage {
	vals := make([]interface{}, 0, len(specs))
	for _, s := range specs {
		vals = append(vals, s.toJSONValue())
	}
	b, _ := json.Marshal(vals)
	return json.RawMessage(b)
}

// buildVariadicSig builds a signature array with n copies of the same type.
func (t *Translator) buildVariadicSig(n int, ts typeSpec) json.RawMessage {
	vals := make([]interface{}, n)
	for i := range vals {
		vals[i] = ts.toJSONValue()
	}
	b, _ := json.Marshal(vals)
	return json.RawMessage(b)
}

// astTypeSpecToTypeSpec converts an AST type specifier to the translator's
// internal typeSpec used for signature inference. Returns nil for tuple/choice
// or other unhandled forms.
func astTypeSpecToTypeSpec(ts ast.TypeSpecifier) typeSpec {
	if ts == nil {
		return nil
	}
	switch v := ts.(type) {
	case *ast.NamedTypeSpecifier:
		return namedTS{resolveTypeName(v.Name)}
	case *ast.ListTypeSpecifier:
		inner := astTypeSpecToTypeSpec(v.ElementType)
		if inner == nil {
			inner = anyTS
		}
		return listTS{inner}
	case *ast.IntervalTypeSpecifier:
		inner := astTypeSpecToTypeSpec(v.PointType)
		if inner == nil {
			inner = anyTS
		}
		return intervalTS{inner}
	case *ast.TupleTypeSpecifier:
		fields := make([]tupleField, 0, len(v.Elements))
		for _, el := range v.Elements {
			if el == nil {
				continue
			}
			fts := astTypeSpecToTypeSpec(el.Type)
			if fts == nil {
				fts = anyTS
			}
			fields = append(fields, tupleField{name: el.Name, ts: fts})
		}
		return tupleTS{fields: fields}
	}
	return nil
}

// elmTypeSpecToTypeSpec converts an ELM TypeSpecifier (model) to the
// translator's internal typeSpec used for signature inference.
func elmTypeSpecToTypeSpec(ts elm.TypeSpecifier) typeSpec {
	if ts == nil {
		return nil
	}
	switch v := ts.(type) {
	case *elm.NamedTypeSpecifier:
		if v.Name == "" {
			return nil
		}
		return namedTS{v.Name}
	case *elm.IntervalTypeSpecifier:
		inner := elmTypeSpecToTypeSpec(v.PointType)
		if inner == nil {
			inner = anyTS
		}
		return intervalTS{inner}
	case *elm.ListTypeSpecifier:
		inner := elmTypeSpecToTypeSpec(v.ElementType)
		if inner == nil {
			inner = anyTS
		}
		return listTS{inner}
	case *elm.TupleTypeSpecifier:
		fields := make([]tupleField, 0, len(v.Element))
		for _, el := range v.Element {
			if el == nil {
				continue
			}
			fts := elmTypeSpecToTypeSpec(el.ElementType)
			if fts == nil {
				fts = anyTS
			}
			fields = append(fields, tupleField{name: el.Name, ts: fts})
		}
		return tupleTS{fields: fields}
	}
	return nil
}

// inferTypeSpec infers the ELM type specifier for a translated expression node.
// Used to build operation signatures from actual argument types.
func (t *Translator) inferTypeSpec(e elm.Expression) typeSpec {
	if e == nil {
		return anyTS
	}
	switch v := e.(type) {
	case *elm.LiteralNode:
		if v.ValueType != "" {
			return namedTS{v.ValueType}
		}
	case *elm.NullNode:
		return anyTS
	case *elm.QuantityNode:
		return quantityTS
	case *elm.RatioNode:
		return namedTS{typesystem.TypeRatio}
	case *elm.CodeNode:
		return namedTS{typesystem.TypeCode}
	case *elm.ConceptNode:
		return namedTS{typesystem.TypeConcept}
	case *elm.DateNode:
		return dateTS
	case *elm.DateTimeNode:
		return dateTimeTS
	case *elm.TimeNode:
		return timeTS
	case *elm.UnaryExpressionNode:
		switch v.Operator {
		case "ToDecimal":
			return decTS
		case "ToInteger":
			return intTS
		case "ToLong":
			return longTS
		case "ToString":
			return strTS
		case "ToBoolean":
			return boolTS
		case "ToDate":
			return dateTS
		case "ToDateTime":
			return dateTimeTS
		case "ToTime":
			return timeTS
		case "ToQuantity":
			return quantityTS
		case "ToList":
			// ToList(scalar) → List<typeof(scalar)>
			return listTS{t.inferTypeSpec(v.Operand)}
		case "Negate", "Abs", "Successor", "Predecessor":
			// These preserve the operand type.
			return t.inferTypeSpec(v.Operand)
		case "Truncate", "Floor", "Ceiling":
			// Decimal → Integer: these round to a whole number.
			return intTS
		case "Start", "End":
			// Start/End of an Interval returns the point type.
			if iv, ok := t.inferTypeSpec(v.Operand).(intervalTS); ok {
				return iv.point
			}
		case "Width":
			if iv, ok := t.inferTypeSpec(v.Operand).(intervalTS); ok {
				return iv.point
			}
			return quantityTS
		case "Size":
			return quantityTS
		case "IsNull", "IsTrue", "IsFalse", "Not", "Exists":
			return boolTS
		case "Length":
			return intTS
		case "First", "Last", "SingletonFrom":
			if lt, ok := t.inferTypeSpec(v.Operand).(listTS); ok {
				return lt.elem
			}
		}
		if ts, ok := fixedResultTypes[v.Operator]; ok {
			return ts
		}
	case *elm.ListNode:
		if v.TypeSpec != nil {
			if ts := elmTypeSpecToTypeSpec(v.TypeSpec); ts != nil {
				return listTS{ts}
			}
		}
		if ts, ok := t.listNodeTypes[v]; ok {
			return listTS{ts}
		}
		if len(v.Element) > 0 {
			// Prefer the first non-null element for type inference.
			for _, el := range v.Element {
				if !isNullLike(el) {
					return listTS{t.nodeResultTS(el)}
				}
			}
			return listTS{t.nodeResultTS(v.Element[0])}
		}
		return listTS{anyTS}
	case *elm.AggregateExpressionNode:
		// An aggregate reduces List<T> to a scalar. Most yield T; the
		// statistical ones are Decimal-valued regardless of element type, and
		// Count is always Integer.
		switch v.Operator {
		case "Count":
			return intTS
		case "Avg", "Median", "StdDev", "PopulationStdDev", "Variance",
			"PopulationVariance", "GeometricMean":
			return decTS
		case "AllTrue", "AnyTrue":
			return boolTS
		}
		if lt, ok := t.inferTypeSpec(v.Source).(listTS); ok {
			return lt.elem
		}
		return anyTS
	case *elm.IntervalNode:
		if v.Low != nil {
			if _, isNull := v.Low.(*elm.NullNode); !isNull {
				return intervalTS{t.inferTypeSpec(v.Low)}
			}
		}
		if v.High != nil {
			if _, isNull := v.High.(*elm.NullNode); !isNull {
				return intervalTS{t.inferTypeSpec(v.High)}
			}
		}
		return intervalTS{anyTS}
	case *elm.AsNode:
		if v.AsType != "" {
			return namedTS{v.AsType}
		}
		if v.AsTypeSpecifier != nil {
			if ts := elmTypeSpecToTypeSpec(v.AsTypeSpecifier); ts != nil {
				return ts
			}
		}
	case *elm.IsNode:
		return boolTS
	case *elm.IfNode:
		return t.inferTypeSpec(v.Then)
	case *elm.CaseNode:
		// The result is the common type of the branches; the first resolvable
		// then-branch stands in for it, falling back to the else.
		for _, ci := range v.CaseItem {
			if ci == nil || ci.Then == nil {
				continue
			}
			if ts := t.inferTypeSpec(ci.Then); !isAnyTS(ts) {
				return ts
			}
		}
		if v.Else != nil {
			return t.inferTypeSpec(v.Else)
		}
	case *elm.ParameterRefNode:
		if ts, ok := t.paramTypeSpecs[v.Name]; ok {
			return ts
		}
		if name, ok := t.paramTypes[v.Name]; ok {
			return namedTS{name}
		}
	case *elm.OperandRefNode:
		if ts, ok := t.operandTypeSpecs[v.Name]; ok {
			return ts
		}
	case *elm.ExpressionRefNode:
		if v.LibraryName != "" {
			if ts, ok := t.libDefTypes[v.LibraryName][v.Name]; ok {
				return ts
			}
			break
		}
		if ts, ok := t.defTypeSpecs[v.Name]; ok {
			// A definition declared in a retrieve context evaluates to one value
			// per context member when referenced from Unfiltered, so the
			// reference is list-valued there.
			if t.inUnfilteredContext() {
				if declared := t.defContexts[v.Name]; declared != "" && declared != "Unfiltered" {
					return listTS{ts}
				}
			}
			return ts
		}
	case *elm.IdentifierRefNode:
		// Only reachable from a sort-by expression, where the identifier names a
		// field of the query's result element ($this being the element itself).
		if t.sortElementTS != nil {
			if v.Name == "$this" {
				return t.sortElementTS
			}
			if tt, ok := t.sortElementTS.(tupleTS); ok {
				for _, f := range tt.fields {
					if f.name == v.Name {
						return f.ts
					}
				}
			}
		}
	case *elm.CodeRefNode:
		return namedTS{typesystem.TypeCode}
	case *elm.ConceptRefNode:
		return namedTS{typesystem.TypeConcept}
	case *elm.CodeSystemRefNode:
		return namedTS{typesystem.TypeCodeSystem}
	case *elm.ValueSetRefNode:
		return namedTS{typesystem.TypeValueSet}
	case *elm.AliasRefNode:
		// Walk alias scopes from innermost outward.
		for i := len(t.queryAliasTypeSpecs) - 1; i >= 0; i-- {
			if ts, ok := t.queryAliasTypeSpecs[i][v.Name]; ok {
				return ts
			}
		}
	case *elm.LetRefNode:
		for i := len(t.queryLetTypeSpecs) - 1; i >= 0; i-- {
			if ts, ok := t.queryLetTypeSpecs[i][v.Name]; ok {
				return ts
			}
		}
	case *elm.QueryNode:
		// An aggregate clause reduces the query to a single value.
		if v.Aggregate != nil && v.Aggregate.Expression != nil {
			return t.inferTypeSpec(v.Aggregate.Expression)
		}
		// Query result type = List<returnElementType>; default = List<sourceElementType>.
		if v.Return != nil && v.Return.Expression != nil {
			return listTS{t.nodeResultTS(v.Return.Expression)}
		}
		if len(v.Source) > 0 && v.Source[0] != nil {
			if lt, ok := t.nodeResultTS(v.Source[0].Expression).(listTS); ok {
				return lt
			}
		}
		return listTS{anyTS}
	case *elm.OperatorExpressionNode:
		if v.Operator == "Indexer" && len(v.Operand) > 0 {
			if lt, ok := t.inferTypeSpec(v.Operand[0]).(listTS); ok {
				return lt.elem
			}
		}
		// Arithmetic ops preserve the (promoted) operand type — both operands
		// have been coerced to the same type by the translator.
		switch v.Operator {
		case "Add", "Subtract", "Multiply", "Divide", "Modulo", "TruncatedDivide", "Power":
			if len(v.Operand) > 0 {
				return t.inferTypeSpec(v.Operand[0])
			}
		case "Equal", "NotEqual", "Less", "LessOrEqual", "Greater", "GreaterOrEqual",
			"And", "Or", "Xor", "Implies", "In", "Contains", "Includes", "IncludedIn",
			"Before", "After", "Meets", "MeetsBefore", "MeetsAfter", "Overlaps",
			"OverlapsBefore", "OverlapsAfter", "SameAs", "SameOrBefore", "SameOrAfter",
			"StartsWith", "EndsWith", "Matches":
			return boolTS
		case "Concatenate":
			return strTS
		case "Coalesce":
			// The result is the common type of the alternatives: an untyped null
			// among them leaves no common type but Any.
			for _, op := range v.Operand {
				if isNullLike(op) {
					return anyTS
				}
			}
			if len(v.Operand) > 0 {
				return t.inferTypeSpec(v.Operand[0])
			}
		case "Union", "Intersect", "Except":
			if len(v.Operand) > 0 {
				return t.inferTypeSpec(v.Operand[0])
			}
		case "Collapse", "Expand", "Distinct", "Flatten", "Slice", "Take", "Skip", "Tail":
			if len(v.Operand) > 0 {
				return t.inferTypeSpec(v.Operand[0])
			}
		case "Width":
			if len(v.Operand) > 0 {
				if iv, ok := t.inferTypeSpec(v.Operand[0]).(intervalTS); ok {
					return iv.point
				}
			}
		}
		if ts, ok := fixedResultTypes[v.Operator]; ok {
			return ts
		}
	case *elm.NamedOperatorExpressionNode:
		if ts, ok := fixedResultTypes[v.Operator]; ok {
			return ts
		}
		// Round preserves its operand's numeric type.
		if v.Operator == "Round" && len(v.Operands) > 0 {
			return t.inferTypeSpec(v.Operands[0].Value)
		}
	case *elm.CalculateAgeNode:
		return intTS
	case *elm.UnaryPrecisionOperatorNode:
		// DateTimeComponentFrom extracts a numeric component.
		if ts, ok := fixedResultTypes[v.Operator]; ok {
			return ts
		}
		return intTS
	case *elm.PrecisionOperatorNode:
		if ts, ok := fixedResultTypes[v.Operator]; ok {
			return ts
		}
		// CalculateAgeAt, DurationBetween and similar return Integer.
		return intTS
	case *elm.InValueSetNode:
		return boolTS
	case *elm.FunctionRefNode:
		// A call to a function declared in this library resolves to its
		// declared return type.
		if v.LibraryName == "" {
			if ts, ok := t.localFuncReturns[v.Name]; ok {
				return ts
			}
		} else if ts, ok := t.libFuncReturns[v.LibraryName][v.Name]; ok {
			// A call into an included library resolves to the return type that
			// library infers for the function, the same way a qualified
			// ExpressionRef resolves through libDefTypes. Fluent calls reach here
			// too, since their libraryName is recovered rather than written.
			return ts
		}
		// FHIRHelpers.ToString (and other primitive helpers) return primitive types.
		if v.LibraryName == t.fhirHelpersLocalName {
			switch v.Name {
			case "ToString", "ToCode", "ToConcept":
				return strTS
			case "ToDateTime":
				return dateTimeTS
			case "ToDate":
				return dateTS
			case "ToTime":
				return timeTS
			case "ToBoolean":
				return boolTS
			case "ToInteger":
				return intTS
			case "ToDecimal":
				return decTS
			case "ToQuantity":
				return quantityTS
			}
		}
	case *elm.TupleNode:
		fields := make([]tupleField, 0, len(v.Element))
		for _, el := range v.Element {
			fields = append(fields, tupleField{name: el.Name, ts: t.inferTypeSpec(el.Value)})
		}
		return tupleTS{fields: fields}
	case *elm.PropertyNode:
		// Property on a tuple type: look up the field type.
		var srcTS typeSpec
		var srcContext string
		if v.Source != nil {
			srcTS = t.inferTypeSpec(v.Source)
			if er, ok := v.Source.(*elm.ExpressionRefNode); ok {
				srcContext = er.Name
			}
		} else if v.Scope != "" {
			for i := len(t.queryAliasTypeSpecs) - 1; i >= 0; i-- {
				if ts, ok := t.queryAliasTypeSpecs[i][v.Scope]; ok {
					srcTS = ts
					break
				}
			}
		}
		if tt, ok := srcTS.(tupleTS); ok && v.Path != "" {
			if ts, ok := tt.field(v.Path); ok {
				return ts
			}
		}
		// FHIR/QICore property-type lookup: when source is a context expression
		// (e.g. "Patient"), consult FHIRPropertyType for the CQL type.
		if srcContext != "" && v.Path != "" {
			key := srcContext + "." + v.Path
			if fhirType, ok := typesystem.FHIRPropertyType[key]; ok {
				return fhirTypeToCQLTypeSpec(fhirType)
			}
		}
	case *elm.RetrieveNode:
		if v.DataType != "" {
			return listTS{namedTS{v.DataType}}
		}
		return listTS{anyTS}
	}
	return anyTS
}

// fhirTypeToCQLTypeSpec maps a FHIR type name (from FHIRPropertyType) to the CQL
// typeSpec that CQF resolves it to. The mapping runs through the System type of
// the FHIR type's "value" element, so it covers real primitives (FHIR.code) and
// binding types (FHIR.AdministrativeGender) alike — both resolve to String.
func fhirTypeToCQLTypeSpec(fhirType string) typeSpec {
	switch typesystem.FHIRPrimitiveValueType[fhirType] {
	case "Boolean":
		return boolTS
	case "Integer":
		return intTS
	case "Long":
		return longTS
	case "Decimal":
		return decTS
	case "String":
		return strTS
	case "DateTime":
		return dateTimeTS
	case "Date":
		return dateTS
	case "Time":
		return timeTS
	case "Quantity":
		return quantityTS
	default:
		return anyTS
	}
}

// toELMTypeSpecifier converts an inferred typeSpec into the ELM TypeSpecifier
// node used for resultTypeSpecifier. Named types return nil: they are recorded
// in resultTypeName instead.
func toELMTypeSpecifier(ts typeSpec) elm.TypeSpecifier {
	switch v := ts.(type) {
	case namedTS:
		return &elm.NamedTypeSpecifier{Name: v.name}
	case listTS:
		return &elm.ListTypeSpecifier{ElementType: toELMTypeSpecifier(v.elem)}
	case intervalTS:
		return &elm.IntervalTypeSpecifier{PointType: toELMTypeSpecifier(v.point)}
	case tupleTS:
		els := make([]*elm.TupleElementDefinition, 0, len(v.fields))
		for _, f := range v.fields {
			els = append(els, &elm.TupleElementDefinition{
				Name:        f.name,
				ElementType: toELMTypeSpecifier(f.ts),
			})
		}
		return &elm.TupleTypeSpecifier{Element: els}
	}
	return nil
}

// resultTypeOf splits an inferred typeSpec into the (resultTypeName,
// resultTypeSpecifier) pair CQF records. A named type goes in the name; every
// structured type goes in the specifier. An unresolved type yields neither —
// stamping Any where CQF resolved a concrete type would be worse than omitting.
func resultTypeOf(ts typeSpec) (string, elm.TypeSpecifier) {
	if ts == nil {
		return "", nil
	}
	if n, ok := ts.(namedTS); ok {
		if n.name == "" || n.name == anyTS.name {
			return "", nil
		}
		return n.name, nil
	}
	return "", toELMTypeSpecifier(ts)
}

// stampTypeSpecifier records the type a source-declared type specifier denotes.
// CQF stamps these (an `as` target, an operand or parameter type, and their
// nested element/point types) but not the specifiers it synthesizes to carry
// result-type information — those are type metadata, not nodes from the source.
func stampTypeSpecifier(spec elm.TypeSpecifier) {
	if spec == nil {
		return
	}
	ts := elmTypeSpecToTypeSpec(spec)
	switch v := spec.(type) {
	case *elm.NamedTypeSpecifier:
		v.ResultTypeName = v.Name
	case *elm.IntervalTypeSpecifier:
		stampTypeSpecifier(v.PointType)
		_, v.ResultTypeSpecifier = resultTypeOf(ts)
	case *elm.ListTypeSpecifier:
		stampTypeSpecifier(v.ElementType)
		_, v.ResultTypeSpecifier = resultTypeOf(ts)
	case *elm.TupleTypeSpecifier:
		for _, el := range v.Element {
			stampTypeSpecifier(el.ElementType)
		}
		_, v.ResultTypeSpecifier = resultTypeOf(ts)
	case *elm.ChoiceTypeSpecifier:
		for _, c := range v.Choice {
			stampTypeSpecifier(c)
		}
	}
}

// nodeResultTS returns the type already recorded on a node, falling back to
// inference. Preferring the recorded value matters for FHIR properties, whose
// node carries the FHIR type while inference reports the type it converts to.
func (t *Translator) nodeResultTS(e elm.Expression) typeSpec {
	if name, spec := elm.GetResultType(e); name != "" || spec != nil {
		if name != "" {
			return namedTS{name}
		}
		if ts := elmTypeSpecToTypeSpec(spec); ts != nil {
			return ts
		}
	}
	return t.inferTypeSpec(e)
}

// isAnyTS reports whether ts is the unresolved/Any named type.
func isAnyTS(ts typeSpec) bool {
	n, ok := ts.(namedTS)
	return ok && n.name == anyTS.name
}

// inUnfilteredContext reports whether the statement being translated is in the
// Unfiltered context (the default when none is declared).
func (t *Translator) inUnfilteredContext() bool {
	return t.currentContextName == "" || t.currentContextName == "Unfiltered"
}
