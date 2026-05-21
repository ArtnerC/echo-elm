package cqloptions_test

import (
	"testing"

	"github.com/artnerc/echo-elm/internal/cqloptions"
	"github.com/artnerc/echo-elm/internal/translator"
)

func TestParse_CanonicalFile(t *testing.T) {
	data := []byte(`{
		"options": ["EnableAnnotations", "EnableLocators", "DisableListDemotion", "DisableListPromotion"],
		"formats": ["XML", "JSON"],
		"validateUnits": true,
		"verifyOnly": false,
		"errorLevel": "Info",
		"signatureLevel": "Overloads",
		"compatibilityLevel": "1.5"
	}`)
	f, err := cqloptions.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Options) != 4 {
		t.Errorf("expected 4 options, got %d", len(f.Options))
	}
	if f.SignatureLevel != "Overloads" {
		t.Errorf("signatureLevel got %q", f.SignatureLevel)
	}
	if f.ValidateUnits == nil || !*f.ValidateUnits {
		t.Error("validateUnits not parsed")
	}
}

func TestApply(t *testing.T) {
	base := translator.DefaultOptions()
	base.EnableAnnotations = false
	base.SignatureLevel = "None"

	f := &cqloptions.File{
		Options:        []string{"EnableAnnotations", "DisableListPromotion"},
		SignatureLevel: "Overloads",
	}
	out := f.Apply(base)
	if !out.EnableAnnotations {
		t.Error("EnableAnnotations not applied")
	}
	if !out.DisableListPromotion {
		t.Error("DisableListPromotion not applied")
	}
	if out.SignatureLevel != "Overloads" {
		t.Errorf("SignatureLevel: got %q", out.SignatureLevel)
	}
}

func TestValidate_UnknownOptions(t *testing.T) {
	f := &cqloptions.File{Options: []string{"EnableLocators", "BogusFlag", "AnotherBogus"}}
	unknown := f.Validate()
	if len(unknown) != 2 {
		t.Errorf("expected 2 unknown, got %v", unknown)
	}
}

func TestFromFHIRLibrary(t *testing.T) {
	libJSON := []byte(`{
		"resourceType": "Library",
		"extension": [{
			"url": "http://hl7.org/fhir/StructureDefinition/cqf-cqlOptions",
			"extension": [
				{"url": "options", "valueCode": "EnableAnnotations"},
				{"url": "options", "valueCode": "EnableLocators"},
				{"url": "signatureLevel", "valueCode": "Overloads"},
				{"url": "compatibilityLevel", "valueString": "1.5"},
				{"url": "validateUnits", "valueBoolean": true}
			]
		}]
	}`)
	f, err := cqloptions.FromFHIRLibrary(libJSON)
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("expected non-nil file")
	}
	if len(f.Options) != 2 {
		t.Errorf("options=%v", f.Options)
	}
	if f.SignatureLevel != "Overloads" {
		t.Errorf("signatureLevel=%q", f.SignatureLevel)
	}
	if f.CompatibilityLevel != "1.5" {
		t.Errorf("compatibilityLevel=%q", f.CompatibilityLevel)
	}
	if f.ValidateUnits == nil || !*f.ValidateUnits {
		t.Error("validateUnits not parsed")
	}
}

func TestFromFHIRLibrary_NoExtension(t *testing.T) {
	libJSON := []byte(`{"resourceType": "Library"}`)
	f, err := cqloptions.FromFHIRLibrary(libJSON)
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Errorf("expected nil, got %+v", f)
	}
}
