// Package elmops is the central registry of CQL 1.5.3 → ELM R1 operator
// mappings used by the translator. It is the single source of truth for:
//
//   - which ELM "type" name to emit for a given CQL operator string,
//   - the structural kind of that ELM expression node (Unary / Operator /
//     Aggregate / NamedFields / Precision),
//   - the named operand slots, when applicable.
//
// The registry is intentionally read-only and self-contained so the
// translator, tests, and any external tooling (e.g. lints, docs gen)
// reference exactly the same mapping. See references/specs/cql/1.5.3.
package elmops

// Kind classifies the ELM JSON/XML node shape for a given operator.
type Kind int

const (
	// KindUnknown is the zero value; callers should treat it as "not in registry".
	KindUnknown Kind = iota
	// KindUnary emits as UnaryExpression with single "operand".
	KindUnary
	// KindOperator emits as generic OperatorExpression with "operand": [a, b, ...].
	KindOperator
	// KindAggregate emits as AggregateExpression with "source" (object).
	KindAggregate
	// KindNamed emits with explicit named operand slots (see NamedFields).
	KindNamed
	// KindPrecision emits as PrecisionOperator (carries Precision attribute).
	KindPrecision
)

// Op describes one CQL 1.5.3 operator and its ELM emission contract.
type Op struct {
	// Name is the ELM operator type name (e.g., "Add", "Substring", "Count").
	Name string
	// Kind is the ELM node shape.
	Kind Kind
	// NamedFields is the ordered list of JSON field names for KindNamed.
	NamedFields []string
}

// All returns a copy of the registry, keyed by ELM operator name.
func All() map[string]Op {
	out := make(map[string]Op, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}

// Lookup returns the registry entry for an ELM operator name and whether it
// was found.
func Lookup(name string) (Op, bool) {
	op, ok := registry[name]
	return op, ok
}

// IsUnary reports whether name is a unary system operator.
func IsUnary(name string) bool {
	op, ok := registry[name]
	return ok && op.Kind == KindUnary
}

// IsAggregate reports whether name is an aggregate system operator.
func IsAggregate(name string) bool {
	op, ok := registry[name]
	return ok && op.Kind == KindAggregate
}

// IsBinaryOperand reports whether name is a binary operator that emits
// operand:[a,b] without named slots.
func IsBinaryOperand(name string) bool {
	op, ok := registry[name]
	return ok && op.Kind == KindOperator
}

// NamedFields returns the ordered named field slots for an operator emitted
// with KindNamed (e.g., Substring → [stringToSub, startIndex, length]).
// Returns nil if the operator isn't a KindNamed entry.
func NamedFields(name string) []string {
	op, ok := registry[name]
	if !ok || op.Kind != KindNamed {
		return nil
	}
	return op.NamedFields
}

// registry is the canonical CQL 1.5.3 → ELM R1 operator mapping.
// Sources: HL7 CQL 1.5.3 §9-§22, ELM R1 schema (expression.xsd /
// clinicalexpression.xsd), CQFramework cqf-tests reference output.
var registry = map[string]Op{
	// ---- Arithmetic (CQL §15) ----
	"Abs":         {Name: "Abs", Kind: KindUnary},
	"Ceiling":     {Name: "Ceiling", Kind: KindUnary},
	"Floor":       {Name: "Floor", Kind: KindUnary},
	"Truncate":    {Name: "Truncate", Kind: KindUnary},
	"Exp":         {Name: "Exp", Kind: KindUnary},
	"Ln":          {Name: "Ln", Kind: KindUnary},
	"Negate":      {Name: "Negate", Kind: KindUnary},
	"Successor":   {Name: "Successor", Kind: KindUnary},
	"Predecessor": {Name: "Predecessor", Kind: KindUnary},
	"Sqrt":        {Name: "Sqrt", Kind: KindUnary},
	"Add":         {Name: "Add", Kind: KindOperator},
	"Subtract":    {Name: "Subtract", Kind: KindOperator},
	"Multiply":    {Name: "Multiply", Kind: KindOperator},
	"Divide":      {Name: "Divide", Kind: KindOperator},
	"TruncatedDivide": {Name: "TruncatedDivide", Kind: KindOperator},
	"Modulo":      {Name: "Modulo", Kind: KindOperator},
	"Power":       {Name: "Power", Kind: KindOperator},
	"Log":         {Name: "Log", Kind: KindOperator},
	"Round":       {Name: "Round", Kind: KindNamed, NamedFields: []string{"operand", "precision"}},

	// ---- Boolean / null (CQL §11, §16) ----
	"Not":     {Name: "Not", Kind: KindUnary},
	"IsNull":  {Name: "IsNull", Kind: KindUnary},
	"IsTrue":  {Name: "IsTrue", Kind: KindUnary},
	"IsFalse": {Name: "IsFalse", Kind: KindUnary},
	"And":     {Name: "And", Kind: KindOperator},
	"Or":      {Name: "Or", Kind: KindOperator},
	"Xor":     {Name: "Xor", Kind: KindOperator},
	"Implies": {Name: "Implies", Kind: KindOperator},
	"Coalesce": {Name: "Coalesce", Kind: KindOperator},

	// ---- Comparison (CQL §12) ----
	"Equal":           {Name: "Equal", Kind: KindOperator},
	"NotEqual":        {Name: "NotEqual", Kind: KindOperator},
	"Less":            {Name: "Less", Kind: KindOperator},
	"LessOrEqual":     {Name: "LessOrEqual", Kind: KindOperator},
	"Greater":         {Name: "Greater", Kind: KindOperator},
	"GreaterOrEqual":  {Name: "GreaterOrEqual", Kind: KindOperator},
	"Equivalent":      {Name: "Equivalent", Kind: KindOperator},
	"NotEquivalent":   {Name: "NotEquivalent", Kind: KindOperator},

	// ---- String (CQL §17) ----
	"Upper":          {Name: "Upper", Kind: KindUnary},
	"Lower":          {Name: "Lower", Kind: KindUnary},
	"Length":         {Name: "Length", Kind: KindUnary},
	"Substring":      {Name: "Substring", Kind: KindNamed, NamedFields: []string{"stringToSub", "startIndex", "length"}},
	"Combine":        {Name: "Combine", Kind: KindNamed, NamedFields: []string{"source", "separator"}},
	"Split":          {Name: "Split", Kind: KindNamed, NamedFields: []string{"stringToSplit", "separator"}},
	"PositionOf":     {Name: "PositionOf", Kind: KindNamed, NamedFields: []string{"pattern", "string"}},
	"LastPositionOf": {Name: "LastPositionOf", Kind: KindNamed, NamedFields: []string{"pattern", "string"}},
	"StartsWith":     {Name: "StartsWith", Kind: KindOperator},
	"EndsWith":       {Name: "EndsWith", Kind: KindOperator},
	"Matches":        {Name: "Matches", Kind: KindOperator},
	"ReplaceMatches": {Name: "ReplaceMatches", Kind: KindOperator},
	"Concatenate":    {Name: "Concatenate", Kind: KindOperator},

	// ---- DateTime (CQL §18) ----
	"DateFrom":     {Name: "DateFrom", Kind: KindUnary},
	"TimeFrom":     {Name: "TimeFrom", Kind: KindUnary},
	"DateTimeFromComponents": {Name: "DateTimeFromComponents", Kind: KindOperator},
	"TimeFromComponents":     {Name: "TimeFromComponents", Kind: KindOperator},

	// ---- Type conversions (CQL §22) ----
	"ToBoolean":  {Name: "ToBoolean", Kind: KindUnary},
	"ToDate":     {Name: "ToDate", Kind: KindUnary},
	"ToDateTime": {Name: "ToDateTime", Kind: KindUnary},
	"ToDecimal":  {Name: "ToDecimal", Kind: KindUnary},
	"ToInteger":  {Name: "ToInteger", Kind: KindUnary},
	"ToLong":     {Name: "ToLong", Kind: KindUnary},
	"ToList":     {Name: "ToList", Kind: KindUnary},
	"ToQuantity": {Name: "ToQuantity", Kind: KindUnary},
	"ToString":   {Name: "ToString", Kind: KindUnary},
	"ToTime":     {Name: "ToTime", Kind: KindUnary},
	"ToRatio":    {Name: "ToRatio", Kind: KindUnary},
	"ToConcept":  {Name: "ToConcept", Kind: KindUnary},

	"ConvertsToBoolean":  {Name: "ConvertsToBoolean", Kind: KindUnary},
	"ConvertsToDate":     {Name: "ConvertsToDate", Kind: KindUnary},
	"ConvertsToDateTime": {Name: "ConvertsToDateTime", Kind: KindUnary},
	"ConvertsToDecimal":  {Name: "ConvertsToDecimal", Kind: KindUnary},
	"ConvertsToInteger":  {Name: "ConvertsToInteger", Kind: KindUnary},
	"ConvertsToLong":     {Name: "ConvertsToLong", Kind: KindUnary},
	"ConvertsToQuantity": {Name: "ConvertsToQuantity", Kind: KindUnary},
	"ConvertsToRatio":    {Name: "ConvertsToRatio", Kind: KindUnary},
	"ConvertsToString":   {Name: "ConvertsToString", Kind: KindUnary},
	"ConvertsToTime":     {Name: "ConvertsToTime", Kind: KindUnary},

	// ---- Interval (CQL §19) ----
	"Width":      {Name: "Width", Kind: KindUnary},
	"Start":      {Name: "Start", Kind: KindUnary},
	"End":        {Name: "End", Kind: KindUnary},
	"Contains":   {Name: "Contains", Kind: KindOperator},
	"Overlaps":   {Name: "Overlaps", Kind: KindOperator},
	"Includes":   {Name: "Includes", Kind: KindOperator},
	"IncludedIn": {Name: "IncludedIn", Kind: KindOperator},
	"Before":     {Name: "Before", Kind: KindOperator},
	"After":      {Name: "After", Kind: KindOperator},
	"Meets":      {Name: "Meets", Kind: KindOperator},
	"MeetsBefore": {Name: "MeetsBefore", Kind: KindOperator},
	"MeetsAfter":  {Name: "MeetsAfter", Kind: KindOperator},
	"Union":       {Name: "Union", Kind: KindOperator},
	"Intersect":   {Name: "Intersect", Kind: KindOperator},
	"Except":      {Name: "Except", Kind: KindOperator},
	"Collapse":    {Name: "Collapse", Kind: KindOperator},
	"Expand":      {Name: "Expand", Kind: KindOperator},

	// ---- List (CQL §20) ----
	"Exists":        {Name: "Exists", Kind: KindUnary},
	"SingletonFrom": {Name: "SingletonFrom", Kind: KindUnary},
	"Distinct":      {Name: "Distinct", Kind: KindUnary},
	"Flatten":       {Name: "Flatten", Kind: KindUnary},
	"In":            {Name: "In", Kind: KindOperator},
	"IndexOf":       {Name: "IndexOf", Kind: KindNamed, NamedFields: []string{"source", "element"}},
	"Slice":         {Name: "Slice", Kind: KindNamed, NamedFields: []string{"source", "startIndex", "endIndex"}},
	"First":         {Name: "First", Kind: KindAggregate},
	"Last":          {Name: "Last", Kind: KindAggregate},

	// ---- Aggregates (CQL §21) ----
	"Count":              {Name: "Count", Kind: KindAggregate},
	"Sum":                {Name: "Sum", Kind: KindAggregate},
	"Product":            {Name: "Product", Kind: KindAggregate},
	"Min":                {Name: "Min", Kind: KindAggregate},
	"Max":                {Name: "Max", Kind: KindAggregate},
	"Avg":                {Name: "Avg", Kind: KindAggregate},
	"Median":             {Name: "Median", Kind: KindAggregate},
	"Mode":               {Name: "Mode", Kind: KindAggregate},
	"StdDev":             {Name: "StdDev", Kind: KindAggregate},
	"PopulationStdDev":   {Name: "PopulationStdDev", Kind: KindAggregate},
	"Variance":           {Name: "Variance", Kind: KindAggregate},
	"PopulationVariance": {Name: "PopulationVariance", Kind: KindAggregate},
	"AllTrue":            {Name: "AllTrue", Kind: KindAggregate},
	"AnyTrue":            {Name: "AnyTrue", Kind: KindAggregate},

	// ---- Precision-bearing operators ----
	"DurationBetween":   {Name: "DurationBetween", Kind: KindPrecision},
	"DifferenceBetween": {Name: "DifferenceBetween", Kind: KindPrecision},
}

// DecimalCoercedAggregates names aggregate functions whose argument CQF wraps
// in a ToDecimal query to coerce List<T> → List<Decimal>.
var DecimalCoercedAggregates = map[string]bool{
	"Avg":              true,
	"Median":           true,
	"StdDev":           true,
	"PopulationStdDev": true,
	"Variance":         true,
	"PopulationVariance": true,
}

// SystemConvertToOps maps a System named type to the unary ELM operator emitted
// for `convert x to <Type>` syntax with a built-in target type.
var SystemConvertToOps = map[string]string{
	"Boolean":  "ToBoolean",
	"Integer":  "ToInteger",
	"Long":     "ToLong",
	"Decimal":  "ToDecimal",
	"String":   "ToString",
	"Date":     "ToDate",
	"DateTime": "ToDateTime",
	"Time":     "ToTime",
	"Quantity": "ToQuantity",
	"Ratio":    "ToRatio",
	"Concept":  "ToConcept",
}
