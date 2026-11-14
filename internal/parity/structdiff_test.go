package parity

import (
	"strings"
	"testing"
)

// TestStructuralDiffAlignsDefinitionsByName pins the property the harness was
// missing: one absent definition must report as one finding, not as a cascade of
// mismatches against whichever definition happened to shift into its index.
//
// This is what produced the phantom "If became Or" findings that issues/05
// Part 1 had to retract.
func TestStructuralDiffAlignsDefinitionsByName(t *testing.T) {
	want := `{"library":{"statements":{"def":[
		{"name":"A","expression":{"type":"If"}},
		{"name":"B","expression":{"type":"Or"}},
		{"name":"C","expression":{"type":"And"}}]}}}`
	// echo-elm is missing A; index alignment would pair B against A and C
	// against B, reporting two type differences that do not exist.
	got := `{"library":{"statements":{"def":[
		{"name":"B","expression":{"type":"Or"}},
		{"name":"C","expression":{"type":"And"}}]}}}`

	diff := StructuralDiff(want, got)
	if !strings.Contains(diff, `definition "A" only in reference`) {
		t.Errorf("missing definition not reported as such:\n%s", diff)
	}
	for _, phantom := range []string{"If", "And"} {
		if strings.Contains(diff, phantom) {
			t.Errorf("reported a phantom difference mentioning %q — definitions were aligned by index:\n%s", phantom, diff)
		}
	}
	if n := strings.Count(strings.TrimSpace(diff), "\n") + 1; n != 1 {
		t.Errorf("one missing definition produced %d findings, want 1:\n%s", n, diff)
	}
}

// TestStructuralDiffReportsOrderingOnce pins that reordering is one finding.
func TestStructuralDiffReportsOrderingOnce(t *testing.T) {
	want := `{"library":{"statements":{"def":[{"name":"A","x":1},{"name":"B","x":2}]}}}`
	got := `{"library":{"statements":{"def":[{"name":"B","x":2},{"name":"A","x":1}]}}}`

	diff := StructuralDiff(want, got)
	if !strings.Contains(diff, "different order") {
		t.Errorf("reordering not reported:\n%s", diff)
	}
	if n := strings.Count(strings.TrimSpace(diff), "\n") + 1; n != 1 {
		t.Errorf("reordering produced %d findings, want 1:\n%s", n, diff)
	}
}

// TestStructuralDiffDetectsWrapper pins that an inserted node is reported as an
// insertion rather than as a difference at every descendant.
func TestStructuralDiffDetectsWrapper(t *testing.T) {
	want := `{"library":{"statements":{"def":[{"name":"A","expression":
		{"type":"Null"}}]}}}`
	got := `{"library":{"statements":{"def":[{"name":"A","expression":
		{"type":"As","asType":"String","operand":{"type":"Null"}}}]}}}`

	diff := StructuralDiff(want, got)
	if !strings.Contains(diff, "wraps this in As") {
		t.Errorf("wrapper not detected:\n%s", diff)
	}
	if n := strings.Count(strings.TrimSpace(diff), "\n") + 1; n != 1 {
		t.Errorf("wrapper produced %d findings, want 1:\n%s", n, diff)
	}
}

// TestStructuralDiffFindsRealDifferences guards against the alignment being so
// forgiving that it stops reporting anything.
func TestStructuralDiffFindsRealDifferences(t *testing.T) {
	want := `{"library":{"statements":{"def":[{"name":"A","expression":{"type":"If"}}]}}}`
	got := `{"library":{"statements":{"def":[{"name":"A","expression":{"type":"Or"}}]}}}`

	diff := StructuralDiff(want, got)
	if !strings.Contains(diff, "If") || !strings.Contains(diff, "Or") {
		t.Errorf("a genuine type difference under a matched name was not reported:\n%s", diff)
	}
}

// TestStructuralDiffEmptyWhenIdentical keeps a match from reporting anything.
func TestStructuralDiffEmptyWhenIdentical(t *testing.T) {
	doc := `{"library":{"statements":{"def":[{"name":"A","expression":{"type":"If"}}]}}}`
	if diff := StructuralDiff(doc, doc); diff != "" {
		t.Errorf("identical documents reported a difference:\n%s", diff)
	}
}

// overloadDef builds a FunctionDef named name with one operand of type opType.
func overloadDef(name, opType, body string) string {
	return `{"name":"` + name + `","type":"FunctionDef","operand":[{"name":"x","operandTypeSpecifier":` +
		`{"type":"NamedTypeSpecifier","name":"` + opType + `"}}],"expression":{"type":"` + body + `"}}`
}

// TestStructuralDiffPairsOverloadsBySignature pins issues/08 Part 7: overloads
// pair by signature, not by position, so emitting them in a different order is
// an ordering finding only — not a mismatch at every overload.
func TestStructuralDiffPairsOverloadsBySignature(t *testing.T) {
	a, b := overloadDef("F", "{x}A", "If"), overloadDef("F", "{x}B", "Or")
	want := `{"library":{"statements":{"def":[` + a + `,` + b + `]}}}`
	got := `{"library":{"statements":{"def":[` + b + `,` + a + `]}}}`

	diff := StructuralDiff(want, got)
	if strings.Contains(diff, "If") || strings.Contains(diff, "only in") {
		t.Errorf("reordered overloads were paired by position:\n%s", diff)
	}
	if !strings.Contains(diff, "different order") {
		t.Errorf("reordering not reported:\n%s", diff)
	}
}

// TestStructuralDiffPairsChangedOverloadSignature pins that an overload whose
// operand type differs between the sides — the issues/08 W3 shape, Claim.Item
// emitted as Item — reports that one field, not two unrelated definitions.
func TestStructuralDiffPairsChangedOverloadSignature(t *testing.T) {
	want := `{"library":{"statements":{"def":[` + overloadDef("F", "{x}Claim.Item", "If") + `,` + overloadDef("F", "{x}Other", "If") + `]}}}`
	got := `{"library":{"statements":{"def":[` + overloadDef("F", "{x}Item", "If") + `,` + overloadDef("F", "{x}Other", "If") + `]}}}`

	diff := StructuralDiff(want, got)
	if strings.Contains(diff, "only in") {
		t.Errorf("a changed signature was reported as unpaired definitions:\n%s", diff)
	}
	if !strings.Contains(diff, "Claim.Item") {
		t.Errorf("the operand type difference was not reported:\n%s", diff)
	}
}
