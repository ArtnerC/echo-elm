package elm

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// ValidationError aggregates validation diagnostics produced by Validate.
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "elm: no validation issues"
	}
	return "elm: " + strings.Join(e.Issues, "; ")
}

// Validate performs a pragmatic Go-native structural validation of serialized
// ELM output. It is not a full XSD schema validator — the official ELM XSDs
// (library.xsd, expression.xsd, clinicalexpression.xsd, types.xsd,
// modelinfo.xsd) require an XSD engine like libxml2. This function instead
// checks well-formedness, root element / shape, and required top-level
// attributes, which catches the overwhelming majority of producer bugs.
//
// format must be "xml" or "json" (case-insensitive). Use the official
// CQFramework tool or a libxml2-based pipeline for true XSD conformance.
func Validate(data []byte, format string) error {
	switch strings.ToLower(format) {
	case "xml":
		return validateXML(data)
	case "json":
		return validateJSON(data)
	default:
		return fmt.Errorf("elm: unsupported validation format %q (want xml|json)", format)
	}
}

func validateXML(data []byte) error {
	if len(data) == 0 {
		return errors.New("elm: empty XML document")
	}
	var issues []string
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	rootSeen := false
	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return &ValidationError{Issues: []string{"xml not well-formed: " + err.Error()}}
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !rootSeen {
			rootSeen = true
			if se.Name.Local != "library" {
				issues = append(issues, fmt.Sprintf("expected root element 'library', got %q", se.Name.Local))
			}
			if se.Name.Space != "urn:hl7-org:elm:r1" && se.Name.Space != "" {
				issues = append(issues, fmt.Sprintf("unexpected root namespace %q", se.Name.Space))
			}
			hasIdentifier := false
			for _, a := range se.Attr {
				if a.Name.Local == "xmlns" && a.Value == "urn:hl7-org:elm:r1" {
					hasIdentifier = true
				}
			}
			if !hasIdentifier && se.Name.Space == "" {
				issues = append(issues, "library element missing xmlns='urn:hl7-org:elm:r1'")
			}
		}
	}
	if !rootSeen {
		issues = append(issues, "no root element found")
	}
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

func validateJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("elm: empty JSON document")
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return &ValidationError{Issues: []string{"json not well-formed: " + err.Error()}}
	}
	var issues []string
	libRaw, ok := root["library"]
	if !ok {
		issues = append(issues, "missing top-level 'library' object")
	} else if _, ok := libRaw.(map[string]any); !ok {
		issues = append(issues, "'library' is not an object")
	}
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}
