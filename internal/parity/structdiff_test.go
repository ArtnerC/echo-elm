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
