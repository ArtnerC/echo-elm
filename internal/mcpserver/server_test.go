package mcpserver_test

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fixtureWorkspace returns the absolute path to the test fixture workspace.
func fixtureWorkspace(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "workspace")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	return abs
}

// echoElmBinary returns the path to the echo-elm binary, platform-aware.
func echoElmBinary(t *testing.T) string {
	t.Helper()
	name := "echo-elm"
	if runtime.GOOS == "windows" {
		name = "echo-elm.exe"
	}
	abs, err := filepath.Abs(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatalf("resolve binary path: %v", err)
	}
	return abs
}

// newMCPClient starts echo-elm mcp as a subprocess and returns a connected client session.
func newMCPClient(t *testing.T, workspace string) *mcp.ClientSession {
	t.Helper()

	binary := echoElmBinary(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "echo-elm-test",
		Version: "0.0.0-test",
	}, nil)

	transport := &mcp.CommandTransport{
		Command: exec.Command(binary, "mcp", "--workdir", workspace),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect to mcp server: %v", err)
	}
	t.Cleanup(func() { session.Close() })

	return session
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func decodeText(t *testing.T, result *mcp.CallToolResult, v any) {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("no content in result")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), v); err != nil {
		t.Fatalf("unmarshal result: %v\ntext: %s", err, text.Text)
	}
}

// -----------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------

func TestMCPTranslateCQL(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "translate_cql", map[string]any{
		"content": "library TestLib version '1.0.0'",
		"format":  "both",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		Diagnostics []any  `json:"diagnostics"`
		ElmXML      string `json:"elmXml"`
		ElmJSON     string `json:"elmJson"`
		HasErrors   bool   `json:"hasErrors"`
	}
	decodeText(t, result, &out)

	if out.ElmXML == "" {
		t.Error("expected non-empty elmXml")
	}
	if out.ElmJSON == "" {
		t.Error("expected non-empty elmJson")
	}
	if out.HasErrors {
		t.Errorf("unexpected errors: %+v", out.Diagnostics)
	}
}

func TestMCPTranslateCQLSyntaxError(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "translate_cql", map[string]any{
		"content": "library Bad\n define @@@ invalid",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		Diagnostics []struct{ Severity string } `json:"diagnostics"`
		HasErrors   bool                        `json:"hasErrors"`
	}
	decodeText(t, result, &out)
	if len(out.Diagnostics) == 0 {
		t.Error("expected diagnostics for syntax error input")
	}
}

func TestMCPTranslateFile(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "translate_file", map[string]any{
		"path":   "libs/Minimal-1.0.0.cql",
		"format": "json",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		ElmJSON    string `json:"elmJson"`
		HasErrors  bool   `json:"hasErrors"`
		ParsedName string `json:"parsedName"`
	}
	decodeText(t, result, &out)
	if out.ElmJSON == "" {
		t.Error("expected non-empty elmJson")
	}
	if out.ParsedName != "Minimal" {
		t.Errorf("expected parsedName=Minimal, got %q", out.ParsedName)
	}
}

func TestMCPListLibraries(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "list_libraries", nil)

	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		Libraries []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"libraries"`
		Total int `json:"total"`
	}
	decodeText(t, result, &out)
	if out.Total < 1 {
		t.Errorf("expected at least 1 library, got %d", out.Total)
	}
	var found bool
	for _, lib := range out.Libraries {
		if lib.Name == "Minimal" && lib.Version == "1.0.0" {
			found = true
		}
	}
	if !found {
		t.Errorf("Minimal 1.0.0 not found in: %+v", out.Libraries)
	}
}

func TestMCPReadLibrary(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "read_library", map[string]any{
		"path": "libs/Minimal-1.0.0.cql",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Content string `json:"content"`
	}
	decodeText(t, result, &out)
	if out.Name != "Minimal" {
		t.Errorf("want name=Minimal, got %q", out.Name)
	}
	if out.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestMCPGetTranslatorOptions(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "get_translator_options", nil)
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		Options []struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"options"`
	}
	decodeText(t, result, &out)
	if len(out.Options) < 5 {
		t.Errorf("expected at least 5 options, got %d", len(out.Options))
	}
	var found bool
	for _, o := range out.Options {
		if o.Name == "signatureLevel" {
			found = true
		}
	}
	if !found {
		t.Error("expected signatureLevel option to be present")
	}
}

func TestMCPValidateELM(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	// valid minimal ELM XML
	validELM := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<library xmlns="urn:hl7-org:elm:r1"><identifier id="T" version="1.0.0"/></library>`
	result := callTool(t, session, "validate_elm", map[string]any{
		"content": validELM,
		"format":  "xml",
	})
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}
	var out struct {
		Valid  bool     `json:"valid"`
		Issues []string `json:"issues,omitempty"`
	}
	decodeText(t, result, &out)
	if !out.Valid {
		t.Errorf("expected valid=true, issues: %v", out.Issues)
	}

	// invalid: wrong root element
	result2 := callTool(t, session, "validate_elm", map[string]any{
		"content": `<notlibrary xmlns="urn:hl7-org:elm:r1"/>`,
		"format":  "xml",
	})
	if result2.IsError {
		t.Fatalf("tool returned error: %+v", result2.Content)
	}
	var out2 struct {
		Valid bool `json:"valid"`
	}
	decodeText(t, result2, &out2)
	if out2.Valid {
		t.Error("expected invalid result for wrong root element")
	}
}

func TestMCPTranslateCQLWithOptions(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "translate_cql", map[string]any{
		"content": "library OptsLib version '1.0.0'",
		"format":  "xml",
		"options": map[string]any{
			"annotations":    true,
			"locators":       true,
			"signatureLevel": "All",
		},
	})
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}
	var out struct {
		ElmXML    string `json:"elmXml"`
		HasErrors bool   `json:"hasErrors"`
	}
	decodeText(t, result, &out)
	if out.ElmXML == "" {
		t.Error("expected non-empty elmXml")
	}
}

// TestMCPCompareWithCQF tests the compare_with_cqf tool.
// No external CQF is required — the fallback (echo-elm CQF-mode) is used.
func TestMCPCompareWithCQF(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "compare_with_cqf", map[string]any{
		"content": "library CompareTest version '1.0.0'",
	})
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}
	var out struct {
		Status      string `json:"status"`
		EchoElmJSON string `json:"echoElmJson"`
		CqfMode     string `json:"cqfMode"`
		Shareable   string `json:"shareable"`
	}
	decodeText(t, result, &out)
	if out.Status == "echo-error" || out.Status == "cqf-error" {
		t.Errorf("unexpected error status %q", out.Status)
	}
	if out.EchoElmJSON == "" {
		t.Error("expected non-empty echoElmJson")
	}
	if out.Shareable == "" {
		t.Error("expected non-empty shareable block")
	}
	if out.CqfMode != "echo-elm-cqf-mode" {
		t.Errorf("expected cqfMode=echo-elm-cqf-mode, got %q", out.CqfMode)
	}
}

// TestMCPCompareWithCQFPreSupplied tests the cqfElm pre-supplied path.
func TestMCPCompareWithCQFPreSupplied(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	// Get ELM from translate to use as the "CQF side".
	r1 := callTool(t, session, "translate_cql", map[string]any{
		"content": "library PreSupplied version '1.0.0'",
		"format":  "json",
	})
	if r1.IsError {
		t.Fatalf("translate failed: %+v", r1.Content)
	}
	var tr struct{ ElmJSON string `json:"elmJson"` }
	decodeText(t, r1, &tr)

	r2 := callTool(t, session, "compare_with_cqf", map[string]any{
		"content": "library PreSupplied version '1.0.0'",
		"cqfElm":  tr.ElmJSON,
	})
	if r2.IsError {
		t.Fatalf("compare failed: %+v", r2.Content)
	}
	var out struct {
		Status  string `json:"status"`
		CqfMode string `json:"cqfMode"`
	}
	decodeText(t, r2, &out)
	if out.CqfMode != "pre-supplied" {
		t.Errorf("expected cqfMode=pre-supplied, got %q", out.CqfMode)
	}
	if out.Status == "echo-error" || out.Status == "cqf-error" {
		t.Errorf("unexpected error status %q", out.Status)
	}
}
