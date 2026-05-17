// Command demo-phase1 is the Phase 1 milestone demo.
// It demonstrates the echo-elm CQL lexer/parser:
//   - Parses multiple CQL library snippets
//   - Prints the extracted AST structure (library name, version, usings, etc.)
//   - Reports diagnostics for invalid CQL
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/echo-health/echo-elm/internal/parser"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║        echo-elm  Phase 1  —  CQL Parser Demo       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	examples := []struct {
		name string
		src  string
	}{
		{
			name: "Minimal library",
			src:  `library Minimal version '1.0.0'`,
		},
		{
			name: "Blood Pressure measure (header only)",
			src: `library BPMeasure version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers
include MATGlobalCommonFunctions version '6.1.000' called Global

codesystem "LOINC": 'http://loinc.org'

valueset "Blood Pressure VS": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1033'

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

context Patient

define "Initial Population":
  exists([Observation: "Blood Pressure VS"])

define function AgeInYears(birthDate DateTime):
  AgeInYearsAt(date from start of "Measurement Period")
`,
		},
		{
			name: "Syntax error recovery",
			src:  `library Bad version 'no-closing-bracket

define "Broken": @@@@invalid`,
		},
	}

	allPassed := true
	for i, ex := range examples {
		fmt.Printf("── Example %d: %s\n", i+1, ex.name)
		fmt.Printf("   CQL: %s\n", preview(ex.src, 80))

		result, err := parser.ParseString(ex.src, ex.name+".cql")
		if err != nil {
			fmt.Printf("   GO ERROR: %v\n", err)
			allPassed = false
			fmt.Println()
			continue
		}

		if result.Library != nil {
			lib := result.Library
			libName := "<anonymous>"
			libVer := ""
			if lib.Name != nil {
				libName = lib.Name.Name
				libVer = lib.Name.Version
			}
			fmt.Printf("   Library:    %s", libName)
			if libVer != "" {
				fmt.Printf(" version '%s'", libVer)
			}
			fmt.Println()

			if len(lib.Usings) > 0 {
				parts := make([]string, len(lib.Usings))
				for j, u := range lib.Usings {
					parts[j] = fmt.Sprintf("%s %s", u.ModelName, u.Version)
				}
				fmt.Printf("   Using:      %s\n", strings.Join(parts, ", "))
			}
			if len(lib.Includes) > 0 {
				parts := make([]string, len(lib.Includes))
				for j, inc := range lib.Includes {
					parts[j] = fmt.Sprintf("%s called %s", inc.Path, inc.LocalName)
				}
				fmt.Printf("   Includes:   %s\n", strings.Join(parts, "; "))
			}
			if len(lib.Codesystems) > 0 {
				parts := make([]string, len(lib.Codesystems))
				for j, cs := range lib.Codesystems {
					parts[j] = cs.Name
				}
				fmt.Printf("   Codesys:    %s\n", strings.Join(parts, ", "))
			}
			if len(lib.Valuesets) > 0 {
				parts := make([]string, len(lib.Valuesets))
				for j, vs := range lib.Valuesets {
					parts[j] = vs.Name
				}
				fmt.Printf("   Valuesets:  %s\n", strings.Join(parts, ", "))
			}
			if len(lib.Parameters) > 0 {
				parts := make([]string, len(lib.Parameters))
				for j, p := range lib.Parameters {
					parts[j] = p.Name
				}
				fmt.Printf("   Parameters: %s\n", strings.Join(parts, ", "))
			}
			if lib.Context != nil {
				fmt.Printf("   Context:    %s\n", lib.Context.Name)
			}
			if len(lib.Statements) > 0 {
				fmt.Printf("   Statements: %d defined\n", len(lib.Statements))
				for _, stmt := range lib.Statements {
					kind := "define"
					if stmt.IsFunction {
						kind = "function"
					}
					fmt.Printf("     - [%s] %s", kind, stmt.Name)
					if stmt.IsFluent {
						fmt.Print(" (fluent)")
					}
					if len(stmt.Operands) > 0 {
						params := make([]string, len(stmt.Operands))
						for j, op := range stmt.Operands {
							params[j] = op.Name
						}
						fmt.Printf("(%s)", strings.Join(params, ", "))
					}
					fmt.Println()
				}
			}
		}

		if result.HasErrors() {
			fmt.Printf("   DIAGNOSTICS (%d):\n", len(result.Diagnostics))
			for _, d := range result.Diagnostics {
				fmt.Printf("     %s\n", d)
			}
		} else {
			fmt.Printf("   ✓ Parsed successfully (0 errors)\n")
		}
		fmt.Println()
	}

	// Also parse any .cql files passed as arguments
	if len(os.Args) > 1 {
		fmt.Println("── User-provided files:")
		for _, path := range os.Args[1:] {
			src, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("   ERROR reading %s: %v\n", path, err)
				continue
			}
			result, err := parser.ParseString(string(src), path)
			if err != nil {
				fmt.Printf("   GO ERROR: %v\n", err)
				continue
			}
			if result.HasErrors() {
				fmt.Printf("   %s: %d error(s)\n", path, len(result.Diagnostics))
				for _, d := range result.Diagnostics {
					fmt.Printf("     %s\n", d)
				}
			} else {
				libName := "<anonymous>"
				if result.Library != nil && result.Library.Name != nil {
					libName = result.Library.Name.Name
				}
				fmt.Printf("   %s: OK — library %s, %d statements\n",
					path, libName, len(result.Library.Statements))
			}
		}
		fmt.Println()
	}

	if allPassed {
		fmt.Println("Phase 1 complete ✓  CQL lexer/parser produces AST and diagnostics.")
	} else {
		fmt.Fprintln(os.Stderr, "Phase 1: some examples failed — see output above.")
		os.Exit(1)
	}
}

func preview(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ↵ ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
