package elm_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/elm"
)

// typeAt walks a dotted path through decoded ELM JSON and returns the "type" of
// the node found there. A path segment of "[N]" indexes an array.
func typeAt(t *testing.T, doc map[string]any, path string) string {
	t.Helper()
	var cur any = doc
	for _, seg := range strings.Split(path, ".") {
		if strings.HasPrefix(seg, "[") {
			idx := 0
			if _, err := fmt.Sscanf(seg, "[%d]", &idx); err != nil {
				t.Fatalf("bad path segment %q", seg)
			}
			arr, ok := cur.([]any)
			if !ok || idx >= len(arr) {
				t.Fatalf("%s: not an array with index %d", path, idx)
			}
			cur = arr[idx]
			continue
		}
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("%s: %q is not an object", path, seg)
		}
		cur, ok = m[seg]
		if !ok {
			t.Fatalf("%s: no field %q", path, seg)
		}
	}
	m, ok := cur.(map[string]any)
	if !ok {
		t.Fatalf("%s: not an object", path)
	}
	s, _ := m["type"].(string)
	return s
}

const sampleELM = `{"library":{
  "identifier": {"id":"T","version":"1.0"},
  "schemaIdentifier": {"id":"urn:hl7-org:elm","version":"r1"},
  "usings": {"def":[{"localIdentifier":"System","uri":"urn:hl7-org:elm-types:r1"}]},
  "includes": {"def":[{"localIdentifier":"C","path":"Common","version":"1.0"}]},
  "parameters": {"def":[{"name":"P"}]},
  "codeSystems": {"def":[{"name":"CS","id":"http://x"}]},
  "valueSets": {"def":[{"name":"VS","id":"urn:oid:1"}]},
  "codes": {"def":[{"name":"C1","id":"1","codeSystem":{"name":"CS"}}]},
  "concepts": {"def":[{"name":"K","code":[{"name":"C1"}]}]},
  "contexts": {"def":[{"name":"Patient"}]},
  "statements": {"def":[
    {"name":"Plain","expression":{"type":"Literal","value":"1"}},
    {"type":"FunctionDef","name":"F","operand":[{"name":"x"}],
     "expression":{"type":"Query",
       "source":[{"alias":"A","expression":{"type":"Retrieve"}}],
       "let":[{"identifier":"l","expression":{"type":"Literal"}}],
       "return":{"expression":{"type":"AliasRef","name":"A"}},
       "sort":{"by":[{"type":"ByColumn","path":"p"}]}}},
    {"name":"Cased","expression":{"type":"Case",
       "caseItem":[{"when":{"type":"Literal"},"then":{"type":"Literal"}}],
       "else":{"type":"Null"}}}
  ]}
}}`

func TestAddTypeDiscriminators(t *testing.T) {
	out, err := elm.AddTypeDiscriminators([]byte(sampleELM))
	if err != nil {
		t.Fatalf("add discriminators: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	for _, c := range []struct{ path, want string }{
		{"library", "Library"},
		{"library.identifier", "VersionedIdentifier"},
		{"library.schemaIdentifier", "VersionedIdentifier"},
		{"library.usings", "Library$Usings"},
		{"library.usings.def.[0]", "UsingDef"},
		{"library.includes", "Library$Includes"},
		{"library.includes.def.[0]", "IncludeDef"},
		{"library.parameters", "Library$Parameters"},
		{"library.parameters.def.[0]", "ParameterDef"},
		{"library.codeSystems", "Library$CodeSystems"},
		{"library.codeSystems.def.[0]", "CodeSystemDef"},
		{"library.valueSets", "Library$ValueSets"},
		{"library.valueSets.def.[0]", "ValueSetDef"},
		{"library.codes", "Library$Codes"},
		{"library.codes.def.[0]", "CodeDef"},
		{"library.codes.def.[0].codeSystem", "CodeSystemRef"},
		{"library.concepts", "Library$Concepts"},
		{"library.concepts.def.[0]", "ConceptDef"},
		{"library.concepts.def.[0].code.[0]", "CodeRef"},
		{"library.contexts", "Library$Contexts"},
		{"library.contexts.def.[0]", "ContextDef"},
		{"library.statements", "Library$Statements"},
		{"library.statements.def.[0]", "ExpressionDef"},
		// A statement that already declares FunctionDef keeps it.
		{"library.statements.def.[1]", "FunctionDef"},
		{"library.statements.def.[1].operand.[0]", "OperandDef"},
		// Query clauses.
		{"library.statements.def.[1].expression.source.[0]", "AliasedQuerySource"},
		{"library.statements.def.[1].expression.let.[0]", "LetClause"},
		{"library.statements.def.[1].expression.return", "ReturnClause"},
		{"library.statements.def.[1].expression.sort", "SortClause"},
		{"library.statements.def.[1].expression.sort.by.[0]", "ByColumn"},
		{"library.statements.def.[2].expression.caseItem.[0]", "CaseItem"},
	} {
		if got := typeAt(t, doc, c.path); got != c.want {
			t.Errorf("%s: type = %q, want %q", c.path, got, c.want)
		}
	}
}

// TestAddTypeDiscriminatorsLeavesEveryNodeTyped is the property that matters to a
// deserializer dispatching on "type": nothing with structure may be left bare.
func TestAddTypeDiscriminatorsLeavesEveryNodeTyped(t *testing.T) {
	out, err := elm.AddTypeDiscriminators([]byte(sampleELM))
	if err != nil {
		t.Fatalf("add discriminators: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	var bare []string
	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch node := v.(type) {
		case map[string]any:
			hasChildren := false
			for _, child := range node {
				switch child.(type) {
				case map[string]any, []any:
					hasChildren = true
				}
			}
			if _, typed := node["type"]; !typed && hasChildren && path != "" {
				bare = append(bare, path)
			}
			for k, child := range node {
				walk(child, path+"."+k)
			}
		case []any:
			for i, item := range node {
				walk(item, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	walk(doc["library"], "library")
	if len(bare) > 0 {
		t.Errorf("nodes left without a type discriminator: %v", bare)
	}
}

func TestAddTypeDiscriminatorsRejectsNonLibrary(t *testing.T) {
	if _, err := elm.AddTypeDiscriminators([]byte(`{"notALibrary":{}}`)); err == nil {
		t.Fatal("expected an error for input that is not an ELM envelope")
	}
	if _, err := elm.AddTypeDiscriminators([]byte(`not json`)); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}
