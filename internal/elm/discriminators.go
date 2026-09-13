package elm

import (
	"encoding/json"
	"fmt"
)

// AddTypeDiscriminators rewrites ELM JSON into the fully type-discriminated form
// produced by the CQF JAXB/MOXy `elm-json` writer — the shape that appears
// inside FHIR Library resources, and the reason `--target bundle` is named for
// where the shape is observed rather than for any consumer that wants it.
//
// It is not required by ELM consumers. The .NET SDK that reads ELM out of these
// resources writes the lean shape itself, and on read treats this one as legacy
// input: the synthetic `type` property is declared only to be validated and
// discarded, and the container converter recognises the "Library$" prefix purely
// so it can skip it. Producing this shape for a schema-driven deserializer would
// be backwards; issues/04 documents that correction in full.
//
// What it is genuinely for is parity. `parity --bundle` and `parity --ref-dir`
// cannot produce a meaningful diff while the two sides are in different
// serializer shapes — every bundle-sourced fixture otherwise drowns in
// implied-discriminator noise. On that path prefer StripImpliedTypes, which
// reduces the reference down rather than inflating echo-elm's output up; this
// direction exists for writing ELM back into a bundle.
//
// The cql-to-elm CLI omits `"type"` on declaration, container and clause nodes
// because their position already determines what they are. This pass adds them
// back from that same positional information, so it needs no help from the
// translator and cannot perturb default (CLI-parity) output.
//
// Nodes that already carry a `"type"` are left alone, so expressions, FunctionDef,
// the With/Without relationship clauses and the sort-by variants keep theirs.
func AddTypeDiscriminators(elmJSON []byte) ([]byte, error) {
	var root map[string]any
	if err := json.Unmarshal(elmJSON, &root); err != nil {
		return nil, fmt.Errorf("elm: parse ELM JSON: %w", err)
	}
	lib, ok := root["library"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("elm: input is not an ELM library envelope")
	}
	setType(lib, "Library")
	annotate(lib, "Library")

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("elm: marshal ELM JSON: %w", err)
	}
	return out, nil
}

// impliedTypes maps a (containing node type, field name) pair to the ELM type of
// the node or nodes in that field. This is the positional knowledge the CLI
// serializer relies on and the discriminated form spells out.
var impliedTypes = map[[2]string]string{
	// Library header
	{"Library", "identifier"}:       "VersionedIdentifier",
	{"Library", "schemaIdentifier"}: "VersionedIdentifier",
	{"Library", "usings"}:           "Library$Usings",
	{"Library", "includes"}:         "Library$Includes",
	{"Library", "parameters"}:       "Library$Parameters",
	{"Library", "codeSystems"}:      "Library$CodeSystems",
	{"Library", "valueSets"}:        "Library$ValueSets",
	{"Library", "codes"}:            "Library$Codes",
	{"Library", "concepts"}:         "Library$Concepts",
	{"Library", "contexts"}:         "Library$Contexts",
	{"Library", "statements"}:       "Library$Statements",

	// The definitions inside each container
	{"Library$Usings", "def"}:      "UsingDef",
	{"Library$Includes", "def"}:    "IncludeDef",
	{"Library$Parameters", "def"}:  "ParameterDef",
	{"Library$CodeSystems", "def"}: "CodeSystemDef",
	{"Library$ValueSets", "def"}:   "ValueSetDef",
	{"Library$Codes", "def"}:       "CodeDef",
	{"Library$Concepts", "def"}:    "ConceptDef",
	{"Library$Contexts", "def"}:    "ContextDef",
	{"Library$Statements", "def"}:  "ExpressionDef",
	{"ExpressionDef", "operand"}:   "OperandDef",
	{"FunctionDef", "operand"}:     "OperandDef",
	{"CodeDef", "codeSystem"}:      "CodeSystemRef",
	{"ConceptDef", "code"}:         "CodeRef",
	{"ValueSetDef", "codeSystem"}:  "CodeSystemRef",

	// Query clauses
	{"Query", "source"}:    "AliasedQuerySource",
	{"Query", "let"}:       "LetClause",
	{"Query", "return"}:    "ReturnClause",
	{"Query", "sort"}:      "SortClause",
	{"Query", "aggregate"}: "AggregateClause",

	// Other clause and element nodes
	{"Ratio", "numerator"}:            "Quantity",
	{"Ratio", "denominator"}:          "Quantity",
	{"Case", "caseItem"}:              "CaseItem",
	{"InValueSet", "valueset"}:        "ValueSetRef",
	{"InCodeSystem", "codesystem"}:    "CodeSystemRef",
	{"TupleTypeSpecifier", "element"}: "TupleElementDefinition",
	{"Tuple", "element"}:              "TupleElement",
	{"Instance", "element"}:           "InstanceElement",
	{"AnyInValueSet", "valueset"}:     "ValueSetRef",
	{"AnyInCodeSystem", "codesystem"}: "CodeSystemRef",
}

// annotate walks the children of a node whose ELM type is parentType, stamping
// each field whose type its position implies, then recursing.
func annotate(node map[string]any, parentType string) {
	for key, child := range node {
		implied := impliedTypes[[2]string{parentType, key}]
		switch v := child.(type) {
		case map[string]any:
			if implied != "" {
				setType(v, implied)
			}
			annotate(v, typeOf(v))
		case []any:
			for _, item := range v {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if implied != "" {
					setType(m, implied)
				}
				annotate(m, typeOf(m))
			}
		}
	}
}

// setType records a node's ELM type, leaving any existing one in place: a
// statement that already says FunctionDef must not be relabelled ExpressionDef.
func setType(node map[string]any, name string) {
	if _, present := node["type"]; present {
		return
	}
	node["type"] = name
}

// typeOf returns a node's ELM type, or "" when it carries none.
func typeOf(node map[string]any) string {
	t, _ := node["type"].(string)
	return t
}

// StripImpliedTypes is the inverse of AddTypeDiscriminators: it removes every
// `"type"` whose value the node's position already determines, reducing the
// JAXB/MOXy bundle shape back to the lean shape the cql-to-elm CLI writes.
//
// This is the direction the parity harness wants. Inflating echo-elm's output up
// to the reference makes the comparison depend on impliedTypes staying in sync
// with a Java class hierarchy; reducing the reference down cannot lose anything,
// because a discriminator is only removed when it agrees with what position
// already implies. A `"type"` that disagrees is left in place — that is a real
// difference (a FunctionDef sitting where an ExpressionDef is implied) and the
// diff must still show it.
//
// Applied to CLI-shaped ELM it is a no-op, so it is safe on every comparison
// path rather than only the bundle one.
func StripImpliedTypes(elmJSON []byte) ([]byte, error) {
	var root map[string]any
	if err := json.Unmarshal(elmJSON, &root); err != nil {
		return nil, fmt.Errorf("elm: parse ELM JSON: %w", err)
	}
	lib, ok := root["library"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("elm: input is not an ELM library envelope")
	}
	clearType(lib, "Library")
	deannotate(lib, "Library")

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("elm: marshal ELM JSON: %w", err)
	}
	return out, nil
}

// StripImpliedTypesTree is StripImpliedTypes over an already-decoded tree, for
// callers that are mid-normalization and would otherwise round-trip through JSON
// twice. It accepts any node, not just a library envelope.
func StripImpliedTypesTree(v any) {
	if lib, ok := v.(map[string]any); ok {
		if inner, ok := lib["library"].(map[string]any); ok {
			clearType(inner, "Library")
			deannotate(inner, "Library")
			return
		}
	}
	deannotate2(v, "")
}

// deannotate walks a node whose own type is parentType, removing implied
// discriminators from its children.
func deannotate(node map[string]any, parentType string) {
	for key, child := range node {
		implied := impliedTypes[[2]string{parentType, key}]
		switch v := child.(type) {
		case map[string]any:
			// Read the child's type before clearing it: the walk into the child
			// is keyed on what the child actually is.
			actual := typeOf(v)
			clearType(v, implied)
			deannotate(v, orImplied(actual, implied))
		case []any:
			for _, item := range v {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				actual := typeOf(m)
				clearType(m, implied)
				deannotate(m, orImplied(actual, implied))
			}
		}
	}
}

// deannotate2 walks a subtree of unknown parentage, used when a caller hands in
// something other than a library envelope.
func deannotate2(v any, parentType string) {
	switch node := v.(type) {
	case map[string]any:
		deannotate(node, parentType)
	case []any:
		for _, item := range node {
			deannotate2(item, parentType)
		}
	}
}

// clearType removes a node's type only when it matches what position implies.
// A disagreeing discriminator is meaningful and stays.
func clearType(node map[string]any, implied string) {
	if implied == "" {
		return
	}
	if t, ok := node["type"].(string); ok && t == implied {
		delete(node, "type")
	}
}

// orImplied returns the node's own type, falling back to the implied one when
// the node carries none — which is exactly the case after stripping.
func orImplied(actual, implied string) string {
	if actual != "" {
		return actual
	}
	return implied
}
