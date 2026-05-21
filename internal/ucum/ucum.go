// Package ucum provides lightweight UCUM (Unified Code for Units of
// Measure, https://ucum.org/) syntactic + atom validation suitable for the
// CQL Quantity unit field.
//
// This is not a full UCUM unit-conversion engine. It validates that a unit
// string matches the UCUM §2 syntax and that every atom is either:
//   - a recognized UCUM base/derived unit code, or
//   - a recognized UCUM prefix followed by a base/derived unit (e.g. "mg"),
//   - a calendar duration keyword permitted by CQL (year, month, week, day,
//     hour, minute, second, millisecond) and their singular/plural forms,
//   - the dimensionless code "1", or
//   - an annotation block "{...}" attached to an otherwise-valid atom.
//
// Reference: UCUM §2 grammar (https://ucum.org/ucum#section-Syntax-Rules).
package ucum

import (
	"strings"
	"unicode"
)

// Validate reports whether s is a syntactically and lexically valid UCUM unit.
// Empty strings are treated as the dimensionless unit ("1") and accepted.
// Use Validate when a translator option such as ValidateUnits is true.
func Validate(s string) error {
	if s == "" || s == "1" {
		return nil
	}
	// Strip annotation blocks {...} — UCUM allows them as comments attached
	// to atoms; they do not affect the unit's meaning.
	stripped := stripAnnotations(s)
	if stripped == "" {
		return nil
	}
	terms := splitTerms(stripped)
	for _, term := range terms {
		atom, _ := splitAtomExponent(term)
		if atom == "" {
			return errUnit("empty atom in: " + s)
		}
		if !isValidAtom(atom) {
			return errUnit("unknown UCUM atom: " + atom + " (in " + s + ")")
		}
	}
	return nil
}

// IsCQLTemporalKeyword reports whether s is one of the CQL calendar-duration
// keywords (year, month, ..., millisecond and their plurals). CQL allows
// these in Quantity literals alongside UCUM codes.
func IsCQLTemporalKeyword(s string) bool {
	_, ok := cqlTemporal[strings.ToLower(s)]
	return ok
}

type unitError struct{ msg string }

func (e *unitError) Error() string { return e.msg }
func errUnit(m string) error       { return &unitError{msg: m} }

// stripAnnotations removes UCUM annotation blocks {...} from a unit string.
// Annotations are comments and do not change validity beyond the requirement
// that they are well-formed (balanced curly braces).
func stripAnnotations(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// splitTerms splits a UCUM expression by "." and "/" — the only term-level
// operators in UCUM §2. Parentheses are preserved as whole atoms.
func splitTerms(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	flush := func() {
		t := strings.TrimSpace(cur.String())
		if t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}
	for _, r := range s {
		switch {
		case r == '(':
			depth++
			cur.WriteRune(r)
		case r == ')':
			if depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case (r == '.' || r == '/') && depth == 0:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// splitAtomExponent separates a UCUM term into its atom prefix and signed
// integer exponent suffix, e.g. "m2" → ("m", "2"), "kg-1" → ("kg", "-1").
// Parenthesized sub-expressions are returned as a single recursive atom.
func splitAtomExponent(s string) (string, string) {
	// Parenthesized sub-expression: validate recursively as a UCUM unit.
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		inner := s[1 : len(s)-1]
		return inner, ""
	}
	// Walk backwards over the optional sign + digits suffix.
	i := len(s)
	for i > 0 && unicode.IsDigit(rune(s[i-1])) {
		i--
	}
	if i > 0 && (s[i-1] == '+' || s[i-1] == '-') {
		i--
	}
	if i == len(s) {
		return s, ""
	}
	// Don't strip a sign if no digits followed it.
	if i == 0 {
		return s, ""
	}
	return s[:i], s[i:]
}

// isValidAtom reports whether atom is a recognized UCUM unit atom, possibly
// prefixed by a UCUM SI prefix, a CQL temporal keyword, the dimensionless
// code "1", or a parenthesized sub-expression.
func isValidAtom(atom string) bool {
	if atom == "1" {
		return true
	}
	if _, ok := cqlTemporal[strings.ToLower(atom)]; ok {
		return true
	}
	// Parenthesized sub-expression must itself validate.
	if strings.HasPrefix(atom, "(") && strings.HasSuffix(atom, ")") {
		return Validate(atom[1:len(atom)-1]) == nil
	}
	if _, ok := ucumAtoms[atom]; ok {
		return true
	}
	// Try stripping a UCUM prefix and re-checking the remainder.
	for prefix := range ucumPrefixes {
		if strings.HasPrefix(atom, prefix) && len(atom) > len(prefix) {
			rest := atom[len(prefix):]
			if _, ok := ucumAtoms[rest]; ok {
				return true
			}
		}
	}
	return false
}

// cqlTemporal is the set of CQL calendar-duration keywords (CQL 1.5.3 §3.5).
var cqlTemporal = map[string]bool{
	"year": true, "years": true,
	"month": true, "months": true,
	"week": true, "weeks": true,
	"day": true, "days": true,
	"hour": true, "hours": true,
	"minute": true, "minutes": true,
	"second": true, "seconds": true,
	"millisecond": true, "milliseconds": true,
}

// ucumPrefixes is the UCUM §4 set of metric prefixes. Listed as code → factor
// (factor not used for validation; kept for documentation).
var ucumPrefixes = map[string]int{
	"Y": 24, "Z": 21, "E": 18, "P": 15, "T": 12,
	"G": 9, "M": 6, "k": 3, "h": 2, "da": 1,
	"d": -1, "c": -2, "m": -3, "u": -6, "n": -9,
	"p": -12, "f": -15, "a": -18, "z": -21, "y": -24,
	// Binary prefixes (UCUM §4.4):
	"Ki": 0, "Mi": 0, "Gi": 0, "Ti": 0,
}

// ucumAtoms is the subset of UCUM §6/§7 unit atoms commonly used in clinical
// quality measures. Coverage is intentionally pragmatic: the goal is to flag
// obviously-wrong units (e.g. "shab-shab-shab") without requiring a full
// 1000-row table. Unknown but well-formed atoms can be added on demand.
var ucumAtoms = map[string]bool{
	// Base units
	"m": true, "s": true, "g": true, "K": true, "C": true, "rad": true,
	"mol": true, "cd": true,
	// Common SI derived
	"Hz": true, "N": true, "Pa": true, "J": true, "W": true,
	"V": true, "F": true, "Ohm": true, "S": true, "Wb": true,
	"T": true, "H": true, "lm": true, "lx": true, "Bq": true,
	"Gy": true, "Sv": true,
	// Time
	"min": true, "h": true, "d": true, "wk": true, "mo": true, "a": true,
	// Length / area / volume
	"L": true, "l": true, "Ar": true, "ar": true,
	"in_i": true, "ft_i": true, "yd_i": true, "mi_i": true,
	// Mass
	"t": true,
	// Pressure
	"bar": true, "mm[Hg]": true, "cm[H2O]": true,
	// Dimensionless / counting
	"%": true, "[ppth]": true, "[ppm]": true, "[ppb]": true, "[pptr]": true,
	// Clinical / lab common
	"U": true, "[IU]": true, "[iU]": true, "Cel": true, "[degF]": true,
	"eq": true, "osm": true, "kat": true,
	"[arb'U]": true, "[USP'U]": true,
	"[in_i'Hg]": true, "[psi]": true,
	"meq": true, "ueq": true,
	"[drp]": true,
	"[tbs_us]": true, "[tsp_us]": true, "[cup_us]": true,
	// FHIR commonly emits these:
	"[pH]": true,
	"[lb_av]": true, "[oz_av]": true,
	"[ft_us]": true, "[in_us]": true, "[mi_us]": true,
}
