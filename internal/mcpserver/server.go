// Package mcpserver implements the echo-elm MCP server over stdio.
// It exposes translation, library browsing, and parity tools as MCP tools
// so AI agents can drive the translator directly.
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
	"path/filepath"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

func registerTools(srv *mcp.Server, opts Options) {
	// translate_cql — translate inline CQL
	type translateCQLIn struct {
		Content string `json:"content" jsonschema:"CQL source text to translate"`
		Format  string `json:"format,omitempty" jsonschema:"Output format: xml, json, or both (default: both)"`
		CQFMode bool   `json:"cqfMode,omitempty" jsonschema:"Enable cqframework-compatible output"`
	}
	type translateOut struct {
		Diagnostics []diagOut `json:"diagnostics"`
		ElmXML      string    `json:"elmXml,omitempty"`
		ElmJSON     string    `json:"elmJson,omitempty"`
		ParsedName  string    `json:"parsedName,omitempty"`
		HasErrors   bool      `json:"hasErrors"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "translate_cql",
		Description: "Translate a CQL snippet to ELM. Returns diagnostics and ELM in XML and/or JSON.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in translateCQLIn) (*mcp.CallToolResult, translateOut, error) {
		if in.Content == "" {
			return nil, translateOut{}, errors.New("content is required")
		}
		if in.Format == "" {
			in.Format = "both"
		}
		result, err := echoelm.Translate([]byte(in.Content), "input.cql",
			echoelm.WithCQFMode(in.CQFMode))
		if err != nil {
			return nil, translateOut{}, err
		}
		out := translateOut{
			Diagnostics: toDiagOuts(result.Diagnostics),
			HasErrors:   hasErrors(result.Diagnostics),
		}
		if result.Library != nil && result.Library.Identifier != nil {
			out.ParsedName = result.Library.Identifier.ID
		}
		if in.Format == "xml" || in.Format == "both" {
			if xmlBytes, e := result.MarshalXML(); e == nil {
				out.ElmXML = string(xmlBytes)
			}
		}
		if in.Format == "json" || in.Format == "both" {
			if jsonBytes, e := json.Marshal(result); e == nil {
				out.ElmJSON = string(jsonBytes)
			}
		}
		return toolResult(out)
	})

	// translate_file — translate a .cql file under workdir
	type translateFileIn struct {
		Path    string `json:"path" jsonschema:"Workspace-relative path to a .cql file"`
		Format  string `json:"format,omitempty" jsonschema:"Output format: xml, json, or both (default: both)"`
		CQFMode bool   `json:"cqfMode,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "translate_file",
		Description: "Translate a .cql file under the workspace directory to ELM. Does not write output files.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in translateFileIn) (*mcp.CallToolResult, translateOut, error) {
		abs, err := safePath(opts.Workdir, in.Path)
		if err != nil {
			return nil, translateOut{}, err
		}
		src, err := os.ReadFile(abs)
		if err != nil {
			return nil, translateOut{}, fmt.Errorf("read %s: %w", in.Path, err)
		}
		if in.Format == "" {
			in.Format = "both"
		}
		result, err := echoelm.Translate(src, filepath.Base(abs), echoelm.WithCQFMode(in.CQFMode))
		if err != nil {
			return nil, translateOut{}, err
		}
		out := translateOut{
			Diagnostics: toDiagOuts(result.Diagnostics),
			HasErrors:   hasErrors(result.Diagnostics),
		}
		if result.Library != nil && result.Library.Identifier != nil {
			out.ParsedName = result.Library.Identifier.ID
		}
		if in.Format == "xml" || in.Format == "both" {
			if xmlBytes, e := result.MarshalXML(); e == nil {
				out.ElmXML = string(xmlBytes)
			}
		}
		if in.Format == "json" || in.Format == "both" {
			if jsonBytes, e := json.Marshal(result); e == nil {
				out.ElmJSON = string(jsonBytes)
			}
		}
		return toolResult(out)
	})

	// list_libraries — walk workspace and return parsed library headers
	type listLibsOut struct {
		Libraries []libraryHeader `json:"libraries"`
		Total     int             `json:"total"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_libraries",
		Description: "Walk the workspace and return parsed CQL library headers (name, version, includes).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, listLibsOut, error) {
		files, _ := walkCQLFiles(opts.Workdir)
		var libs []libraryHeader
		for _, rel := range files {
			abs, err := safePath(opts.Workdir, rel)
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
		Path string `json:"path" jsonschema:"Workspace-relative path to a .cql file"`
	}
	type readLibOut struct {
		libraryHeader
		Content string `json:"content"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "read_library",
		Description: "Read a .cql file from the workspace. Returns content and parsed header.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in readLibIn) (*mcp.CallToolResult, readLibOut, error) {
		abs, err := safePath(opts.Workdir, in.Path)
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

	// list_parity_runs — list parity/runs/* newest-first
	type parityRunSummary struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	type listParityOut struct {
		Runs  []parityRunSummary `json:"runs"`
		Total int                `json:"total"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_parity_runs",
		Description: "List parity test runs under workspace/parity/runs/, newest first.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, listParityOut, error) {
		runsDir := filepath.Join(opts.Workdir, "parity", "runs")
		entries, err := os.ReadDir(runsDir)
		if errors.Is(err, os.ErrNotExist) {
			return toolResult(listParityOut{Runs: []parityRunSummary{}, Total: 0})
		}
		if err != nil {
			return nil, listParityOut{}, err
		}
		var runs []parityRunSummary
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			runs = append(runs, parityRunSummary{
				ID:   e.Name(),
				Path: filepath.Join("parity", "runs", e.Name()),
			})
		}
		sort.Slice(runs, func(i, j int) bool { return runs[i].ID > runs[j].ID })
		if runs == nil {
			runs = []parityRunSummary{}
		}
		return toolResult(listParityOut{Runs: runs, Total: len(runs)})
	})

	// get_parity_report — fetch report for a specific run
	type getParityIn struct {
		ID string `json:"id" jsonschema:"Parity run ID (directory name under parity/runs/)"`
	}
	type getParityOut struct {
		ID         string `json:"id"`
		ReportMD   string `json:"reportMd,omitempty"`
		ReportJSON string `json:"reportJson,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_parity_report",
		Description: "Fetch the markdown and JSON summary for a parity run.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in getParityIn) (*mcp.CallToolResult, getParityOut, error) {
		if strings.ContainsAny(in.ID, `/\`) {
			return nil, getParityOut{}, errors.New("invalid run ID")
		}
		base := filepath.Join(opts.Workdir, "parity", "runs", in.ID)
		out := getParityOut{ID: in.ID}
		if md, err := os.ReadFile(filepath.Join(base, "report.md")); err == nil {
			out.ReportMD = string(md)
		}
		if js, err := os.ReadFile(filepath.Join(base, "report.json")); err == nil {
			out.ReportJSON = string(js)
		}
		if out.ReportMD == "" && out.ReportJSON == "" {
			return nil, getParityOut{}, fmt.Errorf("parity run %q not found", in.ID)
		}
		return toolResult(out)
	})
}

// -----------------------------------------------------------------------
// Shared helpers
// -----------------------------------------------------------------------

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

// safePath validates and resolves a workspace-relative path.
func safePath(workdir, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", errors.New("absolute path not allowed")
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
