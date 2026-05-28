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
func (t *Translator) computeInSig(lhs, rhs elm.Expression, listPromoted bool) json.RawMessage {
	level := t.opts.SignatureLevel
	if level == "" || level == "None" {
		return t.cqfEmptyArrayField()
	}
	if level == "All" {
		return t.buildInferredSig([]elm.Expression{lhs, rhs})
	}
	// Overloads/Differing: emit signature only when coercion is required.
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

// operandIsOpaque retained for backward-compat with anything still calling it.
func (t *Translator) operandIsOpaque(e elm.Expression) bool { return t.operandIsOpaqueMulti(e) }

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
		case "Negate", "Abs", "Truncate", "Floor", "Ceiling", "Successor", "Predecessor":
			// Numeric unary ops preserve operand type.
			return t.inferTypeSpec(v.Operand)
		case "Start", "End":
			// Start/End of an Interval returns the point type.
			if iv, ok := t.inferTypeSpec(v.Operand).(intervalTS); ok {
				return iv.point
			}
		case "Width", "Size":
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
					return listTS{t.inferTypeSpec(el)}
				}
			}
			return listTS{t.inferTypeSpec(v.Element[0])}
		}
		return listTS{anyTS}
	case *elm.AggregateExpressionNode:
		// The source expression is the list being aggregated.
		return t.inferTypeSpec(v.Source)
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
	case *elm.IfNode:
		return t.inferTypeSpec(v.Then)
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
		if ts, ok := t.defTypeSpecs[v.Name]; ok {
			return ts
		}
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
		// Query result type = List<returnElementType>; default = List<sourceElementType>.
		if v.Return != nil && v.Return.Expression != nil {
			return listTS{t.inferTypeSpec(v.Return.Expression)}
		}
		if len(v.Source) > 0 && v.Source[0] != nil {
			if lt, ok := t.inferTypeSpec(v.Source[0].Expression).(listTS); ok {
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
			// Result is the (common) type of the first non-null operand; approximate
			// with the first operand's type.
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
	case *elm.CalculateAgeNode:
		return intTS
	case *elm.PrecisionOperatorNode:
		// CalculateAgeAt and similar return Integer.
		return intTS
	case *elm.InValueSetNode:
		return boolTS
	case *elm.FunctionRefNode:
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

// fhirTypeToCQLTypeSpec maps a FHIR primitive type name (from FHIRPropertyType)
// to the CQL typeSpec that CQF emits for the post-FHIRHelpers-coerced value.
func fhirTypeToCQLTypeSpec(fhirType string) typeSpec {
	switch fhirType {
	case "boolean":
		return boolTS
	case "integer", "positiveInt", "unsignedInt":
		return intTS
	case "decimal":
		return decTS
	case "string", "code", "uri", "url", "canonical", "oid", "id", "markdown", "base64Binary", "xhtml":
		return strTS
	case "dateTime", "instant":
		return dateTimeTS
	case "date":
		return dateTS
	case "time":
		return timeTS
	default:
		return anyTS
	}
}
