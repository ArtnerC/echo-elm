package translator

// System operator tables, the drift check against internal/elmops, and literal selectors.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/elmops"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// dateTimeComponentOperators maps the `<component> from x` precisions that have
// their own ELM operator instead of being a DateTimeComponentFrom precision.
// Every other precision (Year..Millisecond) uses DateTimeComponentFrom.
var dateTimeComponentOperators = map[string]string{
	"Date":           "DateFrom",
	"Time":           "TimeFrom",
	"TimezoneOffset": "TimezoneOffsetFrom",
}

// nullarySystemOps are the CQL built-ins that are ELM operators with no operands.
var nullarySystemOps = map[string]bool{
	"Now": true, "Today": true, "TimeOfDay": true,
}

// unarySystemOps is the set of CQL built-in function names that map to single-operand
// ELM operator expressions (i.e., "operand" is a single object, not an array).
var unarySystemOps = map[string]bool{
	// Arithmetic
	"Abs": true, "Ceiling": true, "Floor": true, "Truncate": true,
	"Exp": true, "Ln": true, "Negate": true,
	"Successor": true, "Predecessor": true,
	"Sqrt": true,
	// Boolean
	"Not": true, "IsNull": true, "IsTrue": true, "IsFalse": true,
	// Collection
	"Exists": true, "SingletonFrom": true,
	"Distinct": true, "Flatten": true,
	"Length": true,
	// String
	"Upper": true, "Lower": true,
	// Type conversions
	"ToBoolean": true, "ToDate": true, "ToDateTime": true, "ToDecimal": true,
	"ToInteger": true, "ToLong": true, "ToList": true, "ToQuantity": true,
	"ToString": true, "ToTime": true, "ToRatio": true, "ToConcept": true,
	// Type conversion checks
	"ConvertsToBoolean": true, "ConvertsToDate": true, "ConvertsToDateTime": true,
	"ConvertsToDecimal": true, "ConvertsToInteger": true, "ConvertsToLong": true,
	"ConvertsToQuantity": true, "ConvertsToRatio": true, "ConvertsToString": true,
	"ConvertsToTime": true,
	// Interval operators
	"Width": true, "Start": true, "End": true,
	"Precision": true,
}

// aggregateSystemOps are CQL built-in aggregate functions that map to ELM AggregateExpression
// nodes. ELM AggregateExpression uses "source" (single object) not "operand".
var aggregateSystemOps = map[string]bool{
	"Count": true, "Sum": true, "Product": true,
	"Min": true, "Max": true,
	"Avg": true, "Median": true, "Mode": true,
	"StdDev": true, "PopulationStdDev": true,
	"Variance": true, "PopulationVariance": true,
	"AllTrue": true, "AnyTrue": true,
	"First": true, "Last": true,
}

// decimalAggregateOps are aggregate functions that CQF always wraps their argument
// in a ToDecimal query to coerce List<T> → List<Decimal>.
var decimalAggregateOps = map[string]bool{
	"Avg": true, "Median": true, "StdDev": true, "PopulationStdDev": true,
	"Variance": true, "PopulationVariance": true,
}

// namedOperatorOps maps CQL built-in operator names with named operand fields
// to the JSON field names CQF emits for each operand slot, in order.
var namedOperatorOps = map[string][]string{
	"Substring":      {"stringToSub", "startIndex", "length"},
	"Combine":        {"source", "separator"},
	"Split":          {"stringToSplit", "separator"},
	"PositionOf":     {"pattern", "string"},
	"LastPositionOf": {"pattern", "string"},
	"Round":          {"operand", "precision"},
	"IndexOf":        {"source", "element"},
	"Message":        {"source", "condition", "code", "severity", "message"},
	"Descendents":    {"source"},
	"Children":       {"source"},
}

// binaryOperandOps are CQL built-in operators that emit "operand": [a, b]
// as a generic operator expression (no named fields).
var binaryOperandOps = map[string]bool{
	"StartsWith": true, "EndsWith": true, "Matches": true,
	"Power": true, "Log": true,
	"Contains": true,
	"Indexer":  true, "HighBoundary": true, "LowBoundary": true,
}

// naryOperandOps are CQL built-in operators emitted as "operand": [a, b, ...]
// for any arity.
var naryOperandOps = map[string]bool{
	"ReplaceMatches": true,
	"Concatenate":    true,
}

// convertToOps maps a System named type to the ELM unary operator emitted by
// CQF for `convert x to <Type>` syntax. Built-in types are normalized to their
// operator-specific form instead of a generic Convert node.
var convertToOps = map[string]string{
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

// Init-time consistency check: every entry in the translator's local maps
// must be present (or be an explicit translator-internal alias) in the
// shared elmops registry. This keeps internal/elmops as the single source
// of truth and surfaces drift as a build failure.
func init() {
	for name := range unarySystemOps {
		if !elmops.IsUnary(name) {
			panic("translator: unarySystemOps drift; missing/wrong kind in elmops: " + name)
		}
	}
	for name := range aggregateSystemOps {
		if !elmops.IsAggregate(name) {
			panic("translator: aggregateSystemOps drift; missing/wrong kind in elmops: " + name)
		}
	}
	for name := range namedOperatorOps {
		if elmops.NamedFields(name) == nil {
			panic("translator: namedOperatorOps drift; missing NamedFields in elmops: " + name)
		}
	}
	for from, to := range convertToOps {
		if elmops.SystemConvertToOps[from] != to {
			panic("translator: convertToOps drift for " + from + " → " + to)
		}
	}
}

// intLiteral builds an Integer literal ELM node from an int value.
func (t *Translator) intLiteral(v int) elm.Expression {
	return &elm.LiteralNode{
		Annotation: t.cqfAnnotation(),
		ValueType:  typesystem.TypeInteger,
		Value:      fmt.Sprintf("%d", v),
	}
}

// parseDateLiteral converts an ast.DateLiteral value string (YYYY[-MM[-DD]]) to a DateNode.
func (t *Translator) parseDateLiteral(value string) *elm.DateNode {
	node := &elm.DateNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
	}
	parts := strings.Split(value, "-")
	if len(parts) >= 1 && parts[0] != "" {
		if y, err := strconv.Atoi(parts[0]); err == nil {
			node.Year = t.intLiteral(y)
		}
	}
	if len(parts) >= 2 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			node.Month = t.intLiteral(m)
		}
	}
	if len(parts) >= 3 {
		if d, err := strconv.Atoi(parts[2]); err == nil {
			node.Day = t.intLiteral(d)
		}
	}
	return node
}

// parseTimeParts parses a time string (HH[:MM[:SS[.mmm]]]) into component expressions.
// Returns nil for absent components.
func (t *Translator) parseTimeParts(s string) (hour, minute, second, millisecond elm.Expression) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) >= 1 && parts[0] != "" {
		if h, err := strconv.Atoi(parts[0]); err == nil {
			hour = t.intLiteral(h)
		}
	}
	if len(parts) >= 2 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			minute = t.intLiteral(m)
		}
	}
	if len(parts) >= 3 {
		secStr := parts[2]
		if dotIdx := strings.Index(secStr, "."); dotIdx >= 0 {
			milStr := secStr[dotIdx+1:]
			secStr = secStr[:dotIdx]
			if ms, err := strconv.Atoi(milStr); err == nil {
				millisecond = t.intLiteral(ms)
			}
		}
		if s2, err := strconv.Atoi(secStr); err == nil {
			second = t.intLiteral(s2)
		}
	}
	return
}

// parseTimeLiteral converts an ast.TimeLiteral value string (HH[:MM[:SS[.mmm]]]) to a TimeNode.
func (t *Translator) parseTimeLiteral(value string) *elm.TimeNode {
	node := &elm.TimeNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
	}
	node.Hour, node.Minute, node.Second, node.Millisecond = t.parseTimeParts(value)
	return node
}

// parseDateTimeLiteral converts an ast.DateTimeLiteral value string (YYYY[-MM[-DD]]T[time][tz]) to a DateTimeNode.
func (t *Translator) parseDateTimeLiteral(value string) *elm.DateTimeNode {
	node := &elm.DateTimeNode{
		Annotation: t.cqfAnnotation(),
		Signature:  t.cqfEmptyArrayField(),
	}

	tIdx := strings.Index(value, "T")
	datePart := value
	timePart := ""
	if tIdx >= 0 {
		datePart = value[:tIdx]
		timePart = value[tIdx+1:]
	}

	// Parse date components.
	if datePart != "" {
		dateParts := strings.Split(datePart, "-")
		if len(dateParts) >= 1 && dateParts[0] != "" {
			if y, err := strconv.Atoi(dateParts[0]); err == nil {
				node.Year = t.intLiteral(y)
			}
		}
		if len(dateParts) >= 2 {
			if m, err := strconv.Atoi(dateParts[1]); err == nil {
				node.Month = t.intLiteral(m)
			}
		}
		if len(dateParts) >= 3 {
			if d, err := strconv.Atoi(dateParts[2]); err == nil {
				node.Day = t.intLiteral(d)
			}
		}
	}

	// Parse time and timezone components.
	if timePart != "" {
		var tzStr string

		// Detect timezone suffix: trailing Z or ±HH:MM after the time digits.
		if strings.HasSuffix(timePart, "Z") {
			tzStr = "0.0"
			timePart = timePart[:len(timePart)-1]
		} else if plusIdx := strings.LastIndexAny(timePart, "+-"); plusIdx > 0 {
			sign := 1.0
			if timePart[plusIdx] == '-' {
				sign = -1.0
			}
			tzParts := strings.SplitN(timePart[plusIdx+1:], ":", 2)
			timePart = timePart[:plusIdx]
			var hrs, mins float64
			if len(tzParts) >= 1 {
				if h, err := strconv.ParseFloat(tzParts[0], 64); err == nil {
					hrs = h
				}
			}
			if len(tzParts) >= 2 {
				if m, err := strconv.ParseFloat(tzParts[1], 64); err == nil {
					mins = m / 60.0
				}
			}
			offset := sign * (hrs + mins)
			tzStr = strconv.FormatFloat(offset, 'f', -1, 64)
			if !strings.Contains(tzStr, ".") {
				tzStr += ".0"
			}
		}

		node.Hour, node.Minute, node.Second, node.Millisecond = t.parseTimeParts(timePart)

		if tzStr != "" {
			node.TimezoneOffset = &elm.LiteralNode{
				Annotation: t.cqfAnnotation(),
				ValueType:  typesystem.TypeDecimal,
				Value:      tzStr,
			}
		}
	}

	return node
}

// negatedEqualityOps maps the CQL negated-equality operators to the ELM
// operator that CQF wraps in Not(...).
var negatedEqualityOps = map[string]string{
	"NotEqual":      "Equal",
	"NotEquivalent": "Equivalent",
}

// sameTypeBinaryOps are the binary operators whose operands share one type, so a
// bare null operand takes the other operand's. `1 = null` resolves Equal(Integer,
// Integer) and CQF records that by wrapping the null in As(Integer). Operators
// whose operands differ in kind — In, Contains, the interval inclusions — are
// deliberately absent: there the null would need a different type than its
// sibling's, and guessing it would be worse than leaving it bare.
var sameTypeBinaryOps = map[string]bool{
	"Equal": true, "Equivalent": true,
	"Less": true, "LessOrEqual": true, "Greater": true, "GreaterOrEqual": true,
	"Add": true, "Subtract": true, "Multiply": true, "Divide": true,
	"Modulo": true, "TruncatedDivide": true, "Power": true,
	"And": true, "Or": true, "Xor": true, "Implies": true,
}
