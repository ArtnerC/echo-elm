package parity_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/pkg/bundle"
)

// TestRunWithRefDir exercises the --ref-dir path: parity compares against ELM
// already on disk rather than invoking the CQF JAR, so it works in an
// environment without Java. The committed goldens stand in for the reference
// set, which is exactly the layout --ref-dir expects.
func TestRunWithRefDir(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	cfg := parity.Config{
		CorpusDir:     filepath.Join(repoRoot, "test", "corpus", "cqframework"),
		RefDir:        filepath.Join(repoRoot, "test", "goldens", "cqf"),
		ProfileFilter: "default",
		// ToolsDir deliberately left empty: nothing may shell out to the JAR.
	}

	results, err := parity.Run(cfg, parity.CQFTranslateFunc())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no fixtures ran")
	}

	matched := 0
	for i := range results {
		r := &results[i]
		switch r.Status {
		case parity.StatusMatch:
			matched++
		case parity.StatusSkipped:
			// Fixtures the corpus expects to fail have no reference ELM.
		default:
			t.Errorf("%s [%s]: %s\n%s", r.Fixture, r.Profile, r.Status, r.Diff)
		}
	}
	if matched == 0 {
		t.Fatal("no fixture matched its reference ELM")
	}
	t.Logf("%d/%d fixtures matched against --ref-dir", matched, len(results))
}

// TestRunWithRefDirMissingReference reports a missing reference as an upstream
// error rather than a match, so a partial reference set cannot pass silently.
func TestRunWithRefDirMissingReference(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	cfg := parity.Config{
		CorpusDir:     filepath.Join(repoRoot, "test", "corpus", "cqframework"),
		RefDir:        t.TempDir(), // empty: nothing to compare against
		ProfileFilter: "default",
	}

	results, err := parity.Run(cfg, parity.CQFTranslateFunc())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for i := range results {
		if results[i].Status == parity.StatusMatch {
			t.Fatalf("%s reported a match with no reference ELM present", results[i].Fixture)
		}
	}
}

// TestRunWithLibDir checks that --lib-dir extends include resolution beyond the
// fixture's own directory, which is what a bundle extracted to one flat
// directory needs.
func TestRunWithLibDir(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	corpusDir := filepath.Join(repoRoot, "test", "corpus", "cqframework")

	// Move a library out of the including fixture's directory so that resolution
	// can only succeed through LibDir.
	libDir := t.TempDir()
	shared, err := os.ReadFile(filepath.Join(corpusDir, "elm-nodes", "SharedTerminology.cql"))
	if err != nil {
		t.Fatalf("read shared library: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "SharedTerminology.cql"), shared, 0o644); err != nil {
		t.Fatalf("write shared library: %v", err)
	}

	fixture := filepath.Join(t.TempDir(), "CrossLibraryTerminology.cql")
	src, err := os.ReadFile(filepath.Join(corpusDir, "elm-nodes", "CrossLibraryTerminology.cql"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(fixture, src, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	corpus, err := parity.LoadCorpus(corpusDir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	profile := corpus.OptionProfiles["default"]

	// Without the extra search directory the include cannot resolve, so the
	// qualified reference degrades to a Property access.
	plain, err := parity.ProfileTranslateFunc(profile)(fixture)
	if err != nil {
		t.Fatalf("translate without lib dir: %v", err)
	}
	if strings.Contains(string(plain), `"type": "ValueSetRef"`) {
		t.Fatal("reference resolved without --lib-dir; the test proves nothing")
	}

	got, err := parity.ProfileTranslateFuncWithLibDir(profile, libDir)(fixture)
	if err != nil {
		t.Fatalf("translate with lib dir: %v", err)
	}
	if !strings.Contains(string(got), `"type": "ValueSetRef"`) {
		t.Error("qualified value set reference did not resolve through --lib-dir")
	}
}

// TestMaterializeBundle covers `parity --bundle`: a bundle is laid out as a
// corpus and compared against the ELM it already carries, with no CQF JAR and no
// manual extraction step.
func TestMaterializeBundle(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	corpusDir := filepath.Join(repoRoot, "test", "corpus", "cqframework", "elm-nodes")
	goldenDir := filepath.Join(repoRoot, "test", "goldens", "cqf", "default", "elm-nodes")

	// Two libraries where one includes the other, so the run also proves includes
	// resolve across the extracted corpus.
	doc := map[string]any{
		"resourceType": "Bundle",
		"type":         "collection",
		"entry": []any{
			bundleEntry(t, "SharedTerminology", "1.0",
				filepath.Join(corpusDir, "SharedTerminology.cql"),
				filepath.Join(goldenDir, "SharedTerminology.json")),
			bundleEntry(t, "CrossLibraryTerminology", "1.0",
				filepath.Join(corpusDir, "CrossLibraryTerminology.cql"),
				filepath.Join(goldenDir, "CrossLibraryTerminology.json")),
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}
	b, err := bundle.Load(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	cfg, err := parity.MaterializeBundle(b, t.TempDir(), "default")
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	corpus, err := parity.LoadCorpus(cfg.CorpusDir)
	if err != nil {
		t.Fatalf("load generated corpus: %v", err)
	}
	results, err := parity.Run(cfg, parity.ProfileTranslateFuncWithLibDir(
		corpus.OptionProfiles["default"], cfg.LibDir))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for i := range results {
		if results[i].Status != parity.StatusMatch {
			t.Errorf("%s: %s\n%s", results[i].Fixture, results[i].Status, results[i].Diff)
		}
	}
}

// bundleEntry builds one Library entry from files on disk.
func bundleEntry(t *testing.T, name, version, cqlPath, elmPath string) map[string]any {
	t.Helper()
	cql, err := os.ReadFile(cqlPath)
	if err != nil {
		t.Fatalf("read %s: %v", cqlPath, err)
	}
	elm, err := os.ReadFile(elmPath)
	if err != nil {
		t.Fatalf("read %s: %v", elmPath, err)
	}
	return map[string]any{
		"resource": map[string]any{
			"resourceType": "Library",
			"name":         name,
			"version":      version,
			"content": []any{
				map[string]any{"contentType": bundle.ContentTypeCQL,
					"data": base64.StdEncoding.EncodeToString(cql)},
				map[string]any{"contentType": bundle.ContentTypeELMJSON,
					"data": base64.StdEncoding.EncodeToString(elm)},
			},
		},
	}
}

// TestFirelyTargetTypesEveryNode runs the discriminated form over the whole
// corpus. The property a schema-driven deserializer needs is that no node with
// structure is left without a "type"; a single fixture cannot show that holds
// across every construct the translator emits.
func TestFirelyTargetTypesEveryNode(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "test", "corpus", "cqframework")
	corpus, err := parity.LoadCorpus(corpusDir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	translate := parity.ProfileTranslateFunc(corpus.OptionProfiles["default"])

	checked := 0
	for _, fix := range corpus.Fixtures {
		if fix.ExpectedStatus != "success" {
			continue
		}
		fix := fix
		t.Run(strings.TrimSuffix(fix.Path, ".cql"), func(t *testing.T) {
			plain, err := translate(filepath.Join(corpusDir, fix.Path))
			if err != nil {
				t.Fatalf("translate: %v", err)
			}
			discriminated, err := elm.AddTypeDiscriminators(plain)
			if err != nil {
				t.Fatalf("add discriminators: %v", err)
			}
			var doc map[string]any
			if err := json.Unmarshal(discriminated, &doc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			for _, path := range untypedNodes(doc["library"], "library") {
				t.Errorf("no type discriminator at %s", path)
			}
		})
		checked++
	}
	if checked == 0 {
		t.Fatal("no fixtures checked")
	}
}

// untypedNodes returns the paths of nodes that have structure but no "type".
func untypedNodes(v any, path string) []string {
	var bare []string
	switch node := v.(type) {
	case map[string]any:
		hasChildren := false
		for _, child := range node {
			switch child.(type) {
			case map[string]any, []any:
				hasChildren = true
			}
		}
		if _, typed := node["type"]; !typed && hasChildren {
			// Annotation s-trees are free-form source spans, not ELM nodes.
			if !strings.Contains(path, ".annotation") {
				bare = append(bare, path)
			}
		}
		for k, child := range node {
			bare = append(bare, untypedNodes(child, path+"."+k)...)
		}
	case []any:
		for i, item := range node {
			bare = append(bare, untypedNodes(item, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return bare
}
