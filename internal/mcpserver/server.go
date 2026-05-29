// Package mcpserver implements the echo-elm MCP server over stdio.
// It exposes translation and library tools as MCP tools so AI agents
// can drive the CQL→ELM translator directly.
//
// Usage: echo-elm mcp [--workdir <path>] [--allow-write]
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	intelm "github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/internal/parser"
	"github.com/artnerc/echo-elm/internal/translator"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

// Options configures the MCP server.
type Options struct {
	Workdir    string // absolute workspace directory (required)
	Version    string
	AllowWrite bool // gate write-capable tools
}

// Run starts the MCP server on stdio. It blocks until the context is cancelled
// or stdio is closed.
func Run(ctx context.Context, opts Options) error {
	if opts.Workdir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("mcp: getwd: %w", err)
		}
		opts.Workdir = wd
	}
	abs, err := filepath.Abs(opts.Workdir)
	if err != nil {
		return fmt.Errorf("mcp: resolve workdir %q: %w", opts.Workdir, err)
	}
	opts.Workdir = abs

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "echo-elm",
		Version: opts.Version,
	}, nil)

	registerTools(srv, opts)

	return srv.Run(ctx, &mcp.StdioTransport{})
}

// -----------------------------------------------------------------------
// Tool registration
// -----------------------------------------------------------------------

// translatorOptions carries the full set of translator knobs an MCP caller
// can override. All fields are optional; absent fields use the default.
type translatorOptions struct {
	Annotations             *bool  `json:"annotations,omitempty"`
	Locators                *bool  `json:"locators,omitempty"`
	SignatureLevel          string `json:"signatureLevel,omitempty"`     // None|Differing|Overloads|All
	CQFMode                 bool   `json:"cqfMode,omitempty"`            // cqframework-compat output
	CompatibilityLevel      string `json:"compatibilityLevel,omitempty"` // 1.3|1.4|1.5
	ValidateUnits           *bool  `json:"validateUnits,omitempty"`
	DisableListDemotion     bool   `json:"disableListDemotion,omitempty"`
	DisableListPromotion    bool   `json:"disableListPromotion,omitempty"`
	DisableListTraversal    bool   `json:"disableListTraversal,omitempty"`
	DisableMethodInvocation bool   `json:"disableMethodInvocation,omitempty"`
	RequireFromKeyword      bool   `json:"requireFromKeyword,omitempty"`
}

func applyTranslatorOptions(o *translator.Options, in translatorOptions) {
	if in.Annotations != nil {
		o.EnableAnnotations = *in.Annotations
	}
	if in.Locators != nil {
		o.EnableLocators = *in.Locators
	}
	if in.SignatureLevel != "" {
		o.SignatureLevel = in.SignatureLevel
	}
	if in.CQFMode {
		o.CQFMode = true
	}
	if in.CompatibilityLevel != "" {
		o.CompatibilityLevel = in.CompatibilityLevel
	}
	if in.ValidateUnits != nil {
		o.ValidateUnits = *in.ValidateUnits
	}
	if in.DisableListDemotion {
		o.DisableListDemotion = true
	}
	if in.DisableListPromotion {
		o.DisableListPromotion = true
	}
	if in.DisableListTraversal {
		o.DisableListTraversal = true
	}
	if in.DisableMethodInvocation {
		o.DisableMethodInvocation = true
	}
	if in.RequireFromKeyword {
		o.RequireFromKeyword = true
	}
}

func registerTools(srv *mcp.Server, opts Options) {
	type translateOut struct {
		Diagnostics []diagOut `json:"diagnostics"`
		ElmXML      string    `json:"elmXml,omitempty"`
		ElmJSON     string    `json:"elmJson,omitempty"`
		ParsedName  string    `json:"parsedName,omitempty"`
		HasErrors   bool      `json:"hasErrors"`
	}

	doTranslate := func(src []byte, sourceName string, format string, tOpts translatorOptions) (translateOut, error) {
		if format == "" {
			format = "both"
		}
		baseOpts := translator.DefaultOptions()
		applyTranslatorOptions(&baseOpts, tOpts)

		result, err := echoelm.Translate(src, sourceName,
			echoelm.WithOptions(baseOpts),
			echoelm.WithCQFMode(tOpts.CQFMode),
		)
		if err != nil {
			return translateOut{}, err
		}
		out := translateOut{
			Diagnostics: toDiagOuts(result.Diagnostics),
			HasErrors:   hasErrors(result.Diagnostics),
		}
		if result.Library != nil && result.Library.Identifier.ID != "" {
			out.ParsedName = result.Library.Identifier.ID
		}
		if format == "xml" || format == "both" {
			if xmlBytes, e := result.XMLBytes(); e == nil {
				out.ElmXML = string(xmlBytes)
			}
		}
		if format == "json" || format == "both" {
			if jsonBytes, e := json.Marshal(result); e == nil {
				out.ElmJSON = string(jsonBytes)
			}
		}
		return out, nil
	}

	// translate_cql — translate inline CQL source
	type translateCQLIn struct {
		Content string            `json:"content"`
		Format  string            `json:"format,omitempty"`
		Options translatorOptions `json:"options,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "translate_cql",
		Description: "Translate a CQL source string to ELM. " +
			"format: xml|json|both (default: both). " +
			"options overrides translator defaults (annotations, locators, signatureLevel, etc.).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in translateCQLIn) (*mcp.CallToolResult, translateOut, error) {
		if in.Content == "" {
			return nil, translateOut{}, errors.New("content is required")
		}
		out, err := doTranslate([]byte(in.Content), "input.cql", in.Format, in.Options)
		if err != nil {
			return nil, translateOut{}, err
		}
		return toolResult(out)
	})

	// translate_file — translate a .cql file
	// Accepts workspace-relative paths when workdir is set, or absolute paths otherwise.
	type translateFileIn struct {
		Path    string            `json:"path"`
		Format  string            `json:"format,omitempty"`
		Options translatorOptions `json:"options,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "translate_file",
		Description: "Translate a CQL file to ELM. " +
			"path: workspace-relative (or absolute if no workspace is configured). " +
			"Returns ELM without writing output files.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in translateFileIn) (*mcp.CallToolResult, translateOut, error) {
		abs, err := resolvePath(opts.Workdir, in.Path)
		if err != nil {
			return nil, translateOut{}, err
		}
		src, err := os.ReadFile(abs)
		if err != nil {
			return nil, translateOut{}, fmt.Errorf("read %s: %w", in.Path, err)
		}
		out, err := doTranslate(src, filepath.Base(abs), in.Format, in.Options)
		if err != nil {
			return nil, translateOut{}, err
		}
		return toolResult(out)
	})

	// validate_elm — structural ELM validation
	type validateIn struct {
		Content string `json:"content"`
		Format  string `json:"format"`
	}
	type validateOut struct {
		Valid  bool     `json:"valid"`
		Issues []string `json:"issues,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "validate_elm",
		Description: "Structurally validate a serialized ELM document. " +
			"format: xml or json. " +
			"Checks well-formedness, root element, and ELM namespace/envelope.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in validateIn) (*mcp.CallToolResult, validateOut, error) {
		if in.Content == "" {
			return nil, validateOut{}, errors.New("content is required")
		}
		err := intelm.Validate([]byte(in.Content), in.Format)
		if err == nil {
			return toolResult(validateOut{Valid: true})
		}
		var ve *intelm.ValidationError
		if errors.As(err, &ve) {
			return toolResult(validateOut{Valid: false, Issues: ve.Issues})
		}
		return toolResult(validateOut{Valid: false, Issues: []string{err.Error()}})
	})

	// get_translator_options — return available options and their defaults
	type optionDesc struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Default     any    `json:"default"`
		Description string `json:"description"`
	}
	type getOptsOut struct {
		Options []optionDesc `json:"options"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_translator_options",
		Description: "Return all available translator options with their types, defaults, and descriptions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, getOptsOut, error) {
		d := translator.DefaultOptions()
		out := getOptsOut{Options: []optionDesc{
			{Name: "annotations", Type: "bool", Default: d.EnableAnnotations, Description: "Emit ELM annotations (EnableAnnotations)"},
			{Name: "locators", Type: "bool", Default: d.EnableLocators, Description: "Emit source locators (EnableLocators)"},
			{Name: "signatureLevel", Type: "string", Default: d.SignatureLevel, Description: "Signature level: None|Differing|Overloads|All"},
			{Name: "cqfMode", Type: "bool", Default: false, Description: "cqframework-compatible output (empty annotation arrays, no translatorOptions header)"},
			{Name: "compatibilityLevel", Type: "string", Default: d.CompatibilityLevel, Description: "CQL compatibility level: 1.3|1.4|1.5"},
			{Name: "validateUnits", Type: "bool", Default: d.ValidateUnits, Description: "Emit Warning diagnostics for unrecognized UCUM units"},
			{Name: "disableListDemotion", Type: "bool", Default: d.DisableListDemotion, Description: "Disable implicit list demotion (DisableListDemotion)"},
			{Name: "disableListPromotion", Type: "bool", Default: d.DisableListPromotion, Description: "Disable implicit list promotion (DisableListPromotion)"},
			{Name: "disableListTraversal", Type: "bool", Default: d.DisableListTraversal, Description: "Disable implicit list traversal (DisableListTraversal)"},
			{Name: "disableMethodInvocation", Type: "bool", Default: d.DisableMethodInvocation, Description: "Disable method-style invocation (DisableMethodInvocation)"},
			{Name: "requireFromKeyword", Type: "bool", Default: d.RequireFromKeyword, Description: "Require 'from' keyword in queries (RequireFromKeyword)"},
		}}
		return toolResult(out)
	})

	// list_libraries — walk workspace and return parsed library headers
	type listLibsOut struct {
		Libraries []libraryHeader `json:"libraries"`
		Total     int             `json:"total"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_libraries",
		Description: "Walk the workspace directory and return parsed CQL library headers (name, version, includes).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, listLibsOut, error) {
		files, _ := walkCQLFiles(opts.Workdir)
		var libs []libraryHeader
		for _, rel := range files {
			abs, err := resolvePath(opts.Workdir, rel)
			if err != nil {
				continue
			}
			src, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			libs = append(libs, parseLibraryHeader(src, rel))
		}
		if libs == nil {
			libs = []libraryHeader{}
		}
		return toolResult(listLibsOut{Libraries: libs, Total: len(libs)})
	})

	// read_library — read a .cql file's content and parsed header
	type readLibIn struct {
		Path string `json:"path"`
	}
	type readLibOut struct {
		libraryHeader
		Content string `json:"content"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "read_library",
		Description: "Read a CQL file from the workspace. Returns the raw source and parsed header (name, version, includes).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in readLibIn) (*mcp.CallToolResult, readLibOut, error) {
		abs, err := resolvePath(opts.Workdir, in.Path)
		if err != nil {
			return nil, readLibOut{}, err
		}
		src, err := os.ReadFile(abs)
		if errors.Is(err, os.ErrNotExist) {
			return nil, readLibOut{}, fmt.Errorf("library not found: %s", in.Path)
		}
		if err != nil {
			return nil, readLibOut{}, err
		}
		hdr := parseLibraryHeader(src, in.Path)
		return toolResult(readLibOut{libraryHeader: hdr, Content: string(src)})
	})
	// compare_with_cqf — translate CQL with echo-elm and compare against CQF output
	type compareIn struct {
		// CQL source — provide exactly one of content or path.
		Content string `json:"content,omitempty"`
		Path    string `json:"path,omitempty"`
		// CQF side — provide one of: cqfJar, cqfScript, cqfElm, or none (fallback to CQF-mode echo-elm).
		CqfJar    string `json:"cqfJar,omitempty"`    // absolute path to cql-to-elm.jar (java must be on PATH)
		CqfScript string `json:"cqfScript,omitempty"` // absolute path to run.bat / run.sh wrapper
		CqfElm    string `json:"cqfElm,omitempty"`    // pre-computed ELM JSON from CQF (paste-in)
		// Translator options for the echo-elm side.
		Options translatorOptions `json:"options,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "compare_with_cqf",
		Description: "Translate CQL with echo-elm, optionally run against a local cqframework installation, " +
			"and compare the ELM outputs. " +
			"Supply cqfJar (path to cql-to-elm.jar, needs java on PATH), " +
			"cqfScript (path to run.bat/run.sh wrapper), " +
			"cqfElm (pre-computed ELM JSON you already ran through CQF), " +
			"or nothing (falls back to echo-elm in CQF-compatibility mode). " +
			"Returns structured diff and a shareable text block for bug reports.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in compareIn) (*mcp.CallToolResult, compareOut, error) {
		// 1. Resolve CQL source.
		var src []byte
		var sourceName = "input.cql"
		switch {
		case in.Content != "":
			src = []byte(in.Content)
		case in.Path != "":
			abs, err := resolvePath(opts.Workdir, in.Path)
			if err != nil {
				return nil, compareOut{}, err
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return nil, compareOut{}, fmt.Errorf("read %s: %w", in.Path, err)
			}
			src = data
			sourceName = filepath.Base(abs)
		default:
			return nil, compareOut{}, errors.New("provide content or path")
		}

		// 2. Run echo-elm.
		baseOpts := translator.DefaultOptions()
		applyTranslatorOptions(&baseOpts, in.Options)
		result, err := echoelm.Translate(src, sourceName,
			echoelm.WithOptions(baseOpts),
			echoelm.WithCQFMode(in.Options.CQFMode),
		)
		if err != nil {
			return toolResult(compareOut{
				Status:    "echo-error",
				Shareable: fmt.Sprintf("echo-elm translation error: %v", err),
			})
		}
		echoBytes, _ := json.MarshalIndent(map[string]interface{}{"library": result.Library}, "", "  ")
		echoNorm := parity.NormalizeForGolden(string(echoBytes))
		diags := toDiagOuts(result.Diagnostics)

		// 3. Get CQF-side ELM.
		var cqfRaw string
		var cqfMode string
		switch {
		case in.CqfElm != "":
			cqfRaw = in.CqfElm
			cqfMode = "pre-supplied"
		case in.CqfJar != "" || in.CqfScript != "":
			// Write CQL to a temp file so cqframework can read it.
			tmpDir, err := os.MkdirTemp("", "echo-elm-compare-*")
			if err != nil {
				return nil, compareOut{}, fmt.Errorf("tempdir: %w", err)
			}
			defer os.RemoveAll(tmpDir)
			tmpCQL := filepath.Join(tmpDir, sourceName)
			if err := os.WriteFile(tmpCQL, src, 0o644); err != nil {
				return nil, compareOut{}, fmt.Errorf("write temp cql: %w", err)
			}
			var launcher string
			if in.CqfJar != "" {
				launcher = "jar:" + in.CqfJar
				cqfMode = "jar:" + filepath.Base(in.CqfJar)
			} else {
				launcher = in.CqfScript
				cqfMode = "script:" + filepath.Base(in.CqfScript)
			}
			raw, stderr, runErr := runCQF(launcher, tmpCQL)
			if runErr != nil {
				out := buildCompareOut("cqf-error", echoNorm, "", cqfMode, diags,
					fmt.Sprintf("CQF error: %v\nstderr: %s", runErr, stderr), in.Content+in.Path, sourceName)
				return toolResult(out)
			}
			cqfRaw = raw
		default:
			// Fallback: translate with echo-elm in CQF-compatibility mode.
			cqfResult, cqfErr := echoelm.Translate(src, sourceName, echoelm.WithCQFOptions())
			if cqfErr != nil {
				return toolResult(compareOut{
					Status:      "cqf-unavailable",
					EchoElmJSON: echoNorm,
					CqfMode:     "cqf-mode-fallback",
					Diagnostics: diags,
					Shareable:   "No CQF instance available. Provide cqfJar, cqfScript, or cqfElm.",
				})
			}
			cqfBytes, _ := json.MarshalIndent(map[string]interface{}{"library": cqfResult.Library}, "", "  ")
			cqfRaw = string(cqfBytes)
			cqfMode = "echo-elm-cqf-mode"
		}

		cqfNorm := parity.NormalizeForGolden(cqfRaw)
		var status string
		var diff string
		if echoNorm == cqfNorm {
			status = "match"
		} else {
			status = "differ"
			diff = parity.SimpleDiff(cqfNorm, echoNorm)
		}

		out := buildCompareOut(status, echoNorm, cqfNorm, cqfMode, diags, diff, string(src), sourceName)
		return toolResult(out)
	})
}

// buildCompareOut assembles the compareOut struct including the shareable text block.
func buildCompareOut(status, echoNorm, cqfNorm, cqfMode string, diags []diagOut, diff, cqlSrc, sourceName string) compareOut {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== echo-elm CQF parity report ===\n")
	fmt.Fprintf(&sb, "File:   %s\n", sourceName)
	fmt.Fprintf(&sb, "Status: %s\n", status)
	fmt.Fprintf(&sb, "CQF:    %s\n", cqfMode)
	if len(diags) > 0 {
		fmt.Fprintf(&sb, "\nDiagnostics:\n")
		for _, d := range diags {
			fmt.Fprintf(&sb, "  [%s] %s\n", d.Severity, d.Message)
		}
	}
	if diff != "" {
		fmt.Fprintf(&sb, "\nDiff (- CQF, + echo-elm):\n```diff\n%s```\n", diff)
	}
	if cqlSrc != "" && len(cqlSrc) < 4000 {
		fmt.Fprintf(&sb, "\nCQL source:\n```cql\n%s\n```\n", strings.TrimSpace(cqlSrc))
	}
	return compareOut{
		Status:      status,
		EchoElmJSON: echoNorm,
		CqfJSON:     cqfNorm,
		CqfMode:     cqfMode,
		Diff:        diff,
		Diagnostics: diags,
		Shareable:   sb.String(),
	}
}

// runCQF invokes the cqframework tool (JAR or script) on the given .cql file.
// launcher prefix "jar:<path>" invokes java -jar; otherwise treats it as a script.
func runCQF(launcher, cqlPath string) (jsonOut, stderr string, err error) {
	dir := filepath.Dir(cqlPath)
	base := strings.TrimSuffix(filepath.Base(cqlPath), ".cql")
	outJSON := filepath.Join(dir, base+".json")
	os.Remove(outJSON)

	var cmd *exec.Cmd
	if strings.HasPrefix(launcher, "jar:") {
		jar := strings.TrimPrefix(launcher, "jar:")
		cmd = exec.Command("java", "-jar", jar, "--input", cqlPath, "--format", "JSON")
	} else {
		if isWindowsPath(launcher) {
			cmd = exec.Command("cmd", "/C", launcher, "--input", cqlPath, "--format", "JSON")
		} else {
			cmd = exec.Command(launcher, "--input", cqlPath, "--format", "JSON")
		}
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf
	if runErr := cmd.Run(); runErr != nil {
		return "", stderrBuf.String(), fmt.Errorf("cqf exit: %w", runErr)
	}

	data, err := os.ReadFile(outJSON)
	if err != nil {
		// Some versions write to stdout instead of a file; not currently handled.
		return "", stderrBuf.String(), fmt.Errorf("read cqf output %s: %w", outJSON, err)
	}
	return string(data), stderrBuf.String(), nil
}

func isWindowsPath(p string) bool {
	return strings.HasSuffix(strings.ToLower(p), ".bat") || strings.HasSuffix(strings.ToLower(p), ".cmd")
}

type diagOut struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Locator  string `json:"locator"`
}

type libraryHeader struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Path     string   `json:"path"`
	Includes []string `json:"includes"`
}

// compareOut is the structured output of compare_with_cqf.
type compareOut struct {
	Status      string    `json:"status"`      // "match"|"differ"|"echo-error"|"cqf-unavailable"|"cqf-error"
	EchoElmJSON string    `json:"echoElmJson"` // normalized JSON from echo-elm
	CqfJSON     string    `json:"cqfJson"`     // normalized JSON from CQF side (empty if unavailable)
	CqfMode     string    `json:"cqfMode"`     // how CQF side was obtained
	Diff        string    `json:"diff,omitempty"`
	Diagnostics []diagOut `json:"diagnostics,omitempty"`
	Shareable   string    `json:"shareable"` // compact text block for bug reports
}

func toDiagOuts(diags []translator.Diagnostic) []diagOut {
	out := make([]diagOut, 0, len(diags))
	for _, d := range diags {
		out = append(out, diagOut{
			Severity: d.Severity,
			Message:  d.Message,
			Locator:  d.Locator,
		})
	}
	return out
}

func hasErrors(diags []translator.Diagnostic) bool {
	for _, d := range diags {
		if strings.EqualFold(d.Severity, "error") {
			return true
		}
	}
	return false
}

// toolResult serialises v to JSON and wraps it in a TextContent CallToolResult.
func toolResult[T any](v T) (*mcp.CallToolResult, T, error) {
	return nil, v, nil
}

// resolvePath resolves a path for file-based tools.
// If path is absolute, it is used directly (no sandbox restriction).
// If path is relative, it is resolved against workdir with escape prevention.
func resolvePath(workdir, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel), nil
	}
	clean := filepath.Clean(filepath.Join(workdir, rel))
	if clean != workdir && !strings.HasPrefix(clean, workdir+string(filepath.Separator)) {
		return "", errors.New("path escapes workspace")
	}
	return clean, nil
}

// walkCQLFiles returns workspace-relative slash-paths of all .cql files.
func walkCQLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".cql") {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files, err
}

// parseLibraryHeader extracts name, version, and includes from CQL source.
func parseLibraryHeader(src []byte, sourceName string) libraryHeader {
	h := libraryHeader{Path: sourceName, Includes: []string{}}
	pr, _ := parser.ParseBytes(src, sourceName)
	if pr == nil || pr.Library == nil {
		return h
	}
	lib := pr.Library
	if lib.Name != nil {
		h.Name = lib.Name.Name
		h.Version = lib.Name.Version
	}
	for _, inc := range lib.Includes {
		alias := inc.LocalName
		if alias == "" {
			alias = inc.Path
		}
		h.Includes = append(h.Includes, alias)
	}
	return h
}
