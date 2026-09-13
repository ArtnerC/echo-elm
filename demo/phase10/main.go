// Phase 10 demo — CQFramework parity harness
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

func main() {
	box("echo-elm  Phase 10  —  Parity Harness Demo")

	// ── Check 1: Corpus NOTICE + PROVENANCE present ─────────────────────────
	fmt.Println("── Check 1: Corpus metadata files present")
	corpusDir := fixtureCorpus()
	check("NOTICE exists", fileExists(filepath.Join(corpusDir, "NOTICE")))
	check("PROVENANCE.md exists", fileExists(filepath.Join(corpusDir, "PROVENANCE.md")))
	check("corpus.yaml exists", fileExists(filepath.Join(corpusDir, "corpus.yaml")))
	fmt.Println()

	// ── Check 2: Corpus fixture count ───────────────────────────────────────
	fmt.Println("── Check 2: Corpus has ≥ 8 fixtures")
	corpus, err := parity.LoadCorpus(corpusDir)
	must("load corpus", err)
	fmt.Printf("   loaded %d fixtures from corpus.yaml\n", len(corpus.Fixtures))
	check("corpus has ≥ 8 fixtures", len(corpus.Fixtures) >= 8)
	fmt.Println()

	// ── Check 3: CQFramework launchers present ───────────────────────────────
	fmt.Println("── Check 3: CQFramework launchers available")
	toolsDir := fixtureTools()
	launchersOK := true
	for _, v := range []string{parity.PinnedCQFVersion} {
		ext := ".sh"
		if runtime.GOOS == "windows" {
			ext = ".bat"
		}
		launcher := filepath.Join(toolsDir, "cqframework", v, "run"+ext)
		ok := fileExists(launcher)
		fmt.Printf("   cqframework %s launcher: %v\n", v, ok)
		if !ok {
			launchersOK = false
		}
	}
	check("all launchers present", launchersOK)
	fmt.Println()

	// ── Check 4: Parity run (smoke tag) ─────────────────────────────────────
	fmt.Printf("── Check 4: Parity run (smoke tag, cqframework %s)\n", parity.PinnedCQFVersion)
	cfg := parity.Config{
		CQFVersion: parity.PinnedCQFVersion,
		ToolsDir:   toolsDir,
		CorpusDir:  corpusDir,
		TagFilter:  "smoke",
	}

	results, err := parity.Run(cfg, echoTranslate)
	must("run parity", err)

	summary := parity.Summary(results)
	for i := range results {
		r := &results[i]
		icon := statusIcon(r.Status)
		fmt.Printf("   %s  %-45s  %s\n", icon, r.Fixture, r.Status)
		if r.Error != "" {
			fmt.Printf("      err: %s\n", r.Error)
		}
	}
	fmt.Printf("\n   match=%d  differ=%d  error=%d  total=%d\n",
		summary[parity.StatusMatch], summary[parity.StatusDifferJSON],
		summary[parity.StatusEchoError]+summary[parity.StatusUpstreamError], len(results))
	// At least one fixture ran (even if differ; full match is a future goal)
	check("at least one fixture ran", len(results) > 0)
	fmt.Println()

	// ── Check 5: Report written ──────────────────────────────────────────────
	fmt.Println("── Check 5: Report files written")
	runID := time.Now().UTC().Format("20060102-150405")
	outDir := filepath.Join(os.TempDir(), "echo-elm-parity-demo-"+runID)
	err = parity.WriteReport(outDir, runID, parity.PinnedCQFVersion, results)
	must("write report", err)
	check("report.json written", fileExists(filepath.Join(outDir, "report.json")))
	check("report.md written", fileExists(filepath.Join(outDir, "report.md")))

	// Verify report.json is valid JSON with expected fields.
	data, _ := os.ReadFile(filepath.Join(outDir, "report.json"))
	var rep map[string]any
	must("parse report.json", json.Unmarshal(data, &rep))
	check("report has total field", rep["total"] != nil)
	check("report has fixtures array", rep["fixtures"] != nil)
	fmt.Printf("   report written to: %s\n", outDir)
	fmt.Println()

	fmt.Println("Phase 10 complete ✓  Parity harness running with CQFramework corpus.")
	fmt.Println()
	fmt.Println("  Known parity gaps (expected until Phase 5/6 hardening):")
	fmt.Println("  • signatureLevel: echo-elm defaults to 'Overloads'; cqf CLI to 'None'")
	fmt.Println("  • ELM contexts section not yet emitted (Library.contexts)")
	fmt.Println("  • Implicit Patient accessor statement not yet generated")
	fmt.Println("  • Full type system / operator resolution still in progress")
}

func echoTranslate(cqlPath string) ([]byte, error) {
	data, err := os.ReadFile(cqlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cqlPath, err)
	}
	sourceName := filepath.Base(cqlPath)
	result, err := echoelm.Translate(data, sourceName, echoelm.WithAnnotations(true))
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(map[string]any{"library": result.Library}, "", "  ")
}

func statusIcon(s parity.Status) string {
	switch s {
	case parity.StatusMatch:
		return "✓"
	case parity.StatusDifferJSON:
		return "≠"
	default:
		return "✗"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fixtureCorpus() string {
	_, file, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(file), "..", "..", "test", "corpus", "cqframework")
	abs, _ := filepath.Abs(p)
	return abs
}

func fixtureTools() string {
	_, file, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(file), "..", "..", "tools")
	abs, _ := filepath.Abs(p)
	return abs
}

func check(label string, ok bool) {
	if ok {
		fmt.Printf("   ✓ %s\n", label)
	} else {
		fmt.Fprintf(os.Stderr, "   ✗ FAIL: %s\n", label)
		os.Exit(1)
	}
}

func must(label string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %s: %v\n", label, err)
		os.Exit(1)
	}
}

func box(title string) {
	pad := strings.Repeat("═", len(title)+4)
	fmt.Printf("╔%s╗\n║  %s  ║\n╚%s╝\n\n", pad, title, pad)
}
