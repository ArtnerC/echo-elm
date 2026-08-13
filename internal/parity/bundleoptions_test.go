package parity_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/pkg/bundle"
)

// elmDeclaring builds a minimal ELM document whose CqlToElmInfo annotation
// declares the given options, matching how CQF records them.
func elmDeclaring(options, signatureLevel string) []byte {
	doc := map[string]any{
		"library": map[string]any{
			"annotation": []any{
				map[string]any{
					"type":              "CqlToElmInfo",
					"translatorVersion": "5.0.0",
					"translatorOptions": options,
					"signatureLevel":    signatureLevel,
				},
			},
			"identifier": map[string]any{"id": "Lib", "version": "1.0"},
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return b
}

// bundleWith builds a Bundle whose libraries carry the given ELM documents.
func bundleWith(t *testing.T, elmByName map[string][]byte) *bundle.Bundle {
	t.Helper()
	entries := make([]any, 0, len(elmByName))
	for name, elmDoc := range elmByName {
		entries = append(entries, map[string]any{
			"resource": map[string]any{
				"resourceType": "Library",
				"name":         name,
				"version":      "1.0",
				"content": []any{
					map[string]any{
						"contentType": bundle.ContentTypeCQL,
						"data":        base64.StdEncoding.EncodeToString([]byte("library " + name + " version '1.0'\n")),
					},
					map[string]any{
						"contentType": bundle.ContentTypeELMJSON,
						"data":        base64.StdEncoding.EncodeToString(elmDoc),
					},
				},
			},
		})
	}
	raw, err := json.Marshal(map[string]any{
		"resourceType": "Bundle", "type": "collection", "entry": entries,
	})
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	b, err := bundle.Load(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	return b
}

// TestReadDeclaredOptions pins that the options a bundle records about its own
// compilation are read back, rather than the harness inventing a profile. This
// is the difference between comparing like with like and reporting every
// locator and resultType node as a translator gap.
func TestReadDeclaredOptions(t *testing.T) {
	const opts = "EnableLocators,EnableResultTypes,DisableListTraversal,DisableListDemotion,DisableListPromotion"
	declared, ok := parity.ReadDeclaredOptions(elmDeclaring(opts, "None"))
	if !ok {
		t.Fatal("expected the CqlToElmInfo annotation to be found")
	}

	profile, unmapped := declared.Profile("test")
	if len(unmapped) != 0 {
		t.Fatalf("unmapped options: %v", unmapped)
	}
	for _, key := range []string{
		"enableLocators", "enableResultTypes",
		"disableListTraversal", "disableListDemotion", "disableListPromotion",
	} {
		if v, ok := profile.TranslatorOptions[key]; !ok || v != true {
			t.Errorf("profile option %s = %v, want true", key, v)
		}
	}
	// Options that were not declared must stay absent rather than default to false.
	if _, present := profile.TranslatorOptions["enableAnnotations"]; present {
		t.Error("enableAnnotations was set from a declaration that did not mention it")
	}
	if got := profile.TranslatorOptions["signatureLevel"]; got != "None" {
		t.Errorf("signatureLevel = %v, want None", got)
	}
}

// TestReadDeclaredOptionsAbsent pins that ELM with no CqlToElmInfo annotation
// reports absence rather than an empty declaration, so callers can tell "built
// with no options" from "says nothing about how it was built".
func TestReadDeclaredOptionsAbsent(t *testing.T) {
	if _, ok := parity.ReadDeclaredOptions([]byte(`{"library":{"identifier":{"id":"X"}}}`)); ok {
		t.Error("reported a declaration where the document carries none")
	}
}

// TestMaterializeBundleDerivesProfile pins that the generated corpus profile
// comes from the bundle rather than from a caller-chosen default.
func TestMaterializeBundleDerivesProfile(t *testing.T) {
	b := bundleWith(t, map[string][]byte{
		"Lib": elmDeclaring("EnableLocators,EnableResultTypes", "Overloads"),
	})

	cfg, err := parity.MaterializeBundle(b, t.TempDir(), "")
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	corpus, err := parity.LoadCorpus(cfg.CorpusDir)
	if err != nil {
		t.Fatalf("load generated corpus: %v", err)
	}
	profile, ok := corpus.OptionProfiles[cfg.ProfileFilter]
	if !ok {
		t.Fatalf("generated corpus has no profile %q", cfg.ProfileFilter)
	}
	if profile.TranslatorOptions["enableLocators"] != true ||
		profile.TranslatorOptions["enableResultTypes"] != true {
		t.Errorf("derived profile did not carry the declared options: %v", profile.TranslatorOptions)
	}
	if profile.TranslatorOptions["signatureLevel"] != "Overloads" {
		t.Errorf("signatureLevel = %v, want Overloads", profile.TranslatorOptions["signatureLevel"])
	}
}

// TestMaterializeBundleRejectsDisagreeingLibraries pins that a bundle whose
// libraries were compiled differently is refused. No single profile can be
// correct for all of them, so comparing them together would silently misreport
// whichever libraries did not match the chosen one.
func TestMaterializeBundleRejectsDisagreeingLibraries(t *testing.T) {
	b := bundleWith(t, map[string][]byte{
		"A": elmDeclaring("EnableLocators", "None"),
		"B": elmDeclaring("EnableAnnotations", "None"),
	})

	_, err := parity.MaterializeBundle(b, t.TempDir(), "")
	if err == nil {
		t.Fatal("expected libraries with different compile options to be rejected")
	}
	if !strings.Contains(err.Error(), "disagree") {
		t.Errorf("error did not explain the disagreement: %v", err)
	}
}

// TestValidateProfileAgainstBundle pins the hard error for an explicitly
// requested profile that does not match the reference. Silently comparing a
// locator-bearing reference against a profile that emits none is exactly the
// failure mode this guards.
func TestValidateProfileAgainstBundle(t *testing.T) {
	b := bundleWith(t, map[string][]byte{
		"Lib": elmDeclaring("EnableLocators,EnableResultTypes", "None"),
	})

	mismatched := parity.OptionProfile{
		TranslatorOptions: map[string]interface{}{
			"enableLocators": false, "signatureLevel": "None",
		},
	}
	err := parity.ValidateProfileAgainstBundle(b, "default", mismatched)
	if err == nil {
		t.Fatal("expected a mismatched profile to be rejected")
	}
	for _, want := range []string{"EnableLocators", "default", "enableLocators"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message missing %q: %v", want, err)
		}
	}

	matching := parity.OptionProfile{
		TranslatorOptions: map[string]interface{}{
			"enableLocators": true, "enableResultTypes": true, "signatureLevel": "None",
		},
	}
	if err := parity.ValidateProfileAgainstBundle(b, "bundle", matching); err != nil {
		t.Errorf("matching profile was rejected: %v", err)
	}
}

// TestDeclaredOptionsUnmappable pins that an option echo-elm's profile format
// cannot express is surfaced instead of dropped. Ignoring one would put the
// comparison back in the state the derivation exists to prevent.
func TestDeclaredOptionsUnmappable(t *testing.T) {
	declared, ok := parity.ReadDeclaredOptions(elmDeclaring("EnableLocators,SomeFutureOption", "None"))
	if !ok {
		t.Fatal("expected a declaration")
	}
	_, unmapped := declared.Profile("test")
	if len(unmapped) != 1 || unmapped[0] != "SomeFutureOption" {
		t.Errorf("unmapped = %v, want [SomeFutureOption]", unmapped)
	}
}

// TestBundleShapeReductionIsPathScoped pins the two halves of the bundle-shape
// contract: the reduction has to happen on the bundle path, and must NOT happen
// on the CLI path.
//
// The CQF CLI emits "signature": [] and signatureLevel; echo-elm matches it
// (issues/04 G1/G2). The JAXB writer used inside Library.content[] omits both,
// because an empty collection maps to no XML elements. Reducing on the CLI path
// would silently stop checking something echo-elm is required to get right.
func TestBundleShapeReductionIsPathScoped(t *testing.T) {
	withEmpties := `{"library":{
		"annotation":[{"type":"CqlToElmInfo","translatorOptions":"","signatureLevel":"None"}],
		"identifier":{"id":"Lib"},
		"statements":{"def":[{"name":"X","signature":[],"let":[],
			"expression":{"type":"Retrieve","codeFilter":[],"dateFilter":[],
				"include":[],"otherFilter":[]}}]}}}`
	lean := `{"library":{
		"annotation":[{"type":"CqlToElmInfo","translatorOptions":""}],
		"identifier":{"id":"Lib"},
		"statements":{"def":[{"name":"X",
			"expression":{"type":"Retrieve"}}]}}}`

	// CLI path: the two must stay distinguishable.
	if parity.NormalizeForGolden(withEmpties) == parity.NormalizeForGolden(lean) {
		t.Error("CLI-path normalization erased the empty collections; " +
			"echo-elm emitting them is a real requirement and must stay checked")
	}

	// Bundle path: the two describe the same library and must compare equal.
	if parity.NormalizeBundleShape(withEmpties) != parity.NormalizeBundleShape(lean) {
		t.Errorf("bundle-path normalization did not reconcile the two writer shapes:\n%s",
			parity.SimpleDiff(parity.NormalizeBundleShape(withEmpties),
				parity.NormalizeBundleShape(lean)))
	}
}

// TestBundleShapeKeepsRealDifferences pins that the reduction only removes
// empties. A collection with content still has to compare.
func TestBundleShapeKeepsRealDifferences(t *testing.T) {
	withContent := `{"library":{"identifier":{"id":"L"},"statements":{"def":[
		{"name":"X","expression":{"type":"Retrieve","codeFilter":[{"path":"code"}]}}]}}}`
	without := `{"library":{"identifier":{"id":"L"},"statements":{"def":[
		{"name":"X","expression":{"type":"Retrieve"}}]}}}`

	if parity.NormalizeBundleShape(withContent) == parity.NormalizeBundleShape(without) {
		t.Error("a non-empty codeFilter was reduced away")
	}
}
