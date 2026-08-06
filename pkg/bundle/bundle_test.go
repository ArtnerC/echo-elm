package bundle_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/pkg/bundle"
)

// makeBundle builds a minimal FHIR Bundle carrying the given libraries.
// Each library is (name, version, cql, elm); an empty elm omits that attachment.
func makeBundle(t *testing.T, libs ...[4]string) string {
	t.Helper()
	entries := make([]any, 0, len(libs))
	for _, l := range libs {
		content := []any{map[string]any{
			"contentType": bundle.ContentTypeCQL,
			"data":        base64.StdEncoding.EncodeToString([]byte(l[2])),
		}}
		if l[3] != "" {
			content = append(content, map[string]any{
				"contentType": bundle.ContentTypeELMJSON,
				"data":        base64.StdEncoding.EncodeToString([]byte(l[3])),
			})
		}
		entries = append(entries, map[string]any{
			"resource": map[string]any{
				"resourceType": "Library",
				"name":         l[0],
				"version":      l[1],
				"content":      content,
			},
		})
	}
	doc := map[string]any{
		"resourceType": "Bundle",
		"type":         "collection",
		"entry":        entries,
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}
	return string(b)
}

func load(t *testing.T, doc string) *bundle.Bundle {
	t.Helper()
	b, err := bundle.Load(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return b
}

func TestLoadDecodesAttachments(t *testing.T) {
	doc := makeBundle(t,
		[4]string{"Common", "1.0", "library Common version '1.0'", `{"library":{}}`},
		[4]string{"Measure", "2.0", "library Measure version '2.0'", ""},
	)
	b := load(t, doc)

	libs := b.Libraries()
	if len(libs) != 2 {
		t.Fatalf("got %d libraries, want 2", len(libs))
	}
	// Libraries are ordered by name for stable output.
	if libs[0].Name != "Common" || libs[1].Name != "Measure" {
		t.Errorf("libraries not ordered by name: %q, %q", libs[0].Name, libs[1].Name)
	}
	if string(libs[0].CQLSource) != "library Common version '1.0'" {
		t.Errorf("Common CQL = %q", libs[0].CQLSource)
	}
	if string(libs[0].ReferenceELM) != `{"library":{}}` {
		t.Errorf("Common reference ELM = %q", libs[0].ReferenceELM)
	}
	if libs[1].ReferenceELM != nil {
		t.Errorf("Measure has no ELM attachment, got %q", libs[1].ReferenceELM)
	}
}

func TestLoadRejectsNonBundle(t *testing.T) {
	_, err := bundle.Load(strings.NewReader(`{"resourceType":"Library"}`))
	if err == nil {
		t.Fatal("expected an error for a non-Bundle resource")
	}
	if !strings.Contains(err.Error(), "want Bundle") {
		t.Errorf("error should name the problem, got: %v", err)
	}
}

func TestLoadIgnoresNonLibraryEntries(t *testing.T) {
	doc := `{"resourceType":"Bundle","entry":[
		{"resource":{"resourceType":"Measure","name":"M"}},
		{"resource":{"resourceType":"Library","name":"L","version":"1.0","content":[
			{"contentType":"text/cql","data":"` +
		base64.StdEncoding.EncodeToString([]byte("library L")) + `"}]}}
	]}`
	b := load(t, doc)
	libs := b.Libraries()
	if len(libs) != 1 || libs[0].Name != "L" {
		t.Fatalf("got %d libraries, want just the Library resource", len(libs))
	}
}

// TestLibrarySourceResolvesWithinBundle covers the reason this package exists:
// a measure's includes resolve against its sibling libraries in the bundle,
// without anything being written to disk.
func TestLibrarySourceResolvesWithinBundle(t *testing.T) {
	doc := makeBundle(t,
		[4]string{"Common", "1.0.0", "library Common version '1.0.0'", ""},
		[4]string{"Measure", "1.0.0", "library Measure version '1.0.0'", ""},
	)
	src := load(t, doc).LibrarySource()

	for _, c := range []struct {
		name, version string
		wantFound     bool
	}{
		{"Common", "1.0.0", true},
		{"Common", "", true},      // any version
		{"Common", "9.9.9", true}, // sole library of that name
		{"Missing", "", false},
	} {
		got, found, err := src.GetLibrarySource(c.name, c.version)
		if err != nil {
			t.Fatalf("%s|%s: %v", c.name, c.version, err)
		}
		if found != c.wantFound {
			t.Errorf("%s|%s: found = %v, want %v", c.name, c.version, found, c.wantFound)
			continue
		}
		if found && !strings.Contains(string(got), "library "+c.name) {
			t.Errorf("%s|%s: resolved to the wrong source: %q", c.name, c.version, got)
		}
	}
}

func TestSetELMReplacesExistingAttachment(t *testing.T) {
	doc := makeBundle(t, [4]string{"Common", "1.0", "library Common", `{"library":{"old":true}}`})
	b := load(t, doc)

	if err := b.SetELM("Common", "1.0", []byte(`{"library":{"new":true}}`)); err != nil {
		t.Fatalf("set ELM: %v", err)
	}
	out, err := b.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	reloaded := load(t, string(out))
	libs := reloaded.Libraries()
	if len(libs) != 1 {
		t.Fatalf("got %d libraries after round-trip", len(libs))
	}
	if string(libs[0].ReferenceELM) != `{"library":{"new":true}}` {
		t.Errorf("ELM not replaced: %q", libs[0].ReferenceELM)
	}
	// Exactly one ELM attachment: replaced, not appended.
	if n := strings.Count(string(out), bundle.ContentTypeELMJSON); n != 1 {
		t.Errorf("got %d elm+json attachments, want 1", n)
	}
}

func TestSetELMAddsMissingAttachment(t *testing.T) {
	doc := makeBundle(t, [4]string{"Measure", "1.0", "library Measure", ""})
	b := load(t, doc)

	if err := b.SetELM("Measure", "1.0", []byte(`{"library":{}}`)); err != nil {
		t.Fatalf("set ELM: %v", err)
	}
	out, err := b.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(load(t, string(out)).Libraries()[0].ReferenceELM) != `{"library":{}}` {
		t.Error("ELM attachment was not added")
	}
}

func TestSetELMUnknownLibrary(t *testing.T) {
	b := load(t, makeBundle(t, [4]string{"Common", "1.0", "library Common", ""}))
	if err := b.SetELM("Nope", "1.0", []byte("{}")); err == nil {
		t.Fatal("expected an error for an unknown library")
	}
}

// TestMarshalPreservesUnmodelledFields matters for round-tripping a real
// measure bundle: everything this package does not model has to survive.
func TestMarshalPreservesUnmodelledFields(t *testing.T) {
	doc := `{"resourceType":"Bundle","id":"keep-me","type":"collection",
		"meta":{"profile":["http://example.org/StructureDefinition/measure-bundle"]},
		"entry":[{"fullUrl":"urn:uuid:1","resource":{"resourceType":"Library",
			"name":"L","version":"1.0","status":"active","content":[
			{"contentType":"text/cql","data":"` +
		base64.StdEncoding.EncodeToString([]byte("library L")) + `"}]}}]}`

	b := load(t, doc)
	if err := b.SetELM("L", "1.0", []byte(`{"library":{}}`)); err != nil {
		t.Fatalf("set ELM: %v", err)
	}
	out, err := b.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for _, want := range []string{`"id": "keep-me"`, `"fullUrl": "urn:uuid:1"`,
		`"status": "active"`, "measure-bundle"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("marshalled bundle dropped %s", want)
		}
	}
}

func TestLibraryFileName(t *testing.T) {
	for _, c := range []struct{ name, version, want string }{
		{"Common", "1.0.0", "Common-1.0.0.cql"},
		{"Common", "", "Common.cql"},
	} {
		lib := bundle.Library{Name: c.name, Version: c.version}
		got := lib.FileName()
		if got != c.want {
			t.Errorf("FileName(%q, %q) = %q, want %q", c.name, c.version, got, c.want)
		}
	}
}

// TestLibraryFileNameCannotEscapeDirectory guards `bundle extract`: library
// names come from a downloaded bundle, so they must not be able to write
// outside the output directory.
func TestLibraryFileNameCannotEscapeDirectory(t *testing.T) {
	for _, name := range []string{"../evil", "a/b", `..\evil`, `x\..\y`, "../../etc/passwd"} {
		lib := bundle.Library{Name: name, Version: "1.0"}
		got := lib.FileName()
		if strings.ContainsAny(got, `/\`) || strings.Contains(got, "..") {
			t.Errorf("FileName(%q) = %q, which can escape the output directory", name, got)
		}
	}
}
