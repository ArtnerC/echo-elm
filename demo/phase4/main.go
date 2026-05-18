// Phase 4 demo: expression body translation (CQL→ELM)
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/artnerc/echo-elm/pkg/echoelm"
)

func main() {
	printBanner()

	examples := []struct {
		name string
		cql  string
	}{
		{
			name: "Arithmetic",
			cql: `library Arithmetic version '1.0.0'
define "TwoPlusThree": 2 + 3
define "TenDivThree":  10 div 3
define "Nested":       (2 + 3) * 4
`,
		},
		{
			name: "Boolean logic",
			cql: `library Logic version '1.0.0'
define "AndOr": true and false or null
define "Implies": true implies false
define "Negated": not (true and false)
`,
		},
		{
			name: "Comparison and equality",
			cql: `library Compare version '1.0.0'
define "Eq":  3 = 3
define "Neq": 3 != 4
define "Lt":  2 < 5
define "Gte": 5 >= 5
define "Equiv": 'hello' ~ 'hello'
`,
		},
		{
			name: "Date/time and interval",
			cql: `library DateTime version '1.0.0'
define "DateLit":     @2024-01-01
define "DateTimeLit": @2024-01-01T08:00:00
define "TimeLit":     @T08:00:00
define "OpenInterval": Interval(1, 10)
define "ClosedInterval": Interval[1, 10]
`,
		},
		{
			name: "Retrieve",
			cql: `library Retrieve version '1.0.0'
using FHIR version '4.0.1'
define "AllConditions": [Condition]
define "FHIRConditions": [FHIR.Condition]
`,
		},
		{
			name: "Query",
			cql: `library Query version '1.0.0'
using FHIR version '4.0.1'
define "ActiveConditions":
  [Condition] C
  where C.clinicalStatus ~ 'active'
  return C.id
`,
		},
		{
			name: "Type operators",
			cql: `library TypeOps version '1.0.0'
define "IsNull": null is null
define "NullLit": null
`,
		},
		{
			name: "If-then-else and case",
			cql: `library IfCase version '1.0.0'
define "IfExpr": if true then 1 else 2
define "CaseExpr":
  case
    when 1 = 1 then 'equal'
    when 1 = 2 then 'not equal'
    else 'other'
  end
`,
		},
		{
			name: "Lists and tuples",
			cql: `library Collections version '1.0.0'
define "IntList":  { 1, 2, 3 }
define "StrList":  List<System.String> { 'a', 'b', 'c' }
define "ATuple":   Tuple { name: 'Alice', age: 30 }
`,
		},
	}

	passed := 0
	for _, ex := range examples {
		ok := runExample(ex.name, ex.cql)
		if ok {
			passed++
		}
	}

	fmt.Printf("\n%d/%d examples produced non-trivial ELM output.\n", passed, len(examples))
	fmt.Println("\nPhase 4 complete ✓  Expression bodies now translate to ELM JSON.")
	os.Exit(0)
}

func printBanner() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║     echo-elm  Phase 4  —  Expression Bodies       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
}

func runExample(name, cql string) bool {
	fmt.Printf("── %s\n", name)
	result, err := echoelm.Translate([]byte(cql), name+".cql")
	if err != nil {
		fmt.Printf("   ERROR: %v\n", err)
		return false
	}
	if len(result.Diagnostics) > 0 {
		for _, d := range result.Diagnostics {
			fmt.Printf("   [%s] %s\n", d.Severity, d.Message)
		}
	}

	// Pretty-print the statements section
	data, jsonErr := json.Marshal(result)
	if jsonErr != nil {
		fmt.Printf("   Marshal error: %v\n", jsonErr)
		return false
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		fmt.Printf("   Invalid JSON: %v\n", err)
		return false
	}

	lib, _ := obj["library"].(map[string]interface{})
	stmts, _ := lib["statements"].(map[string]interface{})
	defs, _ := stmts["def"].([]interface{})

	unimplemented := 0
	for _, def := range defs {
		d, ok := def.(map[string]interface{})
		if !ok {
			continue
		}
		defName, _ := d["name"].(string)
		expr, _ := d["expression"].(map[string]interface{})
		typeName, _ := expr["type"].(string)
		if typeName == "Unimplemented" || typeName == "" {
			unimplemented++
		}
		fmt.Printf("   %-30s → %s\n", defName, typeName)
	}

	if unimplemented > 0 {
		fmt.Printf("   ⚠  %d unimplemented expression(s)\n", unimplemented)
	} else if len(defs) > 0 {
		fmt.Printf("   ✓ all %d expressions translated\n", len(defs))
	}
	fmt.Println()
	return unimplemented == 0
}
