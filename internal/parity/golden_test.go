package parity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/parity"
)

// TestGoldenCorpus runs every fixture in the corpus through echo-elm in
// CQF-compatible mode and compares the JSON against the upstream CQFramework
// reference output (the .json file next to the .cql file). This test does NOT
// invoke the Java cqframework CLI — it uses the committed reference outputs as
// golden files. This locks in current CQF parity as a fast unit test.
//
// Goldens are produced/refreshed by running:
//
//	go run ./demo/parity_full
//
// (which DOES invoke the Java CLI). If a fixture's expectedStatus is "failure"
// it has no JSON output and is skipped here.
func TestGoldenCorpus(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "test", "corpus", "cqframework")
	corpus, err := parity.LoadCorpus(corpusDir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}

	translateFn := parity.CQFTranslateFunc()

	for _, fix := range corpus.Fixtures {
		fix := fix
		name := strings.TrimSuffix(strings.ReplaceAll(fix.Path, "/", "_"), ".cql")
		t.Run(name, func(t *testing.T) {
			cqlPath := filepath.Join(corpusDir, fix.Path)
			goldenPath := strings.TrimSuffix(cqlPath, ".cql") + ".json"

			goldenBytes, err := os.ReadFile(goldenPath)
			if err != nil {
				if fix.ExpectedStatus == "failure" || fix.ExpectedStatus == "upstream-error" {
					t.Skipf("no golden (expected: %s)", fix.ExpectedStatus)
				}
				t.Fatalf("read golden %s: %v", goldenPath, err)
			}

			echoBytes, err := translateFn(cqlPath)
			if err != nil {
				t.Fatalf("translate %s: %v", cqlPath, err)
			}

			gotJSON := parity.NormalizeForGolden(string(echoBytes))
			wantJSON := parity.NormalizeForGolden(string(goldenBytes))

			if gotJSON != wantJSON {
				diff := parity.SimpleDiff(wantJSON, gotJSON)
				t.Errorf("golden mismatch for %s:\n%s", fix.Path, diff)
			}
		})
	}
}
