package parity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/translator"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

// Status classifies the outcome of a single fixture run.
type Status string

const (
	StatusMatch         Status = "match"
	StatusDifferJSON    Status = "differ-json"
	StatusEchoError     Status = "echo-error"
	StatusUpstreamError Status = "upstream-error"
	StatusSkipped       Status = "skipped"
)

// FixtureResult holds the outcome of running one fixture through both translators.
type FixtureResult struct {
	Fixture        string
	Profile        string
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
	// ProfileFilter, if non-empty, runs only this profile name.
	ProfileFilter string
}

// DefaultConfig returns a Config pointing at the default corpus.
func DefaultConfig(version string) Config {
	return Config{
		CQFVersion: version,
		ToolsDir:   "tools",
		CorpusDir:  "test/corpus/cqframework",
	}
}

// profileOptions converts an OptionProfile”s translatorOptions map into
// a translator.Options overlay applied on top of CQFDefaultOptions.
func profileOptions(profile OptionProfile) translator.Options {
	opts := translator.CQFDefaultOptions()
	m := profile.TranslatorOptions
	if v, ok := m["enableAnnotations"]; ok {
		opts.EnableAnnotations, _ = v.(bool)
	}
	if v, ok := m["enableLocators"]; ok {
		opts.EnableLocators, _ = v.(bool)
	}
	if v, ok := m["signatureLevel"]; ok {
		if s, ok := v.(string); ok {
			opts.SignatureLevel = s
		}
	}
	if v, ok := m["disableListDemotion"]; ok {
		opts.DisableListDemotion, _ = v.(bool)
	}
	if v, ok := m["disableListPromotion"]; ok {
		opts.DisableListPromotion, _ = v.(bool)
	}
	if v, ok := m["disableListTraversal"]; ok {
		opts.DisableListTraversal, _ = v.(bool)
	}
	if v, ok := m["disableMethodInvocation"]; ok {
		opts.DisableMethodInvocation, _ = v.(bool)
	}
	if v, ok := m["requireFromKeyword"]; ok {
		opts.RequireFromKeyword, _ = v.(bool)
	}
	if v, ok := m["enableIntervalDemotion"]; ok {
		opts.EnableIntervalDemotion, _ = v.(bool)
	}
	if v, ok := m["enableIntervalPromotion"]; ok {
		opts.EnableIntervalPromotion, _ = v.(bool)
	}
	if v, ok := m["enableResultTypes"]; ok {
		opts.EnableResultTypes, _ = v.(bool)
	}
	if v, ok := m["compatibilityLevel"]; ok {
		if s, ok := v.(string); ok {
			opts.CompatibilityLevel = s
		}
	}
	return opts
}

// translateWithProfile returns an echo-elm translation using the given profile options.
//
//nolint:gocritic // hugeParam: internal helper with stable interface
func translateWithProfile(cqlPath string, opts translator.Options) ([]byte, error) {
	src, err := os.ReadFile(cqlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cqlPath, err)
	}
	libSrc := resolver.NewDirSource(filepath.Dir(cqlPath))
	result, err := echoelm.Translate(src, filepath.Base(cqlPath),
		echoelm.WithOptions(opts),
		echoelm.WithLibrarySource(libSrc),
	)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(map[string]interface{}{"library": result.Library}, "", "  ")
}

// CQFTranslateFunc returns a translateFn that runs echo-elm in CQF-compatible mode
// (CQFMode=true, SignatureLevel=None, no annotations, no locators) using a
// DirSource rooted at the input file”s directory for library resolution.
// This matches the options used by the upstream cqframework CLI”s default profile.
func CQFTranslateFunc() func(cqlPath string) ([]byte, error) {
	opts := translator.CQFDefaultOptions()
	return func(cqlPath string) ([]byte, error) {
		return translateWithProfile(cqlPath, opts)
	}
}

// Run executes all (or filtered) fixtures in the corpus and returns results.
// Each fixture runs against every applicable option profile.
// ProfileTranslateFunc returns a translateFn that applies the given OptionProfile
// on top of CQFDefaultOptions. Use in golden tests to drive each profile.
func ProfileTranslateFunc(profile OptionProfile) func(cqlPath string) ([]byte, error) {
	opts := profileOptions(profile)
	return func(cqlPath string) ([]byte, error) {
		return translateWithProfile(cqlPath, opts)
	}
}

//nolint:gocritic // hugeParam: public API
func Run(cfg Config, translateFn func(cqlPath string) ([]byte, error)) ([]FixtureResult, error) {
	corpus, err := LoadCorpus(cfg.CorpusDir)
	if err != nil {
		return nil, err
	}

	launcher := upstreamLauncher(cfg.ToolsDir, cfg.CQFVersion)

	// Sort profile names for deterministic output.
	profileNames := sortedProfileNames(corpus.OptionProfiles)

	var results []FixtureResult
	for _, fix := range corpus.Fixtures {
		if cfg.TagFilter != "" && !fix.HasTag(cfg.TagFilter) {
			continue
		}

		fixProfiles := resolveProfileNames(fix.ProfileNames(corpus.OptionProfiles), profileNames)
		for _, profileName := range fixProfiles {
			if cfg.ProfileFilter != "" && profileName != cfg.ProfileFilter {
				continue
			}
			profile := corpus.OptionProfiles[profileName]
			opts := profileOptions(profile)
			translateProfileFn := func(cqlPath string) ([]byte, error) {
				return translateWithProfile(cqlPath, opts)
			}
			r := runFixture(fix, corpus.Root, launcher, profile.CLIFlags, translateProfileFn)
			r.Profile = profileName
			results = append(results, r)
		}
	}
	return results, nil
}

//nolint:gocritic // hugeParam: internal helper with stable interface
func runFixture(fix Fixture, corpusRoot, launcher string, extraFlags []string, translateFn func(string) ([]byte, error)) FixtureResult {
	start := time.Now()
	cqlPath := filepath.Join(corpusRoot, fix.Path)

	r := FixtureResult{
		Fixture:     fix.Path,
		Description: fix.Description,
	}

	// Run upstream cqframework CLI.
	upJSON, upStderr, err := runUpstream(launcher, cqlPath, extraFlags)
	r.UpstreamStderr = upStderr
	if err != nil {
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

// runUpstream invokes the launcher with optional extra flags and captures the output JSON file.
func runUpstream(launcher, cqlPath string, extraFlags []string) (jsonOut, stderr string, err error) {
	dir := filepath.Dir(cqlPath)
	base := strings.TrimSuffix(filepath.Base(cqlPath), ".cql")
	outJSON := filepath.Join(dir, base+".json")

	_ = os.Remove(outJSON)

	args := append([]string{"--input", cqlPath, "--format", "JSON"}, extraFlags...)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(context.Background(), "cmd", "/C", launcher)
		cmd.Args = append(cmd.Args, args...)
	} else {
		cmd = exec.CommandContext(context.Background(), launcher, args...)
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

// GenerateGoldens runs the upstream CQF CLI for every non-failure fixture × profile
// and writes the normalized JSON output to outputDir/<version>/<profile>/<fixture>.json.
// It is intended to be called once (e.g. via demo/goldens) to produce or refresh
// the committed golden files used by TestGoldenCorpus.
// Returns the number of golden files written and a version-diff report.
//
//nolint:gocritic // hugeParam: public API
func GenerateGoldens(cfg Config, outputDir string) (written int, versionDiff []string, err error) {
	return generateGoldensInner(cfg, filepath.Join(outputDir, cfg.CQFVersion))
}

// GenerateGoldensTo runs the upstream CQF CLI for every non-failure fixture × profile
// and writes normalized JSON to targetDir/<profile>/<fixture>.json (no version subdirectory).
// Use this to write the canonical collapsed golden set.
//
//nolint:gocritic // hugeParam: public API
func GenerateGoldensTo(cfg Config, targetDir string) (int, error) {
	n, _, err := generateGoldensInner(cfg, targetDir)
	return n, err
}

//nolint:gocritic // hugeParam: internal; stable interface
func generateGoldensInner(cfg Config, baseDir string) (written int, versionDiff []string, err error) {
	corpus, err := LoadCorpus(cfg.CorpusDir)
	if err != nil {
		return 0, nil, err
	}

	launcher := upstreamLauncher(cfg.ToolsDir, cfg.CQFVersion)
	profileNames := sortedProfileNames(corpus.OptionProfiles)

	for _, fix := range corpus.Fixtures {
		fixProfiles := resolveProfileNames(fix.ProfileNames(corpus.OptionProfiles), profileNames)
		for _, profileName := range fixProfiles {
			profile := corpus.OptionProfiles[profileName]
			outPath := filepath.Join(baseDir, profileName,
				strings.TrimSuffix(fix.Path, ".cql")+".json")

			if err2 := os.MkdirAll(filepath.Dir(outPath), 0o755); err2 != nil {
				return written, nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err2)
			}

			jsonOut, upStderr, upErr := runUpstream(launcher, filepath.Join(corpus.Root, fix.Path), profile.CLIFlags)
			if upErr != nil {
				// Skip expected failures silently.
				if fix.ExpectedStatus == "failure" || fix.ExpectedStatus == "upstream-error" {
					continue
				}
				// For unexpected upstream failures (e.g. a profile flag rejects valid code)
				// skip writing a golden and log a warning — don't abort the whole run.
				_ = upStderr
				fmt.Fprintf(os.Stderr, "  SKIP %s/%s: upstream error (%v)\n", profileName, fix.Path, upErr)
				continue
			}

			normalized := normalizeJSON(jsonOut)
			if err2 := os.WriteFile(outPath, []byte(normalized), 0o644); err2 != nil {
				return written, nil, fmt.Errorf("write %s: %w", outPath, err2)
			}
			written++
		}
	}
	return written, nil, nil
}

// CompareVersionGoldens walks two version golden trees and reports any files
// that differ. Returns a list of differing paths (relative to the golden root).
// Fixture paths in skip (as listed in corpus.yaml under versionDivergent) are
// known to differ between upstream versions and are not reported.
func CompareVersionGoldens(outputDir, versionA, versionB string, skip map[string]bool) ([]string, error) {
	dirA := filepath.Join(outputDir, versionA)
	var diffs []string

	err := filepath.Walk(dirA, func(pathA string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dirA, pathA)
		if skip[fixturePathFromGoldenRel(rel)] {
			return nil
		}
		pathB := filepath.Join(outputDir, versionB, rel)

		aBytes, err := os.ReadFile(pathA)
		if err != nil {
			return err
		}
		bBytes, err := os.ReadFile(pathB)
		if os.IsNotExist(err) {
			diffs = append(diffs, rel+" (missing in "+versionB+")")
			return nil
		}
		if err != nil {
			return err
		}
		if normalizeJSON(string(aBytes)) != normalizeJSON(string(bBytes)) {
			diffs = append(diffs, rel)
		}
		return nil
	})
	return diffs, err
}

// fixturePathFromGoldenRel converts a golden path relative to a version root
// ("default/elm-nodes/Foo.json") into the corpus fixture path it came from
// ("elm-nodes/Foo.cql").
func fixturePathFromGoldenRel(rel string) string {
	rel = filepath.ToSlash(rel)
	if i := strings.Index(rel, "/"); i >= 0 {
		rel = rel[i+1:]
	}
	return strings.TrimSuffix(rel, ".json") + ".cql"
}

// normalizeJSON re-marshals JSON with sorted keys and consistent spacing, after
// reducing the few things that cannot be compared across translators.
//
// The rule for adding anything here: a field may only be normalized away when it
// carries no meaning for a consumer of the ELM, or when what it means is checked
// some other way. What is currently reduced, and why:
//
//   - translatorVersion — names the producing translator; volatile by definition.
//   - empty "annotation": [] and "t": [] arrays — CQF 4.8.0 emits them where
//     3.29.0 omits them, so they are pure serializer asymmetry.
//   - "t" tag arrays are sorted by name — tag content is compared, ordering is not.
//   - localId (and the "r" references that point at it) — an internal node index.
//     CQF numbers from a pre-order ANTLR rule visit; echo-elm uses its own counter.
//   - annotation s-tree shape — reduced to its concatenated leaf text. CQF splits
//     the source into per-grammar-rule segments ("define ", "\"X\"", ":\n  ", …)
//     while echo-elm emits one span. The text itself is still compared, so wrong
//     annotation content still fails; only the segmentation is unverified.
//
// Deliberately NOT normalized, because each is meaningful and each was hiding a
// real defect when it was: statement/definition ordering, resultTypeName and
// resultTypeSpecifier, and translatorOptions/signatureLevel.
//
// CqlToElmError annotations are dropped because echo-elm emits no diagnostic
// annotations at all yet; that gap is tracked in issues/03-cqf-parity-gaps.md
// rather than being silently absorbed here.
func normalizeJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return strings.TrimSpace(s)
	}
	if m, ok := v.(map[string]interface{}); ok {
		stripVolatileFields(m)
	}
	stripEmptyAnnotations(v)
	canonicalizeAnnotationFields(v)
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// canonicalizeAnnotationFields strips the internal node index and reduces
// annotation s-trees to their text. See normalizeJSON for the full rationale;
// TestAnnotationTextInvariant checks that the retained text is in fact the
// def's source, so the reduction is not a blank cheque.
func canonicalizeAnnotationFields(v interface{}) {
	switch node := v.(type) {
	case map[string]interface{}:
		delete(node, "localId")
		// Recognize Annotation entries and collapse their s-tree.
		if t, _ := node["type"].(string); t == "Annotation" {
			if sv, ok := node["s"]; ok {
				text := collectAnnotationText(sv)
				node["s"] = map[string]interface{}{
					"s": []interface{}{
						map[string]interface{}{"value": []interface{}{text}},
					},
				}
				_ = ok
			}
		}
		for _, child := range node {
			canonicalizeAnnotationFields(child)
		}
	case []interface{}:
		for _, item := range node {
			canonicalizeAnnotationFields(item)
		}
	}
}

// collectAnnotationText returns the concatenation of every leaf "value" string
// found anywhere inside an annotation s-tree node.
func collectAnnotationText(v interface{}) string {
	var sb strings.Builder
	var walk func(x interface{})
	walk = func(x interface{}) {
		switch node := x.(type) {
		case map[string]interface{}:
			if val, ok := node["value"]; ok {
				switch vv := val.(type) {
				case string:
					sb.WriteString(vv)
				case []interface{}:
					for _, s := range vv {
						if str, ok := s.(string); ok {
							sb.WriteString(str)
						}
					}
				}
			}
			if s, ok := node["s"]; ok {
				walk(s)
			}
		case []interface{}:
			for _, item := range node {
				walk(item)
			}
		}
	}
	walk(v)
	return sb.String()
}

// stripEmptyAnnotations recursively removes version-format-only empty arrays
// from the JSON tree so 3.29.0 and 4.8.0 goldens can collapse:
//   - "annotation": []  on any ELM node (4.8.0 emits it, 3.29.0 omits it)
//   - "t": []           inside Annotation objects (tag array, same asymmetry)
func stripEmptyAnnotations(v interface{}) {
	switch node := v.(type) {
	case map[string]interface{}:
		if arr, ok := node["annotation"].([]interface{}); ok && len(arr) == 0 {
			delete(node, "annotation")
		}
		// Strip empty tag array from Annotation nodes; sort non-empty t arrays by name.
		if typ, _ := node["type"].(string); typ == "Annotation" {
			if arr, ok := node["t"].([]interface{}); ok {
				if len(arr) == 0 {
					delete(node, "t")
				} else {
					sort.SliceStable(arr, func(i, j int) bool {
						mi, _ := arr[i].(map[string]interface{})
						mj, _ := arr[j].(map[string]interface{})
						ni, _ := mi["name"].(string)
						nj, _ := mj["name"].(string)
						return ni < nj
					})
				}
			}
		}
		for _, child := range node {
			stripEmptyAnnotations(child)
		}
	case []interface{}:
		for _, item := range node {
			stripEmptyAnnotations(item)
		}
	}
}

// stripVolatileFields removes fields that differ between translators/runs
// but are not semantically meaningful for parity checks.
func stripVolatileFields(m map[string]interface{}) {
	if lib, ok := m["library"].(map[string]interface{}); ok {
		if anns, ok := lib["annotation"].([]interface{}); ok {
			var kept []interface{}
			for _, a := range anns {
				if ann, ok := a.(map[string]interface{}); ok {
					// translatorVersion is the only genuinely volatile field here:
					// it names the producing translator. translatorOptions and
					// signatureLevel describe how the ELM was produced and are
					// compared. CqlToElmError annotations carry diagnostics that
					// echo-elm does not yet emit; see the parity notes in
					// issues/03-cqf-parity-gaps.md.
					delete(ann, "translatorVersion")
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
	for i := range results {
		m[results[i].Status]++
	}
	return m
}

// sortedProfileNames returns profile keys in sorted order.
func sortedProfileNames(profiles map[string]OptionProfile) []string {
	names := make([]string, 0, len(profiles))
	for k := range profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// resolveProfileNames filters the fixture”s profile names to those present
// in the corpus”s option_profiles map, preserving order.
func resolveProfileNames(fixProfiles, allProfiles []string) []string {
	allSet := make(map[string]bool, len(allProfiles))
	for _, p := range allProfiles {
		allSet[p] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range fixProfiles {
		if allSet[p] && !seen[p] {
			out = append(out, p)
			seen[p] = true
		}
	}
	return out
}
