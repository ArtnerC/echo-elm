// Phase 8 demo: start echo-elm mcp as a subprocess, connect via MCP client,
// exercise each tool, then shut down cleanly.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║       echo-elm  Phase 8  —  MCP Server Demo        ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	binary := resolveBinary()
	workspace := fixtureWorkspace()

	// Connect to echo-elm mcp via CommandTransport.
	client := mcp.NewClient(&mcp.Implementation{
		Name: "phase8-demo", Version: "0.0.0",
	}, nil)
	transport := &mcp.CommandTransport{
		Command: exec.Command(binary, "mcp", "--workdir", workspace),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	must("connect", err)
	defer func() { _ = session.Close() }()

	fmt.Printf("Connected to echo-elm MCP server\n")
	fmt.Printf("Workspace: %s\n\n", workspace)

	// ── Tool 1: translate_cql ─────────────────────────────────────────────
	fmt.Println("── Tool 1: translate_cql")
	result := callTool(ctx, session, "translate_cql", map[string]any{
		"content": "library Demo version '2.0.0'",
		"format":  "json",
	})
	var tr struct {
		ElmJSON    string `json:"elmJson"`
		HasErrors  bool   `json:"hasErrors"`
		ParsedName string `json:"parsedName"`
	}
	decode(result, &tr)
	check("elmJson non-empty", tr.ElmJSON != "")
	check("no errors", !tr.HasErrors)
	fmt.Printf("   parsedName=%s  elmJson[:60]=%s\n\n", tr.ParsedName, clip(tr.ElmJSON, 60))

	// ── Tool 2: translate_file ────────────────────────────────────────────
	fmt.Println("── Tool 2: translate_file")
	result = callTool(ctx, session, "translate_file", map[string]any{
		"path":   "libs/Minimal-1.0.0.cql",
		"format": "xml",
	})
	var tf struct {
		ElmXML     string `json:"elmXml"`
		HasErrors  bool   `json:"hasErrors"`
		ParsedName string `json:"parsedName"`
	}
	decode(result, &tf)
	check("elmXml non-empty", tf.ElmXML != "")
	check("parsedName=Minimal", tf.ParsedName == "Minimal")
	fmt.Printf("   parsedName=%s  elmXml[:60]=%s\n\n", tf.ParsedName, clip(tf.ElmXML, 60))

	// ── Tool 3: list_libraries ────────────────────────────────────────────
	fmt.Println("── Tool 3: list_libraries")
	result = callTool(ctx, session, "list_libraries", nil)
	var ll struct {
		Libraries []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"libraries"`
		Total int `json:"total"`
	}
	decode(result, &ll)
	check("total >= 1", ll.Total >= 1)
	for _, lib := range ll.Libraries {
		fmt.Printf("   library: %s %s\n", lib.Name, lib.Version)
	}
	fmt.Println()

	// ── Tool 4: read_library ──────────────────────────────────────────────
	fmt.Println("── Tool 4: read_library")
	result = callTool(ctx, session, "read_library", map[string]any{
		"path": "libs/BPMeasure-1.0.0.cql",
	})
	var rl struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Content string `json:"content"`
	}
	decode(result, &rl)
	check("name=BPMeasure", rl.Name == "BPMeasure")
	check("content non-empty", rl.Content != "")
	fmt.Printf("   %s %s — %d bytes\n\n", rl.Name, rl.Version, len(rl.Content))

	// ── Tool 5: list_parity_runs ──────────────────────────────────────────
	fmt.Println("── Tool 5: list_parity_runs")
	result = callTool(ctx, session, "list_parity_runs", nil)
	var lp struct {
		Runs  []any `json:"runs"`
		Total int   `json:"total"`
	}
	decode(result, &lp)
	check("runs is array", lp.Runs != nil)
	fmt.Printf("   total=%d (no runs in fixture workspace)\n\n", lp.Total)

	// ── Tool 6: syntax error recovery ────────────────────────────────────
	fmt.Println("── Tool 6: translate_cql (syntax error recovery)")
	result = callTool(ctx, session, "translate_cql", map[string]any{
		"content": "library Bad\n define @@@ broken",
	})
	var te struct {
		Diagnostics []struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"diagnostics"`
		HasErrors bool `json:"hasErrors"`
	}
	decode(result, &te)
	check("has diagnostics", len(te.Diagnostics) > 0)
	for _, d := range te.Diagnostics {
		fmt.Printf("   [%s] %s\n", d.Severity, clip(d.Message, 70))
	}
	fmt.Println()

	fmt.Println("Phase 8 complete ✓  MCP server — all tools functional.")
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func resolveBinary() string {
	_, file, _, _ := runtime.Caller(0)
	bin := filepath.Join(filepath.Dir(file), "..", "..", "echo-elm.exe")
	abs, _ := filepath.Abs(bin)
	return abs
}

func fixtureWorkspace() string {
	_, file, _, _ := runtime.Caller(0)
	ws := filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "workspace")
	abs, _ := filepath.Abs(ws)
	return abs
}

func callTool(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	must("CallTool "+name, err)
	if result.IsError {
		content := ""
		if len(result.Content) > 0 {
			if t, ok := result.Content[0].(*mcp.TextContent); ok {
				content = t.Text
			}
		}
		fmt.Fprintf(os.Stderr, "FATAL: tool %s returned error: %s\n", name, content)
		os.Exit(1)
	}
	return result
}

func decode(result *mcp.CallToolResult, v any) {
	if len(result.Content) == 0 {
		fmt.Fprintln(os.Stderr, "FATAL: empty result content")
		os.Exit(1)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		fmt.Fprintf(os.Stderr, "FATAL: expected TextContent, got %T\n", result.Content[0])
		os.Exit(1)
	}
	if err := json.Unmarshal([]byte(text.Text), v); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: decode result: %v\ntext: %s\n", err, text.Text)
		os.Exit(1)
	}
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

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
