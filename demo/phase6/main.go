// Phase 6 demo: LibrarySource resolver + CQF parity annotations.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║      echo-elm  Phase 6  —  CQF Parity Demo        ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	ok := true
	ok = ok && demoModernMode()
	ok = ok && demoCQFMode()
	ok = ok && demoLibrarySource()
	ok = ok && demoRetrieveFilters()

	fmt.Println()
	if ok {
		fmt.Println("Phase 6 complete ✓  CQF mode annotations, LibrarySource resolver.")
	} else {
		fmt.Println("Phase 6 FAILED — see errors above.")
		os.Exit(1)
	}
}

func demoModernMode() bool {
	fmt.Println("── Example 1: Modern mode — translatorOptions is non-empty")
	cql := []byte(`library Modern version '1.0.0'
using FHIR version '4.0.1'`)
	result, err := echoelm.Translate(cql, "Modern.cql")
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
		return false
	}
	b, _ := json.MarshalIndent(result, "   ", "  ")
	// Extract translatorOptions
	var env map[string]any
	_ = json.Unmarshal(b, &env)
	lib := env["library"].(map[string]any)
	ann := lib["annotation"].([]any)
	info := ann[0].(map[string]any)
	opts, _ := info["translatorOptions"].(string)
	fmt.Printf("   translatorOptions = %q\n", opts)
	if opts == "" {
		fmt.Println("   FAIL: modern mode translatorOptions should not be empty")
		return false
	}
	// UsingDef annotations should be nil/absent
	usings := lib["usings"].(map[string]any)
	defs := usings["def"].([]any)
	for _, d := range defs {
		dm := d.(map[string]any)
		if _, hasAnn := dm["annotation"]; hasAnn {
			fmt.Println("   FAIL: modern mode UsingDef should not have annotation field")
			return false
		}
	}
	fmt.Println("   ✓ translatorOptions set, no annotation arrays on defs")
	return true
}

func demoCQFMode() bool {
	fmt.Println()
	fmt.Println("── Example 2: CQF mode — translatorOptions=\"\", annotation:[] on all defs")
	cql := []byte(`library CQFTest version '1.0.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define "Initial Population": true`)
	result, err := echoelm.Translate(cql, "CQFTest.cql", echoelm.WithCQFMode(true))
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
		return false
	}
	b, _ := json.Marshal(result)
	var env map[string]any
	_ = json.Unmarshal(b, &env)
	lib := env["library"].(map[string]any)

	// translatorOptions must be ""
	ann := lib["annotation"].([]any)
	info := ann[0].(map[string]any)
	opts, _ := info["translatorOptions"].(string)
	if opts != "" {
		fmt.Printf("   FAIL: CQF translatorOptions = %q (want empty)\n", opts)
		return false
	}
	fmt.Printf("   translatorOptions = %q ✓\n", opts)

	// UsingDefs all have annotation:[]
	usings := lib["usings"].(map[string]any)
	defs := usings["def"].([]any)
	for _, d := range defs {
		dm := d.(map[string]any)
		annField, ok := dm["annotation"]
		if !ok {
			fmt.Printf("   FAIL: UsingDef %q missing annotation\n", dm["localIdentifier"])
			return false
		}
		arr, ok := annField.([]any)
		if !ok || len(arr) != 0 {
			fmt.Printf("   FAIL: UsingDef annotation should be []\n")
			return false
		}
	}
	fmt.Printf("   %d UsingDef(s) have annotation:[] ✓\n", len(defs))

	// ContextDef has annotation:[]
	if contexts, ok := lib["contexts"].(map[string]any); ok {
		cdefs := contexts["def"].([]any)
		for _, cd := range cdefs {
			cdm := cd.(map[string]any)
			if _, ok := cdm["annotation"]; !ok {
				fmt.Printf("   FAIL: ContextDef %q missing annotation\n", cdm["name"])
				return false
			}
		}
		fmt.Printf("   %d ContextDef(s) have annotation:[] ✓\n", len(cdefs))
	}

	// StatementDefs all have annotation:[]
	if stmts, ok := lib["statements"].(map[string]any); ok {
		sdefs := stmts["def"].([]any)
		for _, sd := range sdefs {
			sdm := sd.(map[string]any)
			if _, ok := sdm["annotation"]; !ok {
				fmt.Printf("   FAIL: StatementDef %q missing annotation\n", sdm["name"])
				return false
			}
		}
		fmt.Printf("   %d StatementDef(s) have annotation:[] ✓\n", len(sdefs))
	}

	return true
}

func demoLibrarySource() bool {
	fmt.Println()
	fmt.Println("── Example 3: LibrarySource — MapSource + MultiSource")

	// MapSource lookup
	mapSrc := resolver.NewMapSource(map[string][]byte{
		"FHIRHelpers|4.0.1": []byte("library FHIRHelpers version '4.0.1'"),
		"FHIRHelpers":       []byte("library FHIRHelpers version '4.0.1'"),
	})
	src, ok, err := mapSrc.GetLibrarySource("FHIRHelpers", "4.0.1")
	if err != nil || !ok || !strings.Contains(string(src), "FHIRHelpers") {
		fmt.Printf("   FAIL: MapSource versioned lookup: ok=%v err=%v\n", ok, err)
		return false
	}
	fmt.Println("   MapSource versioned lookup ✓")

	// Unversioned fallback
	_, ok, err = mapSrc.GetLibrarySource("FHIRHelpers", "")
	if err != nil || !ok {
		fmt.Printf("   FAIL: MapSource unversioned fallback: ok=%v err=%v\n", ok, err)
		return false
	}
	fmt.Println("   MapSource unversioned fallback ✓")

	// MultiSource
	m1 := resolver.NewMapSource(map[string][]byte{"LibA": []byte("library LibA")})
	m2 := resolver.NewMapSource(map[string][]byte{"LibB": []byte("library LibB")})
	multi := resolver.NewMultiSource(m1, m2)
	_, okA, _ := multi.GetLibrarySource("LibA", "")
	_, okB, _ := multi.GetLibrarySource("LibB", "")
	_, okC, _ := multi.GetLibrarySource("LibC", "")
	if !okA || !okB || okC {
		fmt.Printf("   FAIL: MultiSource: A=%v B=%v C=%v\n", okA, okB, okC)
		return false
	}
	fmt.Println("   MultiSource chain (A✓ B✓ C-not-found✓) ✓")

	// DirSource with temp directory
	tmp, _ := os.MkdirTemp("", "echo-elm-test-*")
	defer os.RemoveAll(tmp)
	_ = os.WriteFile(tmp+"/TestLib-1.0.0.cql", []byte("library TestLib version '1.0.0'"), 0o644)
	dirSrc := resolver.NewDirSource(tmp)
	srcBytes, ok2, err2 := dirSrc.GetLibrarySource("TestLib", "1.0.0")
	if err2 != nil || !ok2 || !strings.Contains(string(srcBytes), "TestLib") {
		fmt.Printf("   FAIL: DirSource: ok=%v err=%v\n", ok2, err2)
		return false
	}
	fmt.Println("   DirSource versioned file lookup ✓")

	return true
}

func demoRetrieveFilters() bool {
	fmt.Println()
	fmt.Println("── Example 4: RetrieveNode CQF filter arrays")
	cql := []byte(`library PatientLib version '1.0.0'
using FHIR version '4.0.1'
context Patient`)
	result, err := echoelm.Translate(cql, "PatientLib.cql", echoelm.WithCQFMode(true))
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
		return false
	}
	b, _ := json.Marshal(result)
	var env map[string]any
	_ = json.Unmarshal(b, &env)
	lib := env["library"].(map[string]any)

	// Find the implicit Patient accessor statement's SingletonFrom.Retrieve
	stmts, ok := lib["statements"].(map[string]any)
	if !ok {
		fmt.Println("   FAIL: no statements")
		return false
	}
	sdefs := stmts["def"].([]any)
	found := false
	for _, sd := range sdefs {
		sdm := sd.(map[string]any)
		if sdm["name"] != "Patient" {
			continue
		}
		expr, ok := sdm["expression"].(map[string]any)
		if !ok {
			continue
		}
		if expr["type"] != "SingletonFrom" {
			continue
		}
		operand, ok := expr["operand"].(map[string]any)
		if !ok || operand["type"] != "Retrieve" {
			continue
		}
		// Check all four filter arrays
		for _, field := range []string{"include", "codeFilter", "dateFilter", "otherFilter"} {
			arr, ok := operand[field].([]any)
			if !ok || len(arr) != 0 {
				fmt.Printf("   FAIL: Retrieve.%s should be [] in CQF mode\n", field)
				return false
			}
		}
		found = true
		break
	}
	if !found {
		fmt.Println("   FAIL: Patient accessor not found in statements")
		return false
	}
	fmt.Println("   Retrieve: include:[], codeFilter:[], dateFilter:[], otherFilter:[] ✓")
	return true
}
