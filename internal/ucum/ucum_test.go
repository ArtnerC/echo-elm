package ucum_test

import (
	"testing"

	"github.com/artnerc/echo-elm/internal/ucum"
)

func TestValidate_Valid(t *testing.T) {
	cases := []string{
		"",
		"1",
		"mg",
		"mg/dL",
		"mmol/L",
		"kg",
		"m",
		"m2",
		"m/s",
		"m/s2",
		"kg.m/s2",
		"mL",
		"uL",
		"ng/mL",
		"mm[Hg]",
		"cm[H2O]",
		"%",
		"[IU]/L",
		"U/L",
		"Cel",
		"a",     // year (UCUM)
		"mo",    // month
		"d",     // day
		"wk",    // week
		"h",     // hour
		"min",   // minute
		"ms",    // millisecond (m + s)
		"years", // CQL temporal keyword
		"months",
		"days",
		"{rbc}",        // pure annotation → dimensionless
		"mg{total}/dL", // annotation on atom
	}
	for _, c := range cases {
		if err := ucum.Validate(c); err != nil {
			t.Errorf("expected valid: %q got %v", c, err)
		}
	}
}

func TestValidate_Invalid(t *testing.T) {
	cases := []string{
		"shab-shab-shab",
		"xyzzy",
		"notarealunit",
		"foo/bar",
	}
	for _, c := range cases {
		if err := ucum.Validate(c); err == nil {
			t.Errorf("expected invalid: %q", c)
		}
	}
}

func TestIsCQLTemporalKeyword(t *testing.T) {
	if !ucum.IsCQLTemporalKeyword("year") || !ucum.IsCQLTemporalKeyword("YEARS") {
		t.Error("year/YEARS should be temporal keywords")
	}
	if ucum.IsCQLTemporalKeyword("mg") {
		t.Error("mg should not be a temporal keyword")
	}
}
