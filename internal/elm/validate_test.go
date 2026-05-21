package elm_test

import (
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/elm"
)

func TestValidate_XMLOK(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<library xmlns="urn:hl7-org:elm:r1" xmlns:t="urn:hl7-org:elm-types:r1">
  <identifier id="Test" version="1.0.0"/>
</library>`)
	if err := elm.Validate(data, "xml"); err != nil {
		t.Errorf("expected valid XML, got %v", err)
	}
}

func TestValidate_XMLNotWellFormed(t *testing.T) {
	data := []byte(`<library xmlns="urn:hl7-org:elm:r1"><identifier id="x" version="1"/>`)
	if err := elm.Validate(data, "xml"); err == nil {
		t.Error("expected error for unbalanced XML")
	}
}

func TestValidate_XMLWrongRoot(t *testing.T) {
	data := []byte(`<root xmlns="urn:hl7-org:elm:r1"/>`)
	err := elm.Validate(data, "xml")
	if err == nil || !strings.Contains(err.Error(), "library") {
		t.Errorf("expected 'library' root error, got %v", err)
	}
}

func TestValidate_JSONOK(t *testing.T) {
	data := []byte(`{"library":{"identifier":{"id":"Test","version":"1.0.0"}}}`)
	if err := elm.Validate(data, "json"); err != nil {
		t.Errorf("expected valid JSON, got %v", err)
	}
}

func TestValidate_JSONMissingLibrary(t *testing.T) {
	data := []byte(`{"notlibrary":{}}`)
	err := elm.Validate(data, "json")
	if err == nil || !strings.Contains(err.Error(), "library") {
		t.Errorf("expected 'library' missing error, got %v", err)
	}
}

func TestValidate_UnsupportedFormat(t *testing.T) {
	if err := elm.Validate([]byte("xx"), "yaml"); err == nil {
		t.Error("expected unsupported format error")
	}
}

func TestValidate_EmptyInputs(t *testing.T) {
	if err := elm.Validate(nil, "xml"); err == nil {
		t.Error("expected empty XML error")
	}
	if err := elm.Validate(nil, "json"); err == nil {
		t.Error("expected empty JSON error")
	}
}
