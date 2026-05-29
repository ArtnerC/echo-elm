// Command demo-phase2 demonstrates the echo-elm Phase 2/3 milestone:
// CQL→ELM translation with JSON and XML output.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/artnerc/echo-elm/pkg/echoelm"
)

const cqlSource = `library BPMeasure version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

codesystem "LOINC": 'http://loinc.org'

valueset "Blood Pressure VS": 'http://example.org/fhir/ValueSet/bp'

parameter "Measurement Period" Interval<DateTime>

context Patient

define "Initial Population":
  true

define function "AgeInYears"(birthDate Date):
  null
`

func main() {
	result, err := echoelm.Translate([]byte(cqlSource), "BPMeasure.cql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Translation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Diagnostics ===")
	if len(result.Diagnostics) == 0 {
		fmt.Println("(none)")
	}
	for _, d := range result.Diagnostics {
		fmt.Println(d)
	}

	fmt.Println("\n=== ELM JSON ===")
	jsonBytes, err := result.MarshalJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
		os.Exit(1)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, jsonBytes, "", "  "); err != nil {
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println(pretty.String())
	}

	fmt.Println("\n=== ELM XML ===")
	xmlBytes, err := result.XMLBytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "XML marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(xmlBytes))
}
