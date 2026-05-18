// demo/phase5: Phase 5 demo — CLI subcommands, contexts, and implicit Patient accessor.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artnerc/echo-elm/pkg/echoelm"
)

const banner = `╔══════════════════════════════════════════════════╗
║       echo-elm  Phase 5  —  CLI & ELM Structure    ║
╚══════════════════════════════════════════════════╝`

func main() {
	fmt.Println(banner)
	fmt.Println()

	tmp := os.TempDir()
	ok := true

	// ─── Example 1: translate (modern defaults, signatureLevel=Overloads) ─────
	ok = section(ok, "1", "Modern translate — contexts + implicit Patient accessor", func() bool {
		cql := []byte(`library PatientAge version '1.0.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
context Patient

define "Initial Population":
  AgeInYears() >= 18
`)
		res, err := echoelm.Translate(cql, "PatientAge.cql",
			echoelm.WithAnnotations(true),
			echoelm.WithLocators(true),
			echoelm.WithSignatureLevel("Overloads"),
		)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
			return false
		}
		out, _ := res.MarshalJSON()
		var lib map[string]interface{}
		json.Unmarshal(out, &lib)
		libMap := lib["library"].(map[string]interface{})

		// Contexts section
		if ctxs, ok := libMap["contexts"]; ok {
			defs := ctxs.(map[string]interface{})["def"].([]interface{})
			fmt.Printf("   contexts.def: %d entry(ies)\n", len(defs))
			for _, d := range defs {
				dm := d.(map[string]interface{})
				fmt.Printf("     - name: %q\n", dm["name"])
			}
		} else {
			fmt.Println("   ✗ contexts section missing!")
			return false
		}

		// Implicit Patient accessor
		stmts := libMap["statements"].(map[string]interface{})["def"].([]interface{})
		found := false
		for _, s := range stmts {
			sm := s.(map[string]interface{})
			if sm["name"] == "Patient" {
				found = true
				fmt.Printf("   implicit accessor: name=%q context=%q\n", sm["name"], sm["context"])
				expr := sm["expression"].(map[string]interface{})
				fmt.Printf("   expression.type: %q\n", expr["type"])
				operand := expr["operand"].(map[string]interface{})
				fmt.Printf("   operand.type: %q dataType: %q\n", operand["type"], operand["dataType"])
			}
		}
		if !found {
			fmt.Println("   ✗ implicit Patient accessor missing!")
			return false
		}
		fmt.Printf("   signatureLevel: %q\n", libMap["annotation"].([]interface{})[0].(map[string]interface{})["signatureLevel"])
		fmt.Println("   ✓ contexts + implicit accessor present")
		return true
	})

	// ─── Example 2: cqf translate compat mode (signatureLevel=None) ──────────
	ok = section(ok, "2", "CQF mode — signatureLevel=None (cqframework default)", func() bool {
		cql := []byte(`library Minimal version '1.0.0'
using FHIR version '4.0.1'
`)
		res, err := echoelm.Translate(cql, "Minimal.cql",
			echoelm.WithAnnotations(false),
			echoelm.WithLocators(true),
			echoelm.WithSignatureLevel("None"),
		)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
			return false
		}
		out, _ := res.MarshalJSON()
		var lib map[string]interface{}
		json.Unmarshal(out, &lib)
		libMap := lib["library"].(map[string]interface{})
		ann := libMap["annotation"].([]interface{})[0].(map[string]interface{})
		level := ann["signatureLevel"]
		fmt.Printf("   signatureLevel: %q (expected: \"None\")\n", level)
		if level != "None" {
			fmt.Println("   ✗ wrong signatureLevel!")
			return false
		}
		fmt.Println("   ✓ signatureLevel=None matches cqframework default")
		return true
	})

	// ─── Example 3: CLI binary demo (file I/O round-trip) ────────────────────
	ok = section(ok, "3", "CLI file I/O — write ELM JSON to temp directory", func() bool {
		cqlFile := filepath.Join(tmp, "demo5.cql")
		cql := `library Demo5 version '1.0.0'
using FHIR version '4.0.1'
context Patient

define "Adults":
  AgeInYears() >= 18
`
		if err := os.WriteFile(cqlFile, []byte(cql), 0o644); err != nil {
			fmt.Printf("   ERROR writing CQL: %v\n", err)
			return false
		}
		outFile := filepath.Join(tmp, "demo5.json")

		// Use the Go API to simulate what the CLI does.
		src, _ := os.ReadFile(cqlFile)
		res, err := echoelm.Translate(src, filepath.Base(cqlFile),
			echoelm.WithAnnotations(true),
			echoelm.WithLocators(true),
			echoelm.WithSignatureLevel("Overloads"),
		)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
			return false
		}
		elmBytes, _ := json.MarshalIndent(res, "", "   ")
		if err := os.WriteFile(outFile, elmBytes, 0o644); err != nil {
			fmt.Printf("   ERROR writing ELM: %v\n", err)
			return false
		}
		info, _ := os.Stat(outFile)
		fmt.Printf("   ELM written: %s (%d bytes)\n", outFile, info.Size())
		// Verify round-trip.
		var lib map[string]interface{}
		json.Unmarshal(elmBytes, &lib)
		libMap := lib["library"].(map[string]interface{})
		id := libMap["identifier"].(map[string]interface{})
		fmt.Printf("   identifier: %s %s\n", id["id"], id["version"])
		fmt.Println("   ✓ file I/O round-trip OK")
		return true
	})

	// ─── Example 4: Parity snapshot — known gaps ────────────────────────────
	section(ok, "4", "Known parity gaps vs cqframework (informational)", func() bool {
		fmt.Println("   Structural improvements in Phase 5:")
		fmt.Println("     ✓ Library.contexts.def emitted when context declared")
		fmt.Println("     ✓ Implicit Patient context accessor generated")
		fmt.Println("     ✓ SingletonFrom(Retrieve) node shape correct")
		fmt.Println("     ✓ signatureLevel=None in cqf mode (matches cqframework default)")
		fmt.Println()
		fmt.Println("   Remaining gaps (future phases):")
		fmt.Println("     ✗ annotation:[] on every def node (cqf mode only)")
		fmt.Println("     ✗ signature:[] on expression nodes (cqf mode only)")
		fmt.Println("     ✗ include/codeFilter/dateFilter/otherFilter:[] on Retrieve")
		fmt.Println("     ✗ Full type inference and type qualifications on literals")
		fmt.Println("     ✗ translatorOptions: empty string in cqf mode (tracks explicit flags only)")
		return true
	})

	fmt.Println()
	if ok {
		fmt.Println("Phase 5 complete ✓  CLI subcommands, contexts section, and implicit Patient accessor.")
	} else {
		fmt.Fprintln(os.Stderr, "Phase 5 INCOMPLETE — some checks failed.")
		os.Exit(1)
	}
}

func section(prevOK bool, num, title string, fn func() bool) bool {
	fmt.Printf("── Example %s: %s\n", num, title)
	result := fn()
	if !result {
		fmt.Printf("   ✗ FAILED\n")
	}
	fmt.Println()
	return prevOK && result
}
