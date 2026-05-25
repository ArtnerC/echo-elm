package parity_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/parity"
)

// TestGoldenCorpus runs every corpus fixture × every applicable option profile
// through echo-elm and compares normalized JSON against the committed CQF reference
// goldens in test/goldens/cqf/<profile>/<fixture>.json.
//
// This test does NOT invoke the Java CQF CLI — it uses pre-generated golden files.
//
// Goldens are produced/refreshed by running:
//
// task parity:goldens
//
// If goldens for a profile are not present the sub-test is skipped.
func TestGoldenCorpus(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "test", "corpus", "cqframework")
	goldensDir := filepath.Join("..", "..", "test", "goldens", "cqf")

	corpus, err := parity.LoadCorpus(corpusDir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}

	if _, statErr := os.Stat(goldensDir); os.IsNotExist(statErr) {
		t.Skip("no canonical goldens — run: task parity:goldens")
	}

	profileNames := sortedKeys(corpus.OptionProfiles)

	for _, profileName := range profileNames {
		profileName := profileName
		profile := corpus.OptionProfiles[profileName]

		profileDir := filepath.Join(goldensDir, profileName)
		if _, statErr := os.Stat(profileDir); os.IsNotExist(statErr) {
			t.Run(profileName, func(t *testing.T) {
				t.Skipf("no goldens for profile %s — run: task parity:goldens", profileName)
			})
			continue
		}

		t.Run(profileName, func(t *testing.T) {
			if profile.PendingImplementation {
				t.Skipf("profile %s pending implementation — not yet emitted by echo-elm", profileName)
			}
			translateFn := parity.ProfileTranslateFunc(profile)
			for _, fix := range corpus.Fixtures {
				fix := fix

				// Skip fixtures that exclude this profile.
				fixProfiles := fix.ProfileNames(corpus.OptionProfiles)
				if !contains(fixProfiles, profileName) {
					continue
				}

				name := strings.TrimSuffix(strings.ReplaceAll(fix.Path, "/", "_"), ".cql")
				t.Run(name, func(t *testing.T) {
					cqlPath := filepath.Join(corpusDir, fix.Path)
					goldenPath := filepath.Join(profileDir,
						strings.TrimSuffix(fix.Path, ".cql")+".json")

					goldenBytes, err := os.ReadFile(goldenPath)
					if err != nil {
						if fix.ExpectedStatus == "failure" || fix.ExpectedStatus == "upstream-error" {
							t.Skipf("no golden (expected: %s)", fix.ExpectedStatus)
						}
						t.Fatalf("read golden %s: %v\n  run: task parity:goldens", goldenPath, err)
					}

					echoBytes, err := translateFn(cqlPath)
					if err != nil {
						t.Fatalf("translate %s: %v", cqlPath, err)
					}

					gotJSON := parity.NormalizeForGolden(string(echoBytes))
					wantJSON := parity.NormalizeForGolden(string(goldenBytes))

					if gotJSON != wantJSON {
						diff := parity.SimpleDiff(wantJSON, gotJSON)
						t.Errorf("golden mismatch %s %s:\n%s",
							profileName, fix.Path, diff)
					}
				})
			}
		})
	}
}

// TestGoldenJSONWellFormed verifies every committed golden is valid JSON.
func TestGoldenJSONWellFormed(t *testing.T) {
	goldensDir := filepath.Join("..", "..", "test", "goldens", "cqf")
	if _, err := os.Stat(goldensDir); os.IsNotExist(err) {
		t.Skip("no goldens committed yet — run: task parity:goldens")
	}

	err := filepath.Walk(goldensDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var v interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("invalid JSON in %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		t.Errorf("malformed golden: %v", err)
	}
}

func sortedKeys(m map[string]parity.OptionProfile) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
