package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

// Status classifies the outcome of a single fixture run.
type Status string

const (
	StatusMatch          Status = "match"
	StatusDifferJSON     Status = "differ-json"
	StatusEchoError      Status = "echo-error"
	StatusUpstreamError  Status = "upstream-error"
	StatusSkipped        Status = "skipped"
)

// FixtureResult holds the outcome of running one fixture through both translators.
type FixtureResult struct {
	Fixture        string
	Description    string
	Status         Status
	UpstreamJSON   string
	EchoJSON       string
	Diff           string
	UpstreamStderr string
	EchoStderr     string
	Error          string
	Duration       time.Duration
}

// Config holds harness configuration.
type Config struct {
	// CQFVersion is "3.29.0" or "4.8.0".
	CQFVersion string
	// ToolsDir is the root of tools/ (default "tools").
	ToolsDir string
	// TagFilter, if non-empty, runs only fixtures with this tag.
	TagFilter string
	// CorpusDir is the corpus root dir.
	CorpusDir string
}

// DefaultConfig returns a Config pointing at the default corpus.
func DefaultConfig(version string) Config {
	return Config{
		CQFVersion: version,
		ToolsDir:   "tools",
		CorpusDir:  "test/corpus/cqframework",
	}
}

// CQFTranslateFunc returns a translateFn that runs echo-elm in CQF-compatible mode
// (CQFMode=true, SignatureLevel=None, no annotations, no locators) using a
// DirSource rooted at the input file's directory for library resolution.
// This matches the options used by the upstream cqframework CLI.
func CQFTranslateFunc() func(cqlPath string) ([]byte, error) {
	return func(cqlPath string) ([]byte, error) {
		src, err := os.ReadFile(cqlPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", cqlPath, err)
		}
		libSrc := resolver.NewDirSource(filepath.Dir(cqlPath))
		result, err := echoelm.Translate(src, filepath.Base(cqlPath),
			echoelm.WithCQFOptions(),
			echoelm.WithLibrarySource(libSrc),
		)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(map[string]interface{}{"library": result.Library}, "", "  ")
	}
}

// Run executes all (or filtered) fixtures in the corpus and returns results.
func Run(cfg Config, translateFn func(cqlPath string) ([]byte, error)) ([]FixtureResult, error) {
	corpus, err := LoadCorpus(cfg.CorpusDir)
	if err != nil {
		return nil, err
	}

	launcher := upstreamLauncher(cfg.ToolsDir, cfg.CQFVersion)

	var results []FixtureResult
	for _, fix := range corpus.Fixtures {
		if cfg.TagFilter != "" && !fix.HasTag(cfg.TagFilter) {
			continue
		}
		r := runFixture(fix, corpus.Root, launcher, translateFn)
		results = append(results, r)
	}
	return results, nil
}

func runFixture(fix Fixture, corpusRoot, launcher string, translateFn func(string) ([]byte, error)) FixtureResult {
	start := time.Now()
	cqlPath := filepath.Join(corpusRoot, fix.Path)

	r := FixtureResult{
		Fixture:     fix.Path,
		Description: fix.Description,
	}

	// Run upstream cqframework CLI.
	upJSON, upStderr, err := runUpstream(launcher, cqlPath)
	r.UpstreamStderr = upStderr
	if err != nil {
		// If the corpus marks this fixture as expected-failure, treat it as skipped.
		if fix.ExpectedStatus == "failure" || fix.ExpectedStatus == "upstream-error" {
			r.Status = StatusSkipped
			r.Error = fmt.Sprintf("upstream (expected): %v", err)
			r.Duration = time.Since(start)
			return r
		}
		r.Status = StatusUpstreamError
		r.Error = fmt.Sprintf("upstream: %v", err)
		r.Duration = time.Since(start)
		return r
	}
	r.UpstreamJSON = normalizeJSON(upJSON)

	// Run echo-elm.
	echoBytes, err := translateFn(cqlPath)
	if err != nil {
		r.Status = StatusEchoError
		r.Error = fmt.Sprintf("echo-elm: %v", err)
		r.Duration = time.Since(start)
		return r
	}
	r.EchoJSON = normalizeJSON(string(echoBytes))

	// Compare.
	if r.UpstreamJSON == r.EchoJSON {
		r.Status = StatusMatch
	} else {
		r.Status = StatusDifferJSON
		r.Diff = simpleDiff(r.UpstreamJSON, r.EchoJSON)
	}

	r.Duration = time.Since(start)
	return r
}

// runUpstream invokes the launcher and captures the output JSON file.
func runUpstream(launcher, cqlPath string) (jsonOut, stderr string, err error) {
	// Determine output path: cqframework writes <name>.json next to input.
	dir := filepath.Dir(cqlPath)
	base := strings.TrimSuffix(filepath.Base(cqlPath), ".cql")
	outJSON := filepath.Join(dir, base+".json")

	// Remove any stale output.
	os.Remove(outJSON)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", launcher, "--input", cqlPath, "--format", "JSON")
	} else {
		cmd = exec.Command(launcher, "--input", cqlPath, "--format", "JSON")
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return "", stderrBuf.String(), fmt.Errorf("launcher exit %w", err)
	}

	data, err := os.ReadFile(outJSON)
	if err != nil {
		return "", stderrBuf.String(), fmt.Errorf("read output %s: %w", outJSON, err)
	}

	return string(data), stderrBuf.String(), nil
}

func upstreamLauncher(toolsDir, version string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(toolsDir, "cqframework", version, "run.bat")
	}
	return filepath.Join(toolsDir, "cqframework", version, "run.sh")
}

// normalizeJSON re-marshals JSON with sorted keys and consistent spacing.
func normalizeJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return strings.TrimSpace(s)
	}
	// Strip translator-version-specific fields before comparison.
	if m, ok := v.(map[string]interface{}); ok {
		stripVolatileFields(m)
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// stripVolatileFields removes fields that differ between translators/runs
// but are not semantically meaningful for parity checks.
func stripVolatileFields(m map[string]interface{}) {
	if lib, ok := m["library"].(map[string]interface{}); ok {
		if anns, ok := lib["annotation"].([]interface{}); ok {
			// Filter to keep only CqlToElmInfo entries (strip diagnostic/warning annotations).
			var kept []interface{}
			for _, a := range anns {
				if ann, ok := a.(map[string]interface{}); ok {
					// Strip version-specific and implementation-specific fields from CqlToElmInfo.
					delete(ann, "translatorVersion")
					delete(ann, "translatorOptions")
					delete(ann, "signatureLevel")
					delete(ann, "compatibilityLevel")
					// Drop diagnostic entries (CqlToElmError) — these are compiler
					// warnings/errors that vary by resolver depth (e.g. FHIRHelpers
					// overload warnings). Keep only structural annotations.
					if t, _ := ann["type"].(string); t == "CqlToElmError" {
						continue
					}
				}
				kept = append(kept, a)
			}
			lib["annotation"] = kept
		}
	}
}

// simpleDiff returns a naive line-by-line diff summary.
func simpleDiff(want, got string) string {
	wLines := strings.Split(want, "\n")
	gLines := strings.Split(got, "\n")

	var sb strings.Builder
	maxLines := len(wLines)
	if len(gLines) > maxLines {
		maxLines = len(gLines)
	}

	diffCount := 0
	for i := 0; i < maxLines && diffCount < 20; i++ {
		var w, g string
		if i < len(wLines) {
			w = wLines[i]
		}
		if i < len(gLines) {
			g = gLines[i]
		}
		if w != g {
			diffCount++
			fmt.Fprintf(&sb, "-%s\n+%s\n", w, g)
		}
	}
	if diffCount == 20 {
		fmt.Fprintf(&sb, "... (truncated after 20 diffs)\n")
	}
	return sb.String()
}

// Summary returns aggregate counts for a slice of results.
func Summary(results []FixtureResult) map[Status]int {
	m := make(map[Status]int)
	for _, r := range results {
		m[r.Status]++
	}
	return m
}
