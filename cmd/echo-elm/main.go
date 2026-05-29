// Command echo-elm is a CQL→ELM translator.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/mcpserver"
	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/internal/translator"
	"github.com/artnerc/echo-elm/internal/ui"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

// Version is set at build time via -ldflags.
var Version = "0.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("echo-elm %s\n", Version)
	case "translate":
		runTranslate(os.Args[2:], false)
	case "cqf":
		if len(os.Args) < 3 || os.Args[2] != "translate" {
			fmt.Fprintln(os.Stderr, "Usage: echo-elm cqf translate [flags]")
			os.Exit(2)
		}
		runTranslate(os.Args[3:], true)
	case "ui":
		runUI(os.Args[2:])
	case "mcp":
		runMCP(os.Args[2:])
	case "parity":
		runParity(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "echo-elm %s\n", Version)
	fmt.Fprintln(os.Stderr, "Usage: echo-elm <command> [flags]")
	fmt.Fprintln(os.Stderr, "  translate        Translate CQL to ELM (modern defaults)")
	fmt.Fprintln(os.Stderr, "  cqf translate    Translate CQL to ELM (cqframework-compatible)")
	fmt.Fprintln(os.Stderr, "  ui               Start local workbench UI")
	fmt.Fprintln(os.Stderr, "  mcp              Start MCP server")
	fmt.Fprintln(os.Stderr, "  parity           Run CQFramework parity harness")
	fmt.Fprintln(os.Stderr, "  version          Print version")
}

// runTranslate implements both `translate` and `cqf translate` subcommands.
// cqfMode applies cqframework-compatible defaults (signatureLevel=None, etc.).
func runTranslate(args []string, cqfMode bool) {
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)

	var (
		input                   string
		output                  string
		format                  string
		annotations             bool
		locators                bool
		sigLevel                string
		validate                bool
		disableListDemotion     bool
		disableListPromotion    bool
		disableListTraversal    bool
		disableMethodInvocation bool
		requireFromKeyword      bool
		enableIntervalDemotion  bool
		enableIntervalPromotion bool
		compatLevel             string
	)

	fs.StringVar(&input, "input", "", "Input CQL file (required)")
	fs.StringVar(&output, "output", "", "Output file or directory (default: next to input)")
	fs.StringVar(&format, "format", "JSON", "Output format: JSON or XML")
	fs.BoolVar(&validate, "validate", false, "Run structural validation on the serialized ELM before writing it")

	// Default flags differ by mode to match each mode's natural behavior.
	annotationsDefault := !cqfMode // modern: true, CQF: false (no annotation content without --annotations)
	locatorsDefault := !cqfMode    // modern: true, CQF: false (no localId without --locators)

	fs.BoolVar(&annotations, "annotations", annotationsDefault, "Emit ELM annotations")
	fs.BoolVar(&locators, "locators", locatorsDefault, "Emit source locators")
	fs.BoolVar(&disableListDemotion, "disable-list-demotion", false, "Disable implicit list demotion")
	fs.BoolVar(&disableListPromotion, "disable-list-promotion", false, "Disable implicit list promotion")
	fs.BoolVar(&disableListTraversal, "disable-list-traversal", false, "Disable implicit list traversal")
	fs.BoolVar(&disableMethodInvocation, "disable-method-invocation", false, "Disable method-style invocation syntax")
	fs.BoolVar(&requireFromKeyword, "require-from-keyword", false, "Require explicit 'from' in queries")
	fs.BoolVar(&enableIntervalDemotion, "enable-interval-demotion", false, "Enable implicit interval demotion")
	fs.BoolVar(&enableIntervalPromotion, "enable-interval-promotion", false, "Enable implicit interval promotion")

	if cqfMode {
		sigLevel = "None"
		compatLevel = "1.5"
		fs.StringVar(&sigLevel, "signatures", sigLevel, "Signature level: None|Differing|Overloads|All")
		fs.StringVar(&compatLevel, "compatibility-level", compatLevel, "Compatibility level: 1.3|1.4|1.5")
	} else {
		sigLevel = "Overloads"
		compatLevel = "1.5"
		fs.StringVar(&sigLevel, "signatures", sigLevel, "Signature level: None|Differing|Overloads|All")
		fs.StringVar(&compatLevel, "compatibility-level", compatLevel, "Compatibility level: 1.3|1.4|1.5")
	}

	// --strict expands: disable-list-traversal + demotion + promotion + method-invocation + require-from-keyword
	var strict bool
	fs.BoolVar(&strict, "strict", false, "Strict mode (disables list traversal, demotion, promotion, method invocation; requires from keyword)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		fs.Usage()
		os.Exit(2)
	}

	src, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", input, err)
		os.Exit(1)
	}

	opts := []echoelm.Option{
		echoelm.WithAnnotations(annotations),
		echoelm.WithLocators(locators),
		echoelm.WithSignatureLevel(sigLevel),
		echoelm.WithCQFMode(cqfMode),
		echoelm.WithIntervalDemotion(enableIntervalDemotion),
		echoelm.WithIntervalPromotion(enableIntervalPromotion),
		func(o *translator.Options) {
			if strict {
				o.DisableListTraversal = true
				o.DisableListDemotion = true
				o.DisableListPromotion = true
				o.DisableMethodInvocation = true
				o.RequireFromKeyword = true
			} else {
				o.DisableListDemotion = disableListDemotion
				o.DisableListPromotion = disableListPromotion
				o.DisableListTraversal = disableListTraversal
				o.DisableMethodInvocation = disableMethodInvocation
				o.RequireFromKeyword = requireFromKeyword
			}
			o.CompatibilityLevel = compatLevel
		},
	}

	result, err := echoelm.Translate(src, filepath.Base(input), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Report diagnostics to stderr.
	hasErrors := false
	for _, d := range result.Diagnostics {
		fmt.Fprintf(os.Stderr, "%s:[n/a] %s\n", d.Severity, d.Message)
		if strings.EqualFold(d.Severity, "error") {
			hasErrors = true
		}
	}

	if cqfMode {
		// CQF banner to stderr.
		sep := strings.Repeat("=", 80)
		fmt.Fprintf(os.Stderr, "%s\n", sep)
		fmt.Fprintf(os.Stderr, "TRANSLATE %s\n", input)
		if hasErrors {
			fmt.Fprintln(os.Stderr, "Translation failed due to errors:")
		} else {
			fmt.Fprintln(os.Stderr, "Translation completed successfully.")
		}
	}

	if hasErrors {
		os.Exit(1)
	}

	// Determine output path.
	outPath := resolveOutput(input, output, format)

	// Serialize.
	var outBytes []byte
	switch strings.ToUpper(format) {
	case "XML":
		outBytes, err = result.XMLBytes()
	default: // JSON
		outBytes, err = json.MarshalIndent(result, "", "   ")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal: %v\n", err)
		os.Exit(1)
	}

	if validate {
		vfmt := "json"
		if strings.EqualFold(format, "XML") {
			vfmt = "xml"
		}
		if verr := echoelm.Validate(outBytes, vfmt); verr != nil {
			fmt.Fprintf(os.Stderr, "Warning:[n/a] ELM validation: %v\n", verr)
		}
	}

	if err := os.WriteFile(outPath, outBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	if cqfMode {
		fmt.Fprintf(os.Stderr, "ELM output written to: %s\n", outPath)
	} else {
		fmt.Printf("%s\n", outPath)
	}
}

// runMCP starts the MCP server over stdio.
func runMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	var (
		workdir    string
		allowWrite bool
	)
	fs.StringVar(&workdir, "workdir", "", "Workspace directory (default: cwd)")
	fs.BoolVar(&allowWrite, "allow-write", false, "Enable write-capable tools")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if err := mcpserver.Run(context.Background(), mcpserver.Options{
		Workdir:    workdir,
		Version:    Version,
		AllowWrite: allowWrite,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "echo-elm mcp: %v\n", err)
		os.Exit(1)
	}
}

// runUI starts the local workbench server.
func runUI(args []string) {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	var (
		addr        string
		workspace   string
		allowRemote bool
	)
	fs.StringVar(&addr, "addr", "127.0.0.1:8787", "Listen address")
	fs.StringVar(&workspace, "workspace", "", "Workspace directory (default: cwd)")
	fs.BoolVar(&allowRemote, "allow-remote", false, "Allow non-loopback bind (unsafe)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	s, err := ui.NewServer(ui.ServerOptions{
		Workspace:   workspace,
		Version:     Version,
		AllowRemote: allowRemote,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "echo-elm ui: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "echo-elm ui  %s  workspace=%s\n", Version, workspace)
	fmt.Fprintf(os.Stderr, "Listening on http://%s\n", addr)

	if err := s.ListenAndServe(addr); err != nil {
		fmt.Fprintf(os.Stderr, "echo-elm ui: %v\n", err)
		os.Exit(1)
	}
}

// runParity runs the CQFramework parity harness.
func runParity(args []string) {
	fs := flag.NewFlagSet("parity", flag.ContinueOnError)
	var (
		cqfVersion string
		corpus     string
		toolsDir   string
		tag        string
		outDir     string
		runID      string
	)
	fs.StringVar(&cqfVersion, "cqf-version", "4.8.0", "CQFramework version to compare against: 3.29.0|4.8.0")
	fs.StringVar(&corpus, "corpus", "test/corpus/cqframework", "Corpus directory containing corpus.yaml")
	fs.StringVar(&toolsDir, "tools-dir", "tools", "Tools directory containing cqframework/ and jdk/")
	fs.StringVar(&tag, "tag", "", "Run only fixtures with this tag (default: all)")
	fs.StringVar(&outDir, "out", "", "Output directory for report.json and report.md (default: parity/runs/<id>)")
	fs.StringVar(&runID, "run-id", "", "Run identifier (default: timestamp)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if runID == "" {
		runID = time.Now().UTC().Format("20060102-150405")
	}
	if outDir == "" {
		outDir = filepath.Join("parity", "runs", runID)
	}

	cfg := parity.Config{
		CQFVersion: cqfVersion,
		ToolsDir:   toolsDir,
		CorpusDir:  corpus,
		TagFilter:  tag,
	}

	fmt.Fprintf(os.Stderr, "echo-elm parity  run=%s  cqf=%s  corpus=%s\n", runID, cqfVersion, corpus)

	results, err := parity.Run(cfg, func(cqlPath string) ([]byte, error) {
		data, err := os.ReadFile(cqlPath)
		if err != nil {
			return nil, err
		}
		// Use CQF defaults: EnableLocators=false, EnableAnnotations=false,
		// SignatureLevel=None, CQFMode=true — matching upstream cqframework CLI.
		result, err := echoelm.Translate(data, filepath.Base(cqlPath), echoelm.WithCQFOptions())
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(map[string]any{"library": result.Library}, "", "  ")
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	summary := parity.Summary(results)
	for i := range results {
		r := &results[i]
		icon := "✓"
		if r.Status != parity.StatusMatch {
			icon = "≠"
		}
		if r.Status == parity.StatusEchoError || r.Status == parity.StatusUpstreamError {
			icon = "✗"
		}
		fmt.Printf("%s  %-45s  %s\n", icon, r.Fixture, r.Status)
	}
	fmt.Printf("\nmatch=%d  differ=%d  error=%d  total=%d\n",
		summary[parity.StatusMatch], summary[parity.StatusDifferJSON],
		summary[parity.StatusEchoError]+summary[parity.StatusUpstreamError], len(results))

	if err := parity.WriteReport(outDir, runID, cqfVersion, results); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write report: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "report written to: %s\n", outDir)
	}

	if summary[parity.StatusEchoError]+summary[parity.StatusUpstreamError] > 0 {
		os.Exit(1)
	}
}

// resolveOutput determines the output file path from the input path, --output flag, and format.
func resolveOutput(input, output, format string) string {
	ext := ".json"
	if strings.EqualFold(format, "XML") {
		ext = ".xml"
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))

	if output == "" {
		return filepath.Join(filepath.Dir(input), base+ext)
	}

	// If output is a directory, write file inside it.
	info, err := os.Stat(output)
	if err == nil && info.IsDir() {
		return filepath.Join(output, base+ext)
	}
	return output
}
