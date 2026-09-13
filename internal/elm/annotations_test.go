package elm

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestFillEmptyAnnotations pins the rule the fill pass implements: every Element
// with no annotation gets the empty container CQF writes, reached through
// interfaces, pointers and slices — including nodes built without one, such as
// an inferred result-type specifier.
func TestFillEmptyAnnotations(t *testing.T) {
	inner := &NamedTypeSpecifier{Name: "{urn:hl7-org:elm-types:r1}Integer"}
	lit := &LiteralNode{ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"}
	SetResultType(lit, "", &ListTypeSpecifier{ElementType: inner})
	list := &ListNode{Element: []Expression{lit}}

	FillEmptyAnnotations(list)

	b, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	// List, Literal, the ListTypeSpecifier and the NamedTypeSpecifier inside it.
	if n := strings.Count(got, `"annotation":[]`); n != 4 {
		t.Errorf("want 4 empty annotation containers, got %d in %s", n, got)
	}
}

// TestFillEmptyAnnotationsKeepsExisting guards the other half of the rule: a
// node that already carries annotations keeps them, and a signature — raw JSON,
// not an Element — is never rewritten.
func TestFillEmptyAnnotationsKeepsExisting(t *testing.T) {
	real := json.RawMessage(`[{"type":"Annotation","t":[{"name":"x","value":"y"}]}]`)
	sig := json.RawMessage(`[{"type":"NamedTypeSpecifier","name":"{urn:hl7-org:elm-types:r1}Integer"}]`)
	lit := &LiteralNode{Annotation: real, ValueType: "{urn:hl7-org:elm-types:r1}Integer", Value: "1"}
	not := &UnaryExpressionNode{Operator: "Not", Signature: sig, Operand: lit}

	FillEmptyAnnotations(not)

	if string(lit.Annotation) != string(real) {
		t.Errorf("an existing annotation was replaced: %s", lit.Annotation)
	}
	if string(not.Signature) != string(sig) {
		t.Errorf("a signature was rewritten: %s", not.Signature)
	}
	if string(not.Annotation) != "[]" {
		t.Errorf("the outer node was not filled: %q", not.Annotation)
	}
}
