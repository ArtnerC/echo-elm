package typesystem

import (
	"os"
	"regexp"
	"testing"
)

// TestFHIRHelpersOverloadedMatchesSource keeps FHIRHelpersOverloaded in step
// with the FHIRHelpers source vendored in the corpus.
func TestFHIRHelpersOverloadedMatchesSource(t *testing.T) {
	src, err := os.ReadFile("../../test/corpus/cqframework/layered/FHIRHelpers.cql")
	if err != nil {
		t.Fatalf("read FHIRHelpers: %v", err)
	}
	re := regexp.MustCompile(`define\s+(?:fluent\s+)?function\s+"?([A-Za-z_]+)"?\s*\(`)
	counts := map[string]int{}
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		counts[m[1]]++
	}
	for name, n := range counts {
		if (n > 1) != FHIRHelpersOverloaded[name] {
			t.Errorf("%s: %d definitions, FHIRHelpersOverloaded says %v", name, n, FHIRHelpersOverloaded[name])
		}
	}
	for name := range FHIRHelpersOverloaded {
		if counts[name] < 2 {
			t.Errorf("%s is listed as overloaded but has %d definitions", name, counts[name])
		}
	}
}
