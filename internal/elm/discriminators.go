package elm

import (
	"encoding/json"
	"fmt"
)

// AddTypeDiscriminators rewrites ELM JSON into the fully type-discriminated form
// produced by the JAXB/MOXy `elm-json` writer — the shape that appears inside
// FHIR Library resources and that the Firely CQL SDK deserializes.
//
// The cql-to-elm CLI omits `"type"` on declaration, container and clause nodes
// because their position already determines what they are; a schema-driven
// deserializer that dispatches on `"type"` cannot recover them. This pass adds
// them back from that same positional information, so it needs no help from the
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
