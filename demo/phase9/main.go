// Phase 9 demo: start echo-elm ui, verify the SvelteKit workbench is served
// correctly (all three routes + API endpoints), then shut down.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║     echo-elm  Phase 9  —  Svelte Workbench Demo    ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	binary := resolveBinary()
	workspace := fixtureWorkspace()

	// Start echo-elm ui as a subprocess.
	cmd := exec.CommandContext(context.Background(), binary, "ui", "--workspace", workspace)
	cmd.Stderr = os.Stderr
	must("start ui server", cmd.Start())
	defer cmd.Process.Kill() //nolint:errcheck

	// Wait for server to accept connections.
	waitReady("http://127.0.0.1:8787/api/healthz", 8*time.Second)
	fmt.Println("Server listening on http://127.0.0.1:8787")

	// ── Check 1: SvelteKit index.html served at / ─────────────────────────
	fmt.Println("── Check 1: SvelteKit app served at /")
	body := mustGet("http://127.0.0.1:8787/")
	check("contains <title>", strings.Contains(body, "<title>"))
	check("contains _app entry script", strings.Contains(body, "_app/immutable/entry/"))
	fmt.Println()

	// ── Check 2: SPA fallback — unknown route returns index.html ─────────
	fmt.Println("── Check 2: SPA fallback for /translate")
	body = mustGet("http://127.0.0.1:8787/translate")
	check("fallback returns html", strings.Contains(body, "_app/immutable/entry/"))
	fmt.Println()

	// ── Check 3: SPA fallback for parity route ───────────────────────────
	fmt.Println("── Check 3: SPA fallback for /parity/some-run-id")
	body = mustGet("http://127.0.0.1:8787/parity/some-run-id")
	check("parity route returns html", strings.Contains(body, "_app/immutable/entry/"))
	fmt.Println()

	// ── Check 4: API still works ──────────────────────────────────────────
	fmt.Println("── Check 4: /api/libraries returns JSON")
	body = mustGet("http://127.0.0.1:8787/api/libraries")
	check("has total field", strings.Contains(body, `"total"`))
	check("has BPMeasure library", strings.Contains(body, "BPMeasure"))
	fmt.Println()

	// ── Check 5: Static asset served ──────────────────────────────────────
	fmt.Println("── Check 5: JS entry bundle served")
	body = mustGet("http://127.0.0.1:8787/_app/env.js")
	check("env.js non-empty", len(body) > 0)
	fmt.Println()

	fmt.Println("Phase 9 complete ✓  SvelteKit workbench embedded and serving.")
	fmt.Printf("\nOpen http://127.0.0.1:8787 in your browser to use the workbench.\n")
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

func waitReady(url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "FATAL: server did not become ready at %s within %v\n", url, timeout)
	os.Exit(1)
}

func mustGet(url string) string {
	resp, err := http.Get(url) //nolint:noctx
	must("GET "+url, err)
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		fmt.Fprintf(os.Stderr, "FATAL: GET %s → %d\n", url, resp.StatusCode)
		os.Exit(1)
	}
	b, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	must("read body", err)
	return string(b)
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
