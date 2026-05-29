// Package cqloptions loads CQL translator options from CQFramework
// conventions: a cql-options.json file (the typical IG content/cql/ pattern)
// or a FHIR Library resource that carries a cqf-cqlOptions extension.
//
// Reference: references/specs/using-cql-with-fhir/2.0.0/README.md and
// references/implementation/echo-elm.md.
package cqloptions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/artnerc/echo-elm/internal/translator"
)

// CQFCQLOptionsExtensionURL is the canonical URL of the cqf-cqlOptions extension.
const CQFCQLOptionsExtensionURL = "http://hl7.org/fhir/StructureDefinition/cqf-cqlOptions"

// File is the schema of a cql-options.json document.
// Fields not consumed by echo-elm are preserved for forward compatibility.
type File struct {
	Options                  []string `json:"options,omitempty"`
	Formats                  []string `json:"formats,omitempty"`
	ValidateUnits            *bool    `json:"validateUnits,omitempty"`
	VerifyOnly               *bool    `json:"verifyOnly,omitempty"`
	ErrorLevel               string   `json:"errorLevel,omitempty"`
	SignatureLevel           string   `json:"signatureLevel,omitempty"`
	CompatibilityLevel       string   `json:"compatibilityLevel,omitempty"`
	AnalyzeDataRequirements  *bool    `json:"analyzeDataRequirements,omitempty"`
	CollapseDataRequirements *bool    `json:"collapseDataRequirements,omitempty"`
}

// LoadFile reads and parses a cql-options.json file.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cqloptions: %w", err)
	}
	return Parse(data)
}

// Parse parses cql-options.json bytes.
func Parse(data []byte) (*File, error) {
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("cqloptions: parse: %w", err)
	}
	return &f, nil
}

// Apply applies the file's settings on top of a base translator.Options.
// Unset/empty fields are ignored.
func (f *File) Apply(base *translator.Options) translator.Options {
	out := *base
	for _, opt := range f.Options {
		applyOptionToken(&out, opt)
	}
	if f.ValidateUnits != nil {
		out.ValidateUnits = *f.ValidateUnits
	}
	if f.ErrorLevel != "" {
		out.ErrorLevel = f.ErrorLevel
	}
	if f.SignatureLevel != "" {
		out.SignatureLevel = f.SignatureLevel
	}
	if f.CompatibilityLevel != "" {
		out.CompatibilityLevel = f.CompatibilityLevel
	}
	return out
}

func applyOptionToken(o *translator.Options, token string) {
	switch strings.TrimSpace(token) {
	case "EnableAnnotations":
		o.EnableAnnotations = true
	case "EnableLocators":
		o.EnableLocators = true
	case "DisableListDemotion":
		o.DisableListDemotion = true
	case "DisableListPromotion":
		o.DisableListPromotion = true
	}
}

// Validate returns option tokens that are not recognized members of the
// CqlCompilerOptions.Options enum. Informational; callers may surface them.
func (f *File) Validate() []string {
	known := map[string]bool{
		"DisableDefaultModelInfoLoad": true,
		"EnableDateRangeOptimization": true,
		"EnableAnnotations":           true,
		"EnableLocators":              true,
		"EnableResultTypes":           true,
		"EnableDetailedErrors":        true,
		"DisableListTraversal":        true,
		"DisableListDemotion":         true,
		"DisableListPromotion":        true,
		"EnableIntervalDemotion":      true,
		"EnableIntervalPromotion":     true,
		"DisableMethodInvocation":     true,
		"RequireFromKeyword":          true,
	}
	var unknown []string
	for _, t := range f.Options {
		if !known[strings.TrimSpace(t)] {
			unknown = append(unknown, t)
		}
	}
	return unknown
}

// --- FHIR cqf-cqlOptions extension support ---

type fhirLibrary struct {
	ResourceType string          `json:"resourceType"`
	Extension    []fhirExtension `json:"extension,omitempty"`
}

type fhirExtension struct {
	URL          string          `json:"url"`
	Extension    []fhirExtension `json:"extension,omitempty"`
	ValueString  *string         `json:"valueString,omitempty"`
	ValueCode    *string         `json:"valueCode,omitempty"`
	ValueBoolean *bool           `json:"valueBoolean,omitempty"`
	ValueInteger *int            `json:"valueInteger,omitempty"`
}

// FromFHIRLibrary extracts cqf-cqlOptions from a FHIR Library JSON resource.
// Returns (nil, nil) if no cqf-cqlOptions extension is present.
func FromFHIRLibrary(data []byte) (*File, error) {
	var lib fhirLibrary
	if err := json.Unmarshal(data, &lib); err != nil {
		return nil, fmt.Errorf("cqloptions: parse FHIR Library: %w", err)
	}
	for _, ext := range lib.Extension {
		if ext.URL != CQFCQLOptionsExtensionURL {
			continue
		}
		return fromComplexExtension(ext), nil
	}
	return nil, nil
}

func fromComplexExtension(ext fhirExtension) *File {
	out := &File{}
	for _, sub := range ext.Extension {
		switch sub.URL {
		case "options", "option":
			if sub.ValueCode != nil {
				out.Options = append(out.Options, *sub.ValueCode)
			} else if sub.ValueString != nil {
				out.Options = append(out.Options, *sub.ValueString)
			}
		case "validateUnits":
			out.ValidateUnits = sub.ValueBoolean
		case "verifyOnly":
			out.VerifyOnly = sub.ValueBoolean
		case "errorLevel":
			out.ErrorLevel = pickString(sub)
		case "signatureLevel":
			out.SignatureLevel = pickString(sub)
		case "compatibilityLevel":
			out.CompatibilityLevel = pickString(sub)
		case "analyzeDataRequirements":
			out.AnalyzeDataRequirements = sub.ValueBoolean
		case "collapseDataRequirements":
			out.CollapseDataRequirements = sub.ValueBoolean
		}
	}
	return out
}

func pickString(e fhirExtension) string {
	if e.ValueCode != nil {
		return *e.ValueCode
	}
	if e.ValueString != nil {
		return *e.ValueString
	}
	return ""
}
