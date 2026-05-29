package ui_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/ui"
)

// fixtureWorkspace returns the absolute path to the test fixture workspace.
func fixtureWorkspace(t *testing.T) string {
	t.Helper()
	// Walk up from the test package to the repo root.
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "workspace")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	return abs
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	s, err := ui.NewServer(ui.ServerOptions{
		Workspace: fixtureWorkspace(t),
		Version:   "0.0.0-test",
		GitSHA:    "abc1234",
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return s.Handler()
}

func do(t *testing.T, h http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, path, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// -----------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/healthz", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var m map[string]string
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["status"] != "ok" {
		t.Errorf("want status=ok, got %q", m["status"])
	}
	if m["version"] != "0.0.0-test" {
		t.Errorf("want version=0.0.0-test, got %q", m["version"])
	}
}

func TestVersion(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/version", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var m map[string]string
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["cqlSpec"] != "1.5.3" {
		t.Errorf("want cqlSpec=1.5.3, got %q", m["cqlSpec"])
	}
}

func TestWorkspace(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/workspace", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Root     string   `json:"root"`
		CQLFiles []string `json:"cqlFiles"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Root == "" {
		t.Error("want non-empty root")
	}
	// Fixture workspace has Minimal-1.0.0.cql and BPMeasure-1.0.0.cql
	if len(resp.CQLFiles) < 1 {
		t.Errorf("want at least 1 CQL file, got %d: %v", len(resp.CQLFiles), resp.CQLFiles)
	}
}

func TestLibraries(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/libraries", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Libraries []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"libraries"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total < 1 {
		t.Fatalf("want total >= 1, got %d", resp.Total)
	}
	var found bool
	for _, lib := range resp.Libraries {
		if lib.Name == "Minimal" && lib.Version == "1.0.0" {
			found = true
		}
	}
	if !found {
		t.Errorf("want Minimal 1.0.0 in libraries, got: %+v", resp.Libraries)
	}
}

func TestLibraryByPath(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/libraries/libs/Minimal-1.0.0.cql", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(resp.Content, "library Minimal") {
		t.Errorf("want 'library Minimal' in content, got: %q", resp.Content)
	}
}

func TestLibraryByPathNotFound(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/libraries/libs/DoesNotExist.cql", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLibraryByPathEscape(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/libraries/../../etc/passwd", nil)
	// Should be 400 (path escapes workspace) or 404 (not found after sanitization)
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for path traversal, got 200")
	}
}

func TestTranslate(t *testing.T) {
	h := newTestServer(t)
	body := mustJSON(t, map[string]string{
		"content": "library Minimal version '1.0.0'",
	})
	w := do(t, h, "POST", "/api/translate", body)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Diagnostics []any  `json:"diagnostics"`
		ElmJSON     string `json:"elmJson"`
		ElmXML      string `json:"elmXml"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ElmJSON == "" {
		t.Error("want non-empty elmJson")
	}
	if resp.ElmXML == "" {
		t.Error("want non-empty elmXml")
	}
}

func TestTranslateSyntaxError(t *testing.T) {
	h := newTestServer(t)
	body := mustJSON(t, map[string]string{
		"content": "library Bad version 'x\n define @@@ invalid",
	})
	w := do(t, h, "POST", "/api/translate", body)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 with diagnostics, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Diagnostics []struct {
			Severity string `json:"severity"`
		} `json:"diagnostics"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Diagnostics) == 0 {
		t.Error("expected diagnostics for syntax error input")
	}
}

func TestTranslateBadBody(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "POST", "/api/translate", []byte("not json"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTranslateEmptyContent(t *testing.T) {
	h := newTestServer(t)
	body := mustJSON(t, map[string]string{"content": ""})
	w := do(t, h, "POST", "/api/translate", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestParityRunsEmpty(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/api/parity/runs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var runs []any
	if err := json.NewDecoder(w.Body).Decode(&runs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Fixture workspace has no parity runs — expect empty array
	if len(runs) != 0 {
		t.Errorf("want 0 runs, got %d", len(runs))
	}
}

func TestStaticFallback(t *testing.T) {
	h := newTestServer(t)
	// Unknown path should serve index.html (SPA fallback)
	w := do(t, h, "GET", "/some/unknown/path", nil)
	// Should be 200 with HTML content
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for SPA fallback, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<html") {
		t.Errorf("expected HTML body for SPA fallback, got: %q", w.Body.String()[:clampInt(200, w.Body.Len())])
	}
}

func clampInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
