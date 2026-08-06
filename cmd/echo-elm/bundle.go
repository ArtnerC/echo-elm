package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/pkg/bundle"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

func bundleUsage() {
	fmt.Fprintln(os.Stderr, "Usage: echo-elm bundle <subcommand> [flags]")
	fmt.Fprintln(os.Stderr, "  extract     Write a bundle's CQL and ELM out to files")
	fmt.Fprintln(os.Stderr, "  translate   Translate a bundle's CQL, optionally verifying or rewriting its ELM")
}

// runBundle dispatches the `bundle` subcommands.
func runBundle(args []string) {
	if len(args) == 0 {
		bundleUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "extract":
		runBundleExtract(args[1:])
	case "translate":
		runBundleTranslate(args[1:])
	default:
		bundleUsage()
		os.Exit(2)
	}
}

// loadBundle opens and parses the bundle at path, exiting on failure.
func loadBundle(path string) *bundle.Bundle {
	b, err := readBundle(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return b
}

// readBundle is separated from loadBundle so the file handle is closed before
// any call to os.Exit.
func readBundle(path string) (*bundle.Bundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return bundle.Load(f)
}

// ---------------------------------------------------------------------------
// bundle extract
// ---------------------------------------------------------------------------

func runBundleExtract(args []string) {
	fs := flag.NewFlagSet("bundle extract", flag.ContinueOnError)
	var (
		input   string
		outDir  string
		content string
		library string
	)
	fs.StringVar(&input, "input", "", "FHIR Bundle JSON file (required)")
	fs.StringVar(&outDir, "out-dir", "", "Directory to write extracted files into (required)")
	fs.StringVar(&content, "content", "both", "Which attachments to extract: cql|elm|both")
	fs.StringVar(&library, "library", "", "Extract only this library (default: all)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if input == "" || outDir == "" {
		fmt.Fprintln(os.Stderr, "error: --input and --out-dir are required")
		fs.Usage()
		os.Exit(2)
	}
	wantCQL := content == "cql" || content == "both"
	wantELM := content == "elm" || content == "both"
	if !wantCQL && !wantELM {
		fmt.Fprintf(os.Stderr, "error: --content must be cql, elm, or both (got %q)\n", content)
		os.Exit(2)
	}

	b := loadBundle(input)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	written := 0
	for _, lib := range b.Libraries() {
		if library != "" && lib.Name != library {
			continue
		}
		base := strings.TrimSuffix(lib.FileName(), ".cql")
		if wantCQL && len(lib.CQLSource) > 0 {
			path := filepath.Join(outDir, base+".cql")
			if err := os.WriteFile(path, lib.CQLSource, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			written++
		}
		if wantELM && len(lib.ReferenceELM) > 0 {
			path := filepath.Join(outDir, base+".json")
			if err := os.WriteFile(path, lib.ReferenceELM, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			written++
		}
	}
	fmt.Fprintf(os.Stderr, "extracted %d file(s) to %s\n", written, outDir)
}

// ---------------------------------------------------------------------------
// bundle translate
// ---------------------------------------------------------------------------

func runBundleTranslate(args []string) {
	fs := flag.NewFlagSet("bundle translate", flag.ContinueOnError)
	var (
		input   string
		output  string
		library string
		format  string
		verify  bool
	)
	fs.StringVar(&input, "input", "", "FHIR Bundle JSON file (required)")
	fs.StringVar(&output, "output", "", "Write an updated bundle with the generated ELM to this path")
	fs.StringVar(&library, "library", "", "Translate only this library (default: all)")
	fs.StringVar(&format, "format", "JSON", "ELM output format: JSON or XML")
	fs.BoolVar(&verify, "verify", false, "Compare generated ELM against the bundled ELM and report differences")
	tf := registerTranslatorFlags(fs, true)

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		fs.Usage()
		os.Exit(2)
	}
	asXML := strings.EqualFold(format, "XML")
	if !asXML && !strings.EqualFold(format, "JSON") {
		fmt.Fprintf(os.Stderr, "error: --format must be JSON or XML (got %q)\n", format)
		os.Exit(2)
	}
	if asXML && verify {
		// The bundled reference attachment is application/elm+json; there is
		// nothing comparable to check XML output against.
		fmt.Fprintln(os.Stderr, "error: --verify compares against the bundled elm+json and cannot be combined with --format XML")
		os.Exit(2)
	}

	b := loadBundle(input)
	libSrc := b.LibrarySource()
	opts := tf.options()

	var translated, differed, failed int
	var verifyFailures []string

	for _, lib := range b.Libraries() {
		if library != "" && lib.Name != library {
			continue
		}
		result, err := echoelm.Translate(lib.CQLSource, lib.FileName(),
			echoelm.WithOptions(opts),
			echoelm.WithLibrarySource(libSrc),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗  %-40s translate error: %v\n", lib.Name, err)
			failed++
			continue
		}
		var elmBytes []byte
		if asXML {
			elmBytes, err = result.XMLBytes()
		} else {
			elmBytes, err = json.MarshalIndent(map[string]any{"library": result.Library}, "", "  ")
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗  %-40s marshal error: %v\n", lib.Name, err)
			failed++
			continue
		}
		translated++

		if verify {
			switch {
			case len(lib.ReferenceELM) == 0:
				fmt.Fprintf(os.Stderr, "–  %-40s no bundled ELM to verify against\n", lib.Name)
			case parity.NormalizeForGolden(string(lib.ReferenceELM)) == parity.NormalizeForGolden(string(elmBytes)):
				fmt.Fprintf(os.Stderr, "✓  %-40s match\n", lib.Name)
			default:
				fmt.Fprintf(os.Stderr, "✗  %-40s differs\n", lib.Name)
				differed++
				verifyFailures = append(verifyFailures, lib.Name)
			}
		}

		if output != "" {
			contentType := bundle.ContentTypeELMJSON
			if asXML {
				contentType = bundle.ContentTypeELMXML
			}
			if err := b.SetContent(lib.Name, lib.Version, contentType, elmBytes); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	if output != "" {
		out, err := b.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(output, out, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot write %s: %v\n", output, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "bundle written to: %s\n", output)
	}

	fmt.Fprintf(os.Stderr, "\ntranslated=%d", translated)
	if verify {
		fmt.Fprintf(os.Stderr, "  differ=%d", differed)
	}
	fmt.Fprintf(os.Stderr, "  error=%d\n", failed)

	if failed > 0 || differed > 0 {
		sort.Strings(verifyFailures)
		for _, name := range verifyFailures {
			fmt.Fprintf(os.Stderr, "  differs: %s\n", name)
		}
		os.Exit(1)
	}
}
