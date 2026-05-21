package echoelm_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

func TestTranslate_Minimal(t *testing.T) {
	res, err := echoelm.Translate([]byte("library Minimal version '1.0.0'"), "Minimal.cql")
	if err != nil {
		t.Fatalf("translate error: %v", err)
	}
	if res.Library == nil {
		t.Fatal("expected Library, got nil")
	}
	if res.Library.Identifier.ID != "Minimal" {
		t.Errorf("got id=%q", res.Library.Identifier.ID)
	}
	if res.Library.Identifier.Version != "1.0.0" {
		t.Errorf("got version=%q", res.Library.Identifier.Version)
	}
}

func TestTranslate_JSONEnvelope(t *testing.T) {
	res, err := echoelm.Translate([]byte("library X version '1.0'"), "X.cql")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"library"`) {
		t.Errorf("expected envelope with 'library' key, got: %s", s)
	}
	if !strings.Contains(s, `"identifier"`) {
		t.Errorf("expected identifier key: %s", s)
	}
}

func TestTranslate_XMLOutput(t *testing.T) {
	res, err := echoelm.Translate([]byte("library X version '1.0'"), "X.cql")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	b, err := res.MarshalXML()
	if err != nil {
		t.Fatalf("xml: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "library") || !strings.Contains(s, "urn:hl7-org:elm:r1") {
		t.Errorf("missing ELM namespace or library element: %s", s)
	}
}

func TestTranslate_WithCQFOptions(t *testing.T) {
	res, err := echoelm.Translate([]byte("library M version '1.0'\ndefine X: 1"), "M.cql",
		echoelm.WithCQFOptions(),
	)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	b, _ := json.Marshal(res)
	s := string(b)
	// CQF mode emits empty annotation arrays on statements; not strictly required
	// at the library level, but the marshalled JSON should be valid.
	if !strings.Contains(s, `"X"`) {
		t.Errorf("expected define X in output: %s", s)
	}
}

func TestTranslate_WithLibrarySource(t *testing.T) {
	src := resolver.NewMapSource(map[string][]byte{
		"Helper": []byte("library Helper version '1.0'\ndefine H: 42"),
	})
	cql := `library Main version '1.0'
include Helper version '1.0' called H
define Use: H.H`
	res, err := echoelm.Translate([]byte(cql), "Main.cql",
		echoelm.WithLibrarySource(src),
	)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if res.Library == nil {
		t.Fatal("nil library")
	}
}

func TestTranslate_ParseError(t *testing.T) {
	// Severely broken CQL.
	res, err := echoelm.Translate([]byte("@@@invalid syntax@@@"), "Bad.cql")
	if err != nil {
		// Parser returned an error directly; acceptable.
		return
	}
	// Otherwise we expect diagnostics with error severity.
	hasErr := false
	for _, d := range res.Diagnostics {
		if d.Severity == "error" {
			hasErr = true
			break
		}
	}
	if !hasErr {
		t.Errorf("expected error diagnostics for invalid CQL")
	}
}
