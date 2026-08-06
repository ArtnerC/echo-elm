package parity_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/parity"
)

// TestAnnotationTextInvariant backs the one reduction normalizeJSON makes to
// annotation s-trees. Parity compares the concatenated leaf text rather than
// CQF's per-grammar-rule segmentation, which is only sound if that text really
// is the definition's source. This checks it directly against the .cql file,
// independently of anything CQF produced.
func TestAnnotationTextInvariant(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "test", "corpus", "cqframework")
	corpus, err := parity.LoadCorpus(corpusDir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	profile, ok := corpus.OptionProfiles["annotations"]
	if !ok {
		t.Skip("no annotations profile")
	}
	translate := parity.ProfileTranslateFunc(profile)

	for _, fix := range corpus.Fixtures {
		if fix.ExpectedStatus != "success" {
			continue
		}
		if !contains(fix.ProfileNames(corpus.OptionProfiles), "annotations") {
			continue
		}
		t.Run(strings.TrimSuffix(fix.Path, ".cql"), func(t *testing.T) {
			cqlPath := filepath.Join(corpusDir, fix.Path)
			source, err := os.ReadFile(cqlPath)
			if err != nil {
				t.Fatalf("read %s: %v", cqlPath, err)
			}
			elmJSON, err := translate(cqlPath)
			if err != nil {
				t.Fatalf("translate %s: %v", cqlPath, err)
			}

			var doc struct {
				Library struct {
					Statements struct {
						Def []struct {
							Name       string            `json:"name"`
							Annotation []json.RawMessage `json:"annotation"`
						} `json:"def"`
					} `json:"statements"`
				} `json:"library"`
			}
			if err := json.Unmarshal(elmJSON, &doc); err != nil {
				t.Fatalf("unmarshal ELM: %v", err)
			}

			normalizedSource := strings.ReplaceAll(string(source), "\r\n", "\n")
			for _, def := range doc.Library.Statements.Def {
				for _, raw := range def.Annotation {
					text, ok := annotationText(raw)
					if !ok || text == "" {
						continue
					}
					if !strings.Contains(normalizedSource, text) {
						t.Errorf("statement %q: annotation text is not a slice of the source\n  got: %q",
							def.Name, truncate(text, 200))
					}
				}
			}
		})
	}
}

// annotationText returns the concatenated leaf text of an Annotation's s-tree.
func annotationText(raw json.RawMessage) (string, bool) {
	var node map[string]interface{}
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", false
	}
	if typ, _ := node["type"].(string); typ != "Annotation" {
		return "", false
	}
	s, ok := node["s"]
	if !ok {
		return "", false
	}
	var sb strings.Builder
	var walk func(interface{})
	walk = func(x interface{}) {
		switch v := x.(type) {
		case map[string]interface{}:
			switch val := v["value"].(type) {
			case string:
				sb.WriteString(val)
			case []interface{}:
				for _, item := range val {
					if str, ok := item.(string); ok {
						sb.WriteString(str)
					}
				}
			}
			if child, ok := v["s"]; ok {
				walk(child)
			}
		case []interface{}:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(s)
	return sb.String(), true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
