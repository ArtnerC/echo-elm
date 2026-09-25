// Command echo-elm is a CQL→ELM translator.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/mcpserver"
	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/translator"
	"github.com/artnerc/echo-elm/internal/ui"
	"github.com/artnerc/echo-elm/internal/workers"
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
//
// --input takes a file or a directory, and may be repeated. A directory is
// searched recursively for .cql files, as the CQF CLI does. Libraries are
// translated --jobs at a time and share one include cache, while everything
// written to stdout and stderr comes out in input order, so a run's output is
// the same whatever the number of jobs.
func runTranslate(args []string, cqfMode bool) {
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)

	var (
		output   string
		format   string
		target   string
		validate bool
		jobs     int
	)
	var inputs, libDirs multiFlag

	fs.Var(&inputs, "input", "Input CQL file or directory, searched recursively for .cql files (required; repeatable)")
	fs.StringVar(&output, "output", "", "Output file, or directory when translating more than one file (default: next to each input)")
	fs.StringVar(&format, "format", "JSON", "Output format: JSON or XML")
	fs.StringVar(&target, "target", "cqf", "ELM shape: cqf (cql-to-elm CLI parity) or bundle (fully type-discriminated, as embedded in FHIR Library resources)")
	fs.BoolVar(&validate, "validate", false, "Run structural validation on the serialized ELM before writing it")
	fs.Var(&libDirs, "lib-dir", "Additional directory to search for included CQL libraries (repeatable; the input file's own directory is always searched)")
	fs.IntVar(&jobs, "jobs", 0, "Libraries to translate at once (0: one per CPU less one, or $"+workers.EnvJobs+")")
	tf := registerTranslatorFlags(fs, cqfMode)

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		fs.Usage()
		os.Exit(2)
	}
	bundleShape := strings.EqualFold(target, "bundle")
	if !bundleShape && !strings.EqualFold(target, "cqf") {
		fmt.Fprintf(os.Stderr, "error: --target must be cqf or bundle (got %q)\n", target)
		os.Exit(2)
	}
	if bundleShape && strings.EqualFold(format, "XML") {
		// The discriminated shape is a property of the elm-json writer; XML
		// already carries xsi:type on every node.
		fmt.Fprintln(os.Stderr, "error: --target bundle applies to JSON output only")
		os.Exit(2)
	}

	files, many, err := collectInputs(inputs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	outPaths, err := planOutputs(files, many, output, format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	job := translateJob{
		cqfMode:  cqfMode,
		format:   format,
		bundle:   bundleShape,
		validate: validate,
		libDirs:  libDirs,
		options:  tf.options(),
		cache:    echoelm.NewIncludeCache(),
	}
	exit := 0
	workers.Ordered(len(files), jobs, func(i int) fileOutcome {
		return job.run(files[i], outPaths[i])
	}, func(_ int, o fileOutcome) {
		_, _ = os.Stderr.Write(o.stderr.Bytes())
		_, _ = os.Stdout.Write(o.stdout.Bytes())
		exit = max(exit, o.exit)
	})
	os.Exit(exit)
}

// collectInputs expands each --input into the CQL files to translate: a file as
// given, a directory as every .cql file beneath it in lexical order. many
// reports whether the run is a multi-file one — more than one input, or any
// directory — whose --output must then be a directory.
func collectInputs(inputs []string) (files []string, many bool, err error) {
	seen := map[string]bool{}
	add := func(p string) {
		if key := filepath.Clean(p); !seen[key] {
			seen[key] = true
			files = append(files, p)
		}
	}
	for _, in := range inputs {
		info, err := os.Stat(in)
		if err != nil {
			return nil, false, fmt.Errorf("cannot read %s: %w", in, err)
		}
		if !info.IsDir() {
			add(in)
			continue
		}
		many = true
		err = filepath.WalkDir(in, func(p string, d iofs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".cql") {
				add(p)
			}
			return nil
		})
		if err != nil {
			return nil, false, err
		}
	}
	if len(files) == 0 {
		return nil, false, fmt.Errorf("no .cql files found in %s", strings.Join(inputs, ", "))
	}
	return files, many || len(files) > 1, nil
}

// planOutputs picks each input's output path. A single file keeps the old
// rules (resolveOutput). With many, --output is a directory — created when
// missing — that every output goes into by base name, as the CQF CLI does, or
// when absent each output goes next to its input. Two inputs that would write
// the same file are an error, not a silent overwrite.
func planOutputs(files []string, many bool, output, format string) ([]string, error) {
	if !many {
		return []string{resolveOutput(files[0], output, format)}, nil
	}
	if output != "" {
		if info, err := os.Stat(output); err == nil && !info.IsDir() {
			return nil, fmt.Errorf("--output %s is a file; it must be a directory when translating more than one file", output)
		}
		if err := os.MkdirAll(output, 0o755); err != nil {
			return nil, fmt.Errorf("create --output directory: %w", err)
		}
	}
	outs := make([]string, len(files))
	from := map[string]string{}
	for i, f := range files {
		outs[i] = resolveOutput(f, output, format)
		key := strings.ToLower(filepath.Clean(outs[i]))
		if prev, dup := from[key]; dup {
			return nil, fmt.Errorf("%s and %s would both be written to %s", prev, f, outs[i])
		}
		from[key] = f
	}
	return outs, nil
}

// translateJob is what every file in one translate run shares.
type translateJob struct {
	cqfMode  bool
	format   string
	bundle   bool
	validate bool
	libDirs  []string
	options  translator.Options
	cache    *echoelm.IncludeCache
}

// fileOutcome is what translating one file printed, held back so that parallel
// runs print in input order, and the exit code it earned.
type fileOutcome struct {
	stderr, stdout bytes.Buffer
	exit           int
}

// run translates one file to outPath.
func (j *translateJob) run(input, outPath string) fileOutcome {
	var o fileOutcome
	fail := func(code int, format string, args ...any) fileOutcome {
		fmt.Fprintf(&o.stderr, format, args...)
		o.exit = code
		return o
	}

	src, err := os.ReadFile(input)
	if err != nil {
		return fail(1, "error: cannot read %s: %v\n", input, err)
	}

	// Includes resolve from the input file's own directory first (matching the
	// CQF CLI), then from any --lib-dir directories in the order given.
	searchDirs := append([]string{filepath.Dir(input)}, j.libDirs...)
	sources := make([]echoelm.LibrarySource, 0, len(searchDirs))
	for _, dir := range searchDirs {
		sources = append(sources, resolver.NewDirSource(dir))
	}

	result, err := echoelm.Translate(src, filepath.Base(input),
		echoelm.WithOptions(j.options),
		echoelm.WithCQFMode(j.cqfMode),
		echoelm.WithLibrarySource(resolver.NewMultiSource(sources...)),
		echoelm.WithIncludeCache(j.cache),
	)
	if err != nil {
		return fail(1, "error: %v\n", err)
	}

	// Report diagnostics to stderr.
	hasErrors := false
	for _, d := range result.Diagnostics {
		fmt.Fprintf(&o.stderr, "%s:[n/a] %s\n", d.Severity, d.Message)
		if strings.EqualFold(d.Severity, "error") {
			hasErrors = true
		}
	}

	if j.cqfMode {
		// CQF banner to stderr.
		fmt.Fprintf(&o.stderr, "%s\n", strings.Repeat("=", 80))
		fmt.Fprintf(&o.stderr, "TRANSLATE %s\n", input)
		if hasErrors {
			fmt.Fprintln(&o.stderr, "Translation failed due to errors:")
		} else {
			fmt.Fprintln(&o.stderr, "Translation completed successfully.")
		}
	}
	if hasErrors {
		o.exit = 1
		return o
	}

	var outBytes []byte
	switch {
	case strings.EqualFold(j.format, "XML"):
		outBytes, err = result.XMLBytes()
	case j.bundle:
		outBytes, err = result.BundleJSON()
	default: // JSON
		outBytes, err = json.MarshalIndent(result, "", "   ")
	}
	if err != nil {
		return fail(1, "error: marshal: %v\n", err)
	}

	if j.validate {
		vfmt := "json"
		if strings.EqualFold(j.format, "XML") {
			vfmt = "xml"
		}
		if verr := echoelm.Validate(outBytes, vfmt); verr != nil {
			fmt.Fprintf(&o.stderr, "Warning:[n/a] ELM validation: %v\n", verr)
		}
	}

	if err := os.WriteFile(outPath, outBytes, 0o644); err != nil {
		return fail(1, "error: write %s: %v\n", outPath, err)
	}

	if j.cqfMode {
		fmt.Fprintf(&o.stderr, "ELM output written to: %s\n", outPath)
	} else {
		fmt.Fprintf(&o.stdout, "%s\n", outPath)
	}
	return o
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
		freshCQF   bool
		jobs       int
	)
	fs.StringVar(&cqfVersion, "cqf-version", parity.PinnedCQFVersion,
		"CQFramework version to compare against (default: the pinned release)")
	fs.StringVar(&corpus, "corpus", "test/corpus/cqframework", "Corpus directory containing corpus.yaml")
	fs.StringVar(&toolsDir, "tools-dir", "tools", "Tools directory containing cqframework/ and jdk/")
	fs.StringVar(&tag, "tag", "", "Run only fixtures with this tag (default: all)")
	fs.StringVar(&outDir, "out", "", "Output directory for report.json and report.md (default: parity/runs/<id>)")
	fs.StringVar(&runID, "run-id", "", "Run identifier (default: timestamp)")
	fs.StringVar(&refDir, "ref-dir", "", "Read reference ELM from <ref-dir>/<profile>/<fixture>.json instead of running the CQF JAR")
	fs.StringVar(&parityLib, "lib-dir", "", "Additional directory to search for included CQL libraries")
	fs.StringVar(&profile, "profile", "", "Run only fixtures under this option profile (default: all)")
	fs.StringVar(&bundlePath, "bundle", "", "Compare against the ELM inside a FHIR Bundle, taking its libraries as the corpus")
	fs.BoolVar(&freshCQF, "fresh-cqf", false, "With --bundle: compare against a fresh CQF compile of the bundle's CQL, with the options the bundle declares, instead of the ELM the bundle carries")
	fs.IntVar(&jobs, "jobs", 0, "Fixture runs at once (0: one per CPU less one, or $"+workers.EnvJobs+")")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if freshCQF && (bundlePath == "" || refDir != "") {
		fmt.Fprintln(os.Stderr, "error: --fresh-cqf requires --bundle and cannot be combined with --ref-dir")
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

		loaded := loadBundle(bundlePath)
		if !freshCQF {
			for _, w := range parity.ReferenceWarnings(loaded, parity.PinnedCQFVersion) {
				fmt.Fprintf(os.Stderr, "warning: %s\n", w)
			}
		}

		// An explicitly named profile is checked against what the bundle says it
		// was compiled with. Comparing a reference built with locators and result
		// types against a profile that emits neither produces thousands of
		// differences that are entirely the harness's doing, so it fails here
		// rather than being reported as translator gaps.
		if profile != "" {
			if err := validateBundleProfile(loaded, cfg.CorpusDir, profile); err != nil {
				_ = os.RemoveAll(workDir)
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}

		bundleCfg, err := parity.MaterializeBundle(loaded, workDir, profile)
		if err != nil {
			_ = os.RemoveAll(workDir)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		bundleCfg.TagFilter = tag
		cfg = bundleCfg
		corpus = bundlePath
		label = "bundle=" + filepath.Base(bundlePath)
		if freshCQF {
			// The bundle supplies the CQL and the options; the reference is
			// what CQF itself produces from them now, since the ELM a bundle
			// carries may come from some other build (issue #5). CQF CLI
			// output is CLI-shaped, so no bundle-shape reduction applies.
			cfg.RefDir = ""
			cfg.BundleShapedRef = false
			cfg.CQFVersion = cqfVersion
			cfg.ToolsDir = toolsDir
			label += " vs cqf=" + cqfVersion
		}
	}
	cfg.Jobs = jobs

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
