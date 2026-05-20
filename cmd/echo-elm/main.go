// Command echo-elm is a CQL→ELM translator.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		fmt.Fprintln(os.Stderr, "echo-elm mcp: not yet implemented")
		os.Exit(1)
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
	fmt.Fprintln(os.Stderr, "  version          Print version")
}

// runTranslate implements both `translate` and `cqf translate` subcommands.
// cqfMode applies cqframework-compatible defaults (signatureLevel=None, etc.).
func runTranslate(args []string, cqfMode bool) {
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)

	var (
		input      string
		output     string
		format     string
		annotations bool
		locators    bool
		sigLevel   string
	)

	fs.StringVar(&input, "input", "", "Input CQL file (required)")
	fs.StringVar(&output, "output", "", "Output file or directory (default: next to input)")
	fs.StringVar(&format, "format", "JSON", "Output format: JSON or XML")
	fs.BoolVar(&annotations, "annotations", true, "Emit ELM annotations")
	fs.BoolVar(&locators, "locators", true, "Emit source locators")

	if cqfMode {
		// CQF defaults match cqframework CLI behavior.
		sigLevel = "None"
		fs.StringVar(&sigLevel, "signatures", sigLevel, "Signature level: None|Differing|Overloads|All")
		// --disable-list-demotion, etc. omitted for now (add in later phase)
	} else {
		sigLevel = "Overloads"
		fs.StringVar(&sigLevel, "signatures", sigLevel, "Signature level: None|Differing|Overloads|All")
	}

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
		outBytes, err = result.MarshalXML()
	default: // JSON
		outBytes, err = json.MarshalIndent(result, "", "   ")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal: %v\n", err)
		os.Exit(1)
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

// resolveOutput determines the output file path from the input path, --output flag, and format.
func resolveOutput(input, output, format string) string {
	ext := ".json"
	if strings.ToUpper(format) == "XML" {
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

