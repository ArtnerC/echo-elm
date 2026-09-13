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

// referenceELM builds a minimal ELM document whose CqlToElmInfo annotation
// carries the given translatorVersion, or none when version is "".
func referenceELM(t *testing.T, version string) []byte {
	t.Helper()
	info := map[string]any{"type": "CqlToElmInfo", "translatorOptions": ""}
	if version != "" {
		info["translatorVersion"] = version
	}
	doc := map[string]any{"library": map[string]any{
		"annotation": []any{info},
		"identifier": map[string]any{"id": "L", "version": "1.0"},
	}}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func bundleOf(t *testing.T, elmByName map[string][]byte) *bundle.Bundle {
	t.Helper()
	entries := make([]any, 0, len(elmByName))
	for name, elm := range elmByName {
		entries = append(entries, map[string]any{"resource": map[string]any{
			"resourceType": "Library",
			"name":         name,
			"version":      "1.0",
			"content": []any{
				map[string]any{"contentType": bundle.ContentTypeCQL,
					"data": base64.StdEncoding.EncodeToString([]byte("library " + name + " version '1.0'"))},
				map[string]any{"contentType": bundle.ContentTypeELMJSON,
					"data": base64.StdEncoding.EncodeToString(elm)},
			},
		}})
	}
	raw, err := json.Marshal(map[string]any{"resourceType": "Bundle", "type": "collection", "entry": entries})
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	b, err := bundle.Load(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	return b
}

// TestReferenceWarningsMissingVersion pins the signal issues/06 identified: a
// reference compiled by the pinned CLI always records translatorVersion, so its
// absence means the reference was produced by something else.
func TestReferenceWarningsMissingVersion(t *testing.T) {
	b := bundleOf(t, map[string][]byte{"Stale": referenceELM(t, "")})
	warnings := parity.ReferenceWarnings(b, "5.0.0")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "declare no translatorVersion") ||
		!strings.Contains(warnings[0], "Stale") {
		t.Fatalf("want one missing-version warning naming the library, got %q", warnings)
	}
}

func TestReferenceWarningsOtherVersion(t *testing.T) {
	b := bundleOf(t, map[string][]byte{"Older": referenceELM(t, "3.29.0")})
	warnings := parity.ReferenceWarnings(b, "5.0.0")
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Older (3.29.0)") {
		t.Fatalf("want one other-version warning naming library and version, got %q", warnings)
	}
}

// TestReferenceWarningsPinnedIsQuiet keeps a correct reference from warning; a
// warning that always fires would be ignored.
func TestReferenceWarningsPinnedIsQuiet(t *testing.T) {
	b := bundleOf(t, map[string][]byte{"Fresh": referenceELM(t, "5.0.0")})
	if warnings := parity.ReferenceWarnings(b, "5.0.0"); len(warnings) != 0 {
		t.Fatalf("a reference from the pinned translator warned: %q", warnings)
	}
}
