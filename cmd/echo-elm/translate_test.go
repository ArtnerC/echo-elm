package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, n := range names {
		p := filepath.Join(root, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("library X"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCollectInputsWalksDirectories pins the CQF CLI's directory behavior: every
// .cql file beneath the directory, recursively and in lexical order, whatever
// the extension's case, and nothing else.
func TestCollectInputsWalksDirectories(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "b.cql", "a.CQL", "notes.txt", "sub/c.cql", "sub/c.json")

	files, many, err := collectInputs([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(files))
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		got = append(got, filepath.ToSlash(rel))
	}
	if want := []string{"a.CQL", "b.cql", "sub/c.cql"}; !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if !many {
		t.Error("a directory input must be a multi-file run")
	}
}

func TestCollectInputsSingleFileAndDuplicates(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "a.cql")
	a := filepath.Join(root, "a.cql")

	files, many, err := collectInputs([]string{a})
	if err != nil || many || len(files) != 1 {
		t.Fatalf("single file: files=%v many=%v err=%v", files, many, err)
	}
	// The same file named twice, or by a directory too, is translated once.
	files, _, err = collectInputs([]string{a, root})
	if err != nil || len(files) != 1 {
		t.Fatalf("duplicates: files=%v err=%v", files, err)
	}
}

func TestCollectInputsErrors(t *testing.T) {
	if _, _, err := collectInputs([]string{filepath.Join(t.TempDir(), "missing.cql")}); err == nil {
		t.Error("a missing input must be an error")
	}
	if _, _, err := collectInputs([]string{t.TempDir()}); err == nil || !strings.Contains(err.Error(), "no .cql files") {
		t.Errorf("an empty directory must be an error, got %v", err)
	}
}

func TestPlanOutputs(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "x/A.cql", "y/B.cql", "y/A.cql")
	xa, yb, ya := filepath.Join(root, "x", "A.cql"), filepath.Join(root, "y", "B.cql"), filepath.Join(root, "y", "A.cql")

	// No --output: each next to its input.
	outs, err := planOutputs([]string{xa, yb}, true, "", "JSON")
	if err != nil {
		t.Fatal(err)
	}
	if outs[0] != filepath.Join(root, "x", "A.json") || outs[1] != filepath.Join(root, "y", "B.json") {
		t.Errorf("next-to-input outputs = %v", outs)
	}

	// A missing --output directory is created, and everything goes into it.
	outDir := filepath.Join(root, "out")
	outs, err = planOutputs([]string{xa, yb}, true, outDir, "XML")
	if err != nil {
		t.Fatal(err)
	}
	if outs[0] != filepath.Join(outDir, "A.xml") || outs[1] != filepath.Join(outDir, "B.xml") {
		t.Errorf("--output dir outputs = %v", outs)
	}
	if info, err := os.Stat(outDir); err != nil || !info.IsDir() {
		t.Errorf("--output directory not created: %v", err)
	}

	// Two inputs with one base name cannot share an output directory.
	if _, err := planOutputs([]string{xa, ya}, true, outDir, "JSON"); err == nil || !strings.Contains(err.Error(), "both be written") {
		t.Errorf("colliding outputs must be an error, got %v", err)
	}

	// A file is not a directory.
	if _, err := planOutputs([]string{xa, yb}, true, xa, "JSON"); err == nil {
		t.Error("an existing file as --output for many inputs must be an error")
	}

	// A single input keeps the old rules: --output may name the file.
	outs, err = planOutputs([]string{xa}, false, filepath.Join(root, "custom.json"), "JSON")
	if err != nil || outs[0] != filepath.Join(root, "custom.json") {
		t.Errorf("single-file --output = %v, %v", outs, err)
	}
}
