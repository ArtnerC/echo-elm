package parity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/parity"
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
