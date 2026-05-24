package parity_test

import (
"os"
"path/filepath"
"strings"
"testing"

"github.com/artnerc/echo-elm/internal/parity"
)

// TestGoldenCorpus runs every corpus fixture through echo-elm in CQF-compatible
// mode and compares the JSON against the committed CQFramework reference outputs
// in test/goldens/<version>/. This test does NOT invoke the Java CQF CLI — it
// uses pre-generated golden files as the reference.
//
// Goldens are produced/refreshed by running:
//
//task parity:goldens
//
// which DOES invoke the Java CLI for each supported CQF version. If goldens for
// a version are not present the sub-test is skipped with an informative message.
func TestGoldenCorpus(t *testing.T) {
corpusDir := filepath.Join("..", "..", "test", "corpus", "cqframework")
goldensDir := filepath.Join("..", "..", "test", "goldens")

corpus, err := parity.LoadCorpus(corpusDir)
if err != nil {
t.Fatalf("load corpus: %v", err)
}

translateFn := parity.CQFTranslateFunc()

// Test against every CQF version for which committed goldens exist.
for _, version := range []string{"3.29.0", "4.8.0"} {
version := version
versionDir := filepath.Join(goldensDir, version)

if _, statErr := os.Stat(versionDir); os.IsNotExist(statErr) {
t.Run("cqf-"+version, func(t *testing.T) {
t.Skipf("no goldens for cqf %s — run: task parity:goldens", version)
})
continue
}

t.Run("cqf-"+version, func(t *testing.T) {
for _, fix := range corpus.Fixtures {
fix := fix
name := strings.TrimSuffix(strings.ReplaceAll(fix.Path, "/", "_"), ".cql")
t.Run(name, func(t *testing.T) {
cqlPath := filepath.Join(corpusDir, fix.Path)
goldenPath := filepath.Join(versionDir,
strings.TrimSuffix(fix.Path, ".cql")+".json")

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
t.Errorf("golden mismatch for cqf-%s %s:\n%s", version, fix.Path, diff)
}
})
}
})
}
}
