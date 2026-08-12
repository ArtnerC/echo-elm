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

	"github.com/artnerc/echo-elm/internal/elm"
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

// PinnedCQFVersion is the single cqframework release echo-elm targets. It names
// the launcher directory under tools/cqframework/ that `task parity:jar`
// installs, and it is the version the committed goldens were generated from.
//
// echo-elm previously pinned 3.29.0 and 4.8.0 at once and collapsed their
// goldens into one set. That only worked because the harness normalized away
// every field the two disagreed on, and those normalizations were load-bearing
// for the collapse rather than justified on their own terms. One pin means a
// difference is a difference.
const PinnedCQFVersion = "5.0.0"

// Config holds harness configuration.
type Config struct {
	// CQFVersion selects the launcher under tools/cqframework/<version>/.
	// Defaults to PinnedCQFVersion; other values are for ad-hoc investigation
	// of upstream behavior changes, not for the committed baseline.
	CQFVersion string
	// ToolsDir is the root of tools/ (default "tools").
	ToolsDir string
	// TagFilter, if non-empty, runs only fixtures with this tag.
	TagFilter string
	// CorpusDir is the corpus root dir.
	CorpusDir string
	// ProfileFilter, if non-empty, runs only this profile name.
	ProfileFilter string
	// RefDir, when non-empty, reads reference ELM from
	// <RefDir>/<profile>/<fixture>.json instead of invoking the CQF JAR. Use it
	// to compare against ELM that already exists — a measure bundle's compiled
	// ELM, or a committed golden set — and in environments without Java.
	RefDir string
	// LibDir, when non-empty, is searched for included CQL libraries in addition
	// to each fixture's own directory. A bundle extracted to one flat directory
	// needs this, since its libraries do not sit beside the fixture.
	LibDir string
	// BundleShapedRef marks the reference ELM as coming from FHIR
	// Library.content[], which the CQF JAXB/MOXy writer produces rather than the
	// cql-to-elm CLI. The two shapes disagree about things that carry no meaning
	// — empty collections, implied type discriminators — so the comparison
	// reduces both sides to the lean CLI shape first. Set by MaterializeBundle.
	//
	// It must stay off for CLI-sourced references: there, echo-elm emitting
	// "signature": [] is a real requirement that CQF also satisfies, and
	// reducing it away would stop checking it.
	BundleShapedRef bool
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
	return translateWithProfileAndLibDir(cqlPath, opts, "")
}

// translateWithProfileAndLibDir translates cqlPath, resolving includes from the
// fixture's own directory first and then from libDir when one is given.
//
//nolint:gocritic // hugeParam: internal helper with stable interface
func translateWithProfileAndLibDir(cqlPath string, opts translator.Options, libDir string) ([]byte, error) {
	src, err := os.ReadFile(cqlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cqlPath, err)
	}
	var libSrc resolver.LibrarySource = resolver.NewDirSource(filepath.Dir(cqlPath))
	if libDir != "" {
		libSrc = resolver.NewMultiSource(libSrc, resolver.NewDirSource(libDir))
	}
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

// ProfileTranslateFuncWithLibDir is ProfileTranslateFunc with an additional
// library search directory, for corpora whose libraries do not sit beside the
// fixture that includes them.
func ProfileTranslateFuncWithLibDir(profile OptionProfile, libDir string) func(cqlPath string) ([]byte, error) {
	opts := profileOptions(profile)
	return func(cqlPath string) ([]byte, error) {
		return translateWithProfileAndLibDir(cqlPath, opts, libDir)
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
			libDir := cfg.LibDir
			translateProfileFn := func(cqlPath string) ([]byte, error) {
				return translateWithProfileAndLibDir(cqlPath, opts, libDir)
			}
			r := runFixtureWithRef(fix, corpus.Root, launcher, profile.CLIFlags,
				translateProfileFn, cfg.RefDir, profileName, cfg.BundleShapedRef)
			r.Profile = profileName
			results = append(results, r)
		}
	}
	return results, nil
}

// referenceELM returns the reference ELM for a fixture. With refDir set it is
// read from <refDir>/<profile>/<fixture>.json; otherwise the upstream CQF CLI
// produces it.
func referenceELM(refDir, profileName, corpusRoot, launcher string, fix *Fixture, extraFlags []string) (elmJSON, stderr string, err error) {
	if refDir == "" {
		return runUpstream(launcher, filepath.Join(corpusRoot, fix.Path), extraFlags)
	}
	refPath := filepath.Join(refDir, profileName, strings.TrimSuffix(fix.Path, ".cql")+".json")
	data, readErr := os.ReadFile(refPath)
	if readErr != nil {
		return "", "", fmt.Errorf("reference ELM not found: %w", readErr)
	}
	return string(data), "", nil
}

//nolint:gocritic // hugeParam: internal helper with stable interface
func runFixtureWithRef(fix Fixture, corpusRoot, launcher string, extraFlags []string, translateFn func(string) ([]byte, error), refDir, profileName string, cfgShape bool) FixtureResult {
	start := time.Now()
	cqlPath := filepath.Join(corpusRoot, fix.Path)

	r := FixtureResult{
		Fixture:     fix.Path,
		Description: fix.Description,
	}

	// Reference ELM: pre-built when --ref-dir is given, else from the CQF CLI.
	upJSON, upStderr, err := referenceELM(refDir, profileName, corpusRoot, launcher, &fix, extraFlags)
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
	r.UpstreamJSON = normalizeShape(upJSON, cfgShape)

	// Run echo-elm.
	echoBytes, err := translateFn(cqlPath)
	if err != nil {
		r.Status = StatusEchoError
		r.Error = fmt.Sprintf("echo-elm: %v", err)
		r.Duration = time.Since(start)
		return r
	}
	r.EchoJSON = normalizeShape(string(echoBytes), cfgShape)

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

// normalizeJSON re-marshals JSON with sorted keys and consistent spacing, after
// reducing the few things that cannot be compared across translators.
//
// The rule for adding anything here: a field may only be normalized away when it
// carries no meaning for a consumer of the ELM, or when what it means is checked
// some other way. What is currently reduced, and why:
//
//   - translatorVersion — names the producing translator; volatile by definition.
//   - "type" discriminators whose value the node's position already implies.
//     Reference ELM read out of a FHIR bundle comes from the JAXB/MOXy writer,
//     which stamps a type on nearly every node; the cql-to-elm CLI does not.
//     Comparing the two shapes is otherwise pure noise. Only a discriminator
//     that AGREES with its position is dropped, so a FunctionDef sitting where
//     an ExpressionDef is implied still differs. No-op on CLI-shaped input.
//   - empty "annotation": [] and "t": [] arrays — CQF emits the empty container
//     on nearly every node; echo-elm omits it. An empty container carries no
//     information for a consumer of the ELM, so this meets the rule above, but
//     note what changed: it used to be justified as 3.29.0-vs-4.8.0 serializer
//     asymmetry, and that reason died with the second pin. It is now squarely an
//     echo-elm emission gap that this reduction is choosing to tolerate.
//     Measured against CQF 5.0.0: 130 of 294 fixtures depend on it.
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
	elm.StripImpliedTypesTree(v)
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

// normalizeShape is normalizeJSON plus, when the reference came out of a FHIR
// bundle, the extra reduction that shape requires.
func normalizeShape(s string, bundleShaped bool) string {
	if !bundleShaped {
		return normalizeJSON(s)
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return strings.TrimSpace(s)
	}
	if m, ok := v.(map[string]interface{}); ok {
		stripVolatileFields(m)
		stripSignatureLevel(m)
	}
	elm.StripImpliedTypesTree(v)
	stripEmptyAnnotations(v)
	stripEmptyBundleCollections(v)
	canonicalizeAnnotationFields(v)
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// bundleOmittedCollections are the collection fields the JAXB/MOXy writer drops
// when empty, because an empty collection maps to no XML elements and comes back
// out of the object model as an absent key. The cql-to-elm CLI emits them, and
// echo-elm matches the CLI (see issues/04 G1) — so on the bundle path the
// difference is the writer's, and reducing it is the only way to compare.
var bundleOmittedCollections = []string{
	"signature", "let", "include", "codeFilter", "dateFilter", "otherFilter",
	"element", "operand", "codeSystem", "source", "relationship", "sort", "by",
	"caseItem", "def", "usings", "parameters", "codes", "concepts", "contexts",
}

// stripEmptyBundleCollections removes those fields wherever they are empty.
func stripEmptyBundleCollections(v interface{}) {
	switch node := v.(type) {
	case map[string]interface{}:
		for _, field := range bundleOmittedCollections {
			if arr, ok := node[field].([]interface{}); ok && len(arr) == 0 {
				delete(node, field)
			}
		}
		for _, child := range node {
			stripEmptyBundleCollections(child)
		}
	case []interface{}:
		for _, item := range node {
			stripEmptyBundleCollections(item)
		}
	}
}

// stripSignatureLevel drops the CqlToElmInfo signatureLevel on the bundle path.
// The CLI writes it unconditionally; the bundle shape omits it, so it cannot be
// compared there. It is still compared on every CLI-sourced path.
func stripSignatureLevel(m map[string]interface{}) {
	lib, ok := m["library"].(map[string]interface{})
	if !ok {
		return
	}
	anns, ok := lib["annotation"].([]interface{})
	if !ok {
		return
	}
	for _, a := range anns {
		if ann, ok := a.(map[string]interface{}); ok {
			if t, _ := ann["type"].(string); t == "CqlToElmInfo" {
				delete(ann, "signatureLevel")
			}
		}
	}
}

// stripEmptyAnnotations recursively removes empty annotation containers from the
// JSON tree:
//   - "annotation": []  on any ELM node (CQF emits it, echo-elm omits it)
//   - "t": []           inside Annotation objects (tag array, same)
//
// See normalizeJSON for why this is tolerated and what it is hiding.
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
