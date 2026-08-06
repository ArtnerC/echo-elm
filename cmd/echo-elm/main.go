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
	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/ui"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

// Version is set at build time via -ldflags.
var Version = "0.0.0-dev"

// multiFlag collects a repeatable string flag into a slice.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

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
	case "bundle":
		runBundle(os.Args[2:])
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
	fmt.Fprintln(os.Stderr, "  bundle           Read/write CQL and ELM in a FHIR Bundle")
	fmt.Fprintln(os.Stderr, "  parity           Run CQFramework parity harness")
	fmt.Fprintln(os.Stderr, "  version          Print version")
}

// runTranslate implements both `translate` and `cqf translate` subcommands.
// cqfMode applies cqframework-compatible defaults (signatureLevel=None, etc.).
func runTranslate(args []string, cqfMode bool) {
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)

	var (
		input    string
		output   string
		format   string
		validate bool
	)
	var libDirs multiFlag

	fs.StringVar(&input, "input", "", "Input CQL file (required)")
	fs.StringVar(&output, "output", "", "Output file or directory (default: next to input)")
	fs.StringVar(&format, "format", "JSON", "Output format: JSON or XML")
	fs.BoolVar(&validate, "validate", false, "Run structural validation on the serialized ELM before writing it")
	fs.Var(&libDirs, "lib-dir", "Additional directory to search for included CQL libraries (repeatable; the input file's own directory is always searched)")
	tf := registerTranslatorFlags(fs, cqfMode)

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

	// Includes resolve from the input file's own directory first (matching the
	// CQF CLI), then from any --lib-dir directories in the order given.
	searchDirs := append([]string{filepath.Dir(input)}, libDirs...)
	sources := make([]echoelm.LibrarySource, 0, len(searchDirs))
	for _, dir := range searchDirs {
		sources = append(sources, resolver.NewDirSource(dir))
	}

	opts := []echoelm.Option{
		echoelm.WithOptions(tf.options()),
		echoelm.WithCQFMode(cqfMode),
		echoelm.WithLibrarySource(resolver.NewMultiSource(sources...)),
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
		refDir     string
		parityLib  string
		profile    string
		bundlePath string
	)
	fs.StringVar(&cqfVersion, "cqf-version", "4.8.0", "CQFramework version to compare against: 3.29.0|4.8.0")
	fs.StringVar(&corpus, "corpus", "test/corpus/cqframework", "Corpus directory containing corpus.yaml")
	fs.StringVar(&toolsDir, "tools-dir", "tools", "Tools directory containing cqframework/ and jdk/")
	fs.StringVar(&tag, "tag", "", "Run only fixtures with this tag (default: all)")
	fs.StringVar(&outDir, "out", "", "Output directory for report.json and report.md (default: parity/runs/<id>)")
	fs.StringVar(&runID, "run-id", "", "Run identifier (default: timestamp)")
	fs.StringVar(&refDir, "ref-dir", "", "Read reference ELM from <ref-dir>/<profile>/<fixture>.json instead of running the CQF JAR")
	fs.StringVar(&parityLib, "lib-dir", "", "Additional directory to search for included CQL libraries")
	fs.StringVar(&profile, "profile", "", "Run only fixtures under this option profile (default: all)")
	fs.StringVar(&bundlePath, "bundle", "", "Compare against the ELM inside a FHIR Bundle, taking its libraries as the corpus")

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
		CQFVersion:    cqfVersion,
		ToolsDir:      toolsDir,
		CorpusDir:     corpus,
		TagFilter:     tag,
		ProfileFilter: profile,
		RefDir:        refDir,
		LibDir:        parityLib,
	}
	label := referenceLabel(cqfVersion, refDir)

	// --bundle lays the bundle out as a corpus and compares against the ELM it
	// already carries, which is the extract-then-compare workflow in one step.
	// The extracted corpus is removed once the run is done. Cleanup is not
	// deferred: several paths below end in os.Exit, which would skip it.
	var bundleWorkDir string
	if bundlePath != "" {
		workDir, err := os.MkdirTemp("", "echo-elm-bundle-")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		bundleWorkDir = workDir

		bundleCfg, err := parity.MaterializeBundle(loadBundle(bundlePath), workDir, profile)
		if err != nil {
			_ = os.RemoveAll(workDir)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		bundleCfg.TagFilter = tag
		cfg = bundleCfg
		corpus = bundlePath
		label = "bundle=" + filepath.Base(bundlePath)
	}

	fmt.Fprintf(os.Stderr, "echo-elm parity  run=%s  %s  corpus=%s\n", runID, label, corpus)

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
	if bundleWorkDir != "" {
		_ = os.RemoveAll(bundleWorkDir)
	}
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

// referenceLabel names where a parity run takes its reference ELM from.
func referenceLabel(cqfVersion, refDir string) string {
	if refDir != "" {
		return "ref-dir=" + refDir
	}
	return "cqf=" + cqfVersion
}
