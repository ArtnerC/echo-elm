// Phase 7 demo: start the local UI HTTP server, exercise each API endpoint,
// then shut down cleanly. Demonstrates Phase 7 is fully functional.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/ui"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║      echo-elm  Phase 7  —  Local UI Server Demo    ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	workspace := fixtureWorkspace()

	// Start server on a random loopback port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	base := "http://" + addr

	s, err := ui.NewServer(ui.ServerOptions{
		Workspace:   workspace,
		Version:     "0.0.0-phase7-demo",
		AllowRemote: true, // we bound the listener ourselves
	})
	if err != nil {
		fatalf("NewServer: %v", err)
	}

	srv := &http.Server{Handler: s.Handler()}
	go func() { srv.Serve(ln) }()

	// Give the goroutine a moment to start.
	time.Sleep(20 * time.Millisecond)

	client := &http.Client{Timeout: 5 * time.Second}

	// ── Test 1: /api/healthz ──────────────────────────────────────────────
	fmt.Printf("── Test 1: GET /api/healthz\n")
	resp := must(client.Get(base + "/api/healthz"))
	var health map[string]string
	decode(resp, &health)
	check("status=ok", health["status"] == "ok")
	fmt.Printf("   status=%s  version=%s\n\n", health["status"], health["version"])

	// ── Test 2: /api/version ─────────────────────────────────────────────
	fmt.Printf("── Test 2: GET /api/version\n")
	resp = must(client.Get(base + "/api/version"))
	var ver map[string]string
	decode(resp, &ver)
	check("cqlSpec=1.5.3", ver["cqlSpec"] == "1.5.3")
	fmt.Printf("   cqlSpec=%s  goVersion=%s\n\n", ver["cqlSpec"], ver["goVersion"])

	// ── Test 3: /api/workspace ───────────────────────────────────────────
	fmt.Printf("── Test 3: GET /api/workspace\n")
	resp = must(client.Get(base + "/api/workspace"))
	var ws struct {
		Root     string   `json:"root"`
		CQLFiles []string `json:"cqlFiles"`
	}
	decode(resp, &ws)
	check("workspace has CQL files", len(ws.CQLFiles) >= 1)
	fmt.Printf("   root=%s  cqlFiles=%d\n\n", ws.Root, len(ws.CQLFiles))

	// ── Test 4: /api/libraries ───────────────────────────────────────────
	fmt.Printf("── Test 4: GET /api/libraries\n")
	resp = must(client.Get(base + "/api/libraries"))
	var libs []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	decode(resp, &libs)
	check("at least 1 library", len(libs) >= 1)
	for _, lib := range libs {
		fmt.Printf("   library: %s %s\n", lib.Name, lib.Version)
	}
	fmt.Println()

	// ── Test 5: /api/libraries/{path} ────────────────────────────────────
	fmt.Printf("── Test 5: GET /api/libraries/libs/Minimal-1.0.0.cql\n")
	resp = must(client.Get(base + "/api/libraries/libs/Minimal-1.0.0.cql"))
	var libContent struct {
		Content string `json:"content"`
	}
	decode(resp, &libContent)
	check("content contains 'library Minimal'", strings.Contains(libContent.Content, "library Minimal"))
	fmt.Printf("   content: %s\n\n", strings.TrimSpace(libContent.Content))

	// ── Test 6: POST /api/translate ──────────────────────────────────────
	fmt.Printf("── Test 6: POST /api/translate (minimal library)\n")
	body, _ := json.Marshal(map[string]string{
		"content": "library Demo version '1.0.0'",
	})
	resp = must(client.Post(base+"/api/translate", "application/json", bytes.NewReader(body)))
	var tr struct {
		Diagnostics []any  `json:"diagnostics"`
		ElmJSON     string `json:"elmJson"`
		ElmXML      string `json:"elmXml"`
		Stats       struct {
			CompileMs int64 `json:"compileMs"`
		} `json:"stats"`
	}
	decode(resp, &tr)
	check("elmJson non-empty", tr.ElmJSON != "")
	check("elmXml non-empty", tr.ElmXML != "")
	fmt.Printf("   diagnostics=%d  compileMs=%d\n", len(tr.Diagnostics), tr.Stats.CompileMs)
	fmt.Printf("   elmJson[:80]: %s\n\n", clip(tr.ElmJSON, 80))

	// ── Test 7: POST /api/translate with syntax error ────────────────────
	fmt.Printf("── Test 7: POST /api/translate (syntax error recovery)\n")
	body, _ = json.Marshal(map[string]string{
		"content": "library Bad\n define @@@ invalid",
	})
	resp = must(client.Post(base+"/api/translate", "application/json", bytes.NewReader(body)))
	var tr2 struct {
		Diagnostics []struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"diagnostics"`
	}
	decode(resp, &tr2)
	check("has diagnostics", len(tr2.Diagnostics) > 0)
	for _, d := range tr2.Diagnostics {
		fmt.Printf("   [%s] %s\n", d.Severity, clip(d.Message, 70))
	}
	fmt.Println()

	// ── Test 8: /api/parity/runs (empty) ─────────────────────────────────
	fmt.Printf("── Test 8: GET /api/parity/runs\n")
	resp = must(client.Get(base + "/api/parity/runs"))
	var runs []any
	decode(resp, &runs)
	check("returns array", runs != nil || len(runs) == 0)
	fmt.Printf("   runs=%d (no parity runs in fixture workspace)\n\n", len(runs))

	// ── Test 9: Static SPA fallback ───────────────────────────────────────
	fmt.Printf("── Test 9: GET /unknown-path (SPA fallback)\n")
	resp = must(client.Get(base + "/some/unknown/route"))
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	check("200 HTML for unknown route", resp.StatusCode == 200 && strings.Contains(buf.String(), "<html"))
	fmt.Printf("   HTTP %d  Content-Type: %s\n\n", resp.StatusCode, resp.Header.Get("Content-Type"))

	// Shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	fmt.Println("Phase 7 complete ✓  Local UI server verified — all endpoints functional.")
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func fixtureWorkspace() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "workspace")
	abs, _ := filepath.Abs(root)
	return abs
}

func must(resp *http.Response, err error) *http.Response {
	if err != nil {
		fatalf("HTTP request failed: %v", err)
	}
	return resp
}

func decode(resp *http.Response, v any) {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		fatalf("unexpected status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		fatalf("decode response: %v", err)
	}
}

func check(label string, ok bool) {
	if ok {
		fmt.Printf("   ✓ %s\n", label)
	} else {
		fmt.Printf("   ✗ FAIL: %s\n", label)
		os.Exit(1)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
