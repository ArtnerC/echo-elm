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

// newMCPClient starts echo-elm mcp as a subprocess and returns a connected client session.
func newMCPClient(t *testing.T, workspace string) *mcp.ClientSession {
	t.Helper()

	// Build the binary if not already done (use go run for simplicity in tests).
	binary, err := filepath.Abs(filepath.Join("..", "..", "echo-elm.exe"))
	if err != nil {
		t.Fatalf("resolve binary: %v", err)
	}

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

func TestMCPListParityRuns(t *testing.T) {
	ws := fixtureWorkspace(t)
	session := newMCPClient(t, ws)

	result := callTool(t, session, "list_parity_runs", nil)

	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	var out struct {
		Runs  []any `json:"runs"`
		Total int   `json:"total"`
	}
	decodeText(t, result, &out)
	// Fixture has no parity runs; just verify shape is correct
	if out.Runs == nil {
		t.Error("expected runs to be an array (even if empty)")
	}
}
