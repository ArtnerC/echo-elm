// Package ui implements the loopback HTTP server backing the echo-elm workbench.
// It exposes a JSON API at /api/* and serves the embedded SvelteKit static build at /.
// The server refuses to bind non-loopback addresses unless AllowRemote is set.
package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/parser"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

const cqlSpec = "1.5.3"

// ServerOptions configure the UI server.
type ServerOptions struct {
	Workspace   string // absolute path; defaults to cwd
	Version     string
	GitSHA      string
	AllowRemote bool // skip loopback enforcement (power-user only)
}

// Server is the echo-elm workbench HTTP server.
type Server struct {
	opts ServerOptions
	mux  *http.ServeMux
}

// NewServer creates a Server from the provided options.
// workspace is cleaned and resolved to an absolute path.
func NewServer(opts ServerOptions) (*Server, error) {
	if opts.Workspace == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("ui: getwd: %w", err)
		}
		opts.Workspace = wd
	}
	abs, err := filepath.Abs(opts.Workspace)
	if err != nil {
		return nil, fmt.Errorf("ui: resolve workspace %q: %w", opts.Workspace, err)
	}
	opts.Workspace = abs

	s := &Server{opts: opts}
	s.mux = s.buildMux()
	return s, nil
}

// Handler returns the HTTP handler (for testing without a real listener).
func (s *Server) Handler() http.Handler { return s.mux }

// ListenAndServe binds addr and starts serving. It blocks until the server
// exits or an error occurs. addr defaults to "127.0.0.1:8787".
func (s *Server) ListenAndServe(addr string) error {
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	if !s.opts.AllowRemote {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("ui: invalid addr %q: %w", addr, err)
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("ui: refusing non-loopback bind %q (use --allow-remote to override)", addr)
		}
	}
	return http.ListenAndServe(addr, s.mux)
}

// -----------------------------------------------------------------------
// Router
// -----------------------------------------------------------------------

func (s *Server) buildMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Meta
	mux.HandleFunc("GET /api/healthz", s.handleHealthz)
	mux.HandleFunc("GET /api/version", s.handleVersion)

	// Workspace / libraries
	mux.HandleFunc("GET /api/workspace", s.handleWorkspace)
	mux.HandleFunc("GET /api/libraries", s.handleLibraries)
	mux.HandleFunc("GET /api/libraries/{filepath...}", s.handleLibraryByPath)

	// Translate
	mux.HandleFunc("POST /api/translate", s.handleTranslate)
	mux.HandleFunc("OPTIONS /api/translate", handleOptions)

	// Parity (read-only viewer)
	mux.HandleFunc("GET /api/parity/runs", s.handleParityRuns)
	mux.HandleFunc("GET /api/parity/runs/{id}/report.md", s.handleParityRunReportMD)
	mux.HandleFunc("GET /api/parity/runs/{id}/diffs/{file...}", s.handleParityDiff)
	mux.HandleFunc("GET /api/parity/runs/{id}", s.handleParityRun)

	// Static SPA — must be last
	mux.Handle("/", s.spaHandler())

	return mux
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------
// Static SPA handler
// -----------------------------------------------------------------------

// spaFS wraps an fs.FS so that missing files are served as index.html,
// enabling client-side SPA routing.
type spaFS struct{ fs.FS }

func (s spaFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		return s.FS.Open("index.html")
	}
	return f, err
}

func (s *Server) spaHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		// Shouldn't happen — the embed always has static/
		panic("ui: cannot sub staticFiles: " + err.Error())
	}
	return http.FileServerFS(spaFS{sub})
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

type problemJSON struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	p := problemJSON{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

// safePath validates and resolves a workspace-relative path.
// Returns an error if path is absolute or escapes the workspace.
func (s *Server) safePath(rel string) (string, error) {
	// Reject absolute paths immediately
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute path not allowed")
	}
	clean := filepath.Clean(filepath.Join(s.opts.Workspace, rel))
	ws := s.opts.Workspace
	// Must be inside workspace
	if clean != ws && !strings.HasPrefix(clean, ws+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return clean, nil
}

// -----------------------------------------------------------------------
// Meta handlers
// -----------------------------------------------------------------------

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": s.opts.Version,
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":   s.opts.Version,
		"gitSha":    s.opts.GitSHA,
		"goVersion": runtime.Version(),
		"cqlSpec":   cqlSpec,
	})
}

// -----------------------------------------------------------------------
// Workspace / library handlers
// -----------------------------------------------------------------------

type workspaceResponse struct {
	Root       string   `json:"root"`
	CQLFiles   []string `json:"cqlFiles"`
	ModelInfos []string `json:"modelInfos"`
}

func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	files, _ := walkCQLFiles(s.opts.Workspace)
	writeJSON(w, http.StatusOK, workspaceResponse{
		Root:       s.opts.Workspace,
		CQLFiles:   files,
		ModelInfos: []string{},
	})
}

type libraryHeader struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Path     string   `json:"path"`
	Includes []string `json:"includes"`
}

func (s *Server) handleLibraries(w http.ResponseWriter, r *http.Request) {
	files, _ := walkCQLFiles(s.opts.Workspace)
	headers := make([]libraryHeader, 0, len(files))
	for _, rel := range files {
		abs, err := s.safePath(rel)
		if err != nil {
			continue
		}
		src, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		hdr := parseLibraryHeader(src, rel)
		headers = append(headers, hdr)
	}
	if headers == nil {
		headers = []libraryHeader{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"libraries": headers,
		"total":     len(headers),
	})
}

func (s *Server) handleLibraryByPath(w http.ResponseWriter, r *http.Request) {
	rel := r.PathValue("filepath")
	abs, err := s.safePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	src, err := os.ReadFile(abs)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "library not found: "+rel)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"path":    rel,
		"content": string(src),
	})
}

// -----------------------------------------------------------------------
// Translate handler
// -----------------------------------------------------------------------

type translateRequest struct {
	Content string              `json:"content"`
	Path    string              `json:"path"`
	Options translateReqOptions `json:"options"`
}

type translateReqOptions struct {
	CQFMode              bool   `json:"cqfMode"`
	EnableAnnotations    bool   `json:"enableAnnotations"`
	EnableLocators       bool   `json:"enableLocators"`
	SignatureLevel       string `json:"signatureLevel"`
	DisableListDemotion  bool   `json:"disableListDemotion"`
	DisableListPromotion bool   `json:"disableListPromotion"`
	ValidateUnits        bool   `json:"validateUnits"`
}

type translateResponse struct {
	Diagnostics []diagnosticResp `json:"diagnostics"`
	ElmXML      string           `json:"elmXml"`
	ElmJSON     string           `json:"elmJson"`
	Stats       translateStats   `json:"stats"`
}

type diagnosticResp struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Locator  string `json:"locator"`
}

type translateStats struct {
	ParseMs     int64 `json:"parseMs"`
	CompileMs   int64 `json:"compileMs"`
	SerializeMs int64 `json:"serializeMs"`
}

func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req translateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	sourceName := req.Path
	if sourceName == "" {
		sourceName = "input.cql"
	}

	// Build echoelm options
	opts := buildTranslateOptions(req.Options)

	t0 := time.Now()
	result, err := echoelm.Translate([]byte(req.Content), sourceName, opts...)
	compileMs := time.Since(t0).Milliseconds()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "translation error: "+err.Error())
		return
	}

	t1 := time.Now()
	jsonBytes, _ := json.Marshal(result)
	xmlBytes, _ := result.XMLBytes()
	serializeMs := time.Since(t1).Milliseconds()

	diags := make([]diagnosticResp, 0, len(result.Diagnostics))
	for _, d := range result.Diagnostics {
		diags = append(diags, diagnosticResp{
			Severity: d.Severity,
			Message:  d.Message,
			Locator:  d.Locator,
		})
	}
	if diags == nil {
		diags = []diagnosticResp{}
	}

	writeJSON(w, http.StatusOK, translateResponse{
		Diagnostics: diags,
		ElmXML:      string(xmlBytes),
		ElmJSON:     string(jsonBytes),
		Stats: translateStats{
			ParseMs:     0,
			CompileMs:   compileMs,
			SerializeMs: serializeMs,
		},
	})
}

func buildTranslateOptions(o translateReqOptions) []echoelm.Option {
	opts := []echoelm.Option{
		echoelm.WithCQFMode(o.CQFMode),
		echoelm.WithAnnotations(o.EnableAnnotations),
		echoelm.WithLocators(o.EnableLocators),
	}
	if o.SignatureLevel != "" {
		opts = append(opts, echoelm.WithSignatureLevel(o.SignatureLevel))
	}
	return opts
}

// -----------------------------------------------------------------------
// Parity handlers (read-only)
// -----------------------------------------------------------------------

type parityRunSummary struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func (s *Server) handleParityRuns(w http.ResponseWriter, r *http.Request) {
	runsDir := filepath.Join(s.opts.Workspace, "parity", "runs")
	entries, err := os.ReadDir(runsDir)
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, http.StatusOK, []parityRunSummary{})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	runs := make([]parityRunSummary, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		runs = append(runs, parityRunSummary{
			ID:   e.Name(),
			Path: filepath.Join("parity", "runs", e.Name()),
		})
	}
	// Newest first (directory names are typically timestamp-prefixed)
	sort.Slice(runs, func(i, j int) bool { return runs[i].ID > runs[j].ID })
	if runs == nil {
		runs = []parityRunSummary{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) handleParityRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path, err := s.safePath(filepath.Join("parity", "runs", id, "report.json"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.serveFileRaw(w, path, "application/json")
}

func (s *Server) handleParityRunReportMD(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path, err := s.safePath(filepath.Join("parity", "runs", id, "report.md"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.serveFileRaw(w, path, "text/markdown; charset=utf-8")
}

func (s *Server) handleParityDiff(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	file := r.PathValue("file")
	path, err := s.safePath(filepath.Join("parity", "runs", id, "diffs", file))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Determine content type from extension
	ct := "application/octet-stream"
	switch {
	case strings.HasSuffix(file, ".json"):
		ct = "application/json"
	case strings.HasSuffix(file, ".xml"):
		ct = "application/xml"
	case strings.HasSuffix(file, ".md"):
		ct = "text/markdown; charset=utf-8"
	}
	s.serveFileRaw(w, path, ct)
}

func (s *Server) serveFileRaw(w http.ResponseWriter, absPath, contentType string) {
	data, err := os.ReadFile(absPath)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// -----------------------------------------------------------------------
// Filesystem helpers
// -----------------------------------------------------------------------

// walkCQLFiles returns workspace-relative paths of all .cql files under dir.
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

// parseLibraryHeader parses CQL source and extracts name, version, and include aliases.
func parseLibraryHeader(src []byte, sourceName string) libraryHeader {
	hdr := libraryHeader{Path: sourceName, Includes: []string{}}
	pr, _ := parser.ParseBytes(src, sourceName)
	if pr == nil || pr.Library == nil {
		return hdr
	}
	lib := pr.Library
	if lib.Name != nil {
		hdr.Name = lib.Name.Name
		hdr.Version = lib.Name.Version
	}
	for _, inc := range lib.Includes {
		alias := inc.LocalName
		if alias == "" {
			alias = inc.Path
		}
		hdr.Includes = append(hdr.Includes, alias)
	}
	return hdr
}
