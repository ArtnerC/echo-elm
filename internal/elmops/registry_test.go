package elmops_test

import (
	"testing"

	"github.com/artnerc/echo-elm/internal/elmops"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name string
		kind elmops.Kind
	}{
		{"Add", elmops.KindOperator},
		{"Abs", elmops.KindUnary},
		{"Count", elmops.KindAggregate},
		{"Substring", elmops.KindNamed},
		{"DurationBetween", elmops.KindPrecision},
		{"DoesNotExist", elmops.KindUnknown},
	}
	for _, tt := range tests {
		op, ok := elmops.Lookup(tt.name)
		if tt.kind == elmops.KindUnknown {
			if ok {
				t.Errorf("%s should not exist", tt.name)
			}
			continue
		}
		if !ok {
			t.Errorf("%s missing from registry", tt.name)
			continue
		}
		if op.Kind != tt.kind {
			t.Errorf("%s kind: got %d want %d", tt.name, op.Kind, tt.kind)
		}
	}
}

func TestPredicates(t *testing.T) {
	if !elmops.IsUnary("Abs") {
		t.Error("Abs should be unary")
	}
	if !elmops.IsAggregate("Count") {
		t.Error("Count should be aggregate")
	}
	if !elmops.IsBinaryOperand("Add") {
		t.Error("Add should be operator")
	}
	if elmops.IsUnary("Add") {
		t.Error("Add should not be unary")
	}
}

func TestNamedFields(t *testing.T) {
	got := elmops.NamedFields("Substring")
	want := []string{"stringToSub", "startIndex", "length"}
	if len(got) != len(want) {
		t.Fatalf("Substring fields: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("idx %d: got %q want %q", i, got[i], want[i])
		}
	}
	if elmops.NamedFields("Add") != nil {
		t.Error("Add should have no NamedFields")
	}
}

func TestAllReturnsCopy(t *testing.T) {
	a := elmops.All()
	a["__test_injection__"] = elmops.Op{}
	b := elmops.All()
	if _, exists := b["__test_injection__"]; exists {
		t.Error("All() must return a copy, not the underlying map")
	}
}

func TestSystemConvertToOps(t *testing.T) {
	if elmops.SystemConvertToOps["Integer"] != "ToInteger" {
		t.Error("Integer→ToInteger mapping missing")
	}
}
