// goldens — generate committed CQF reference golden files.
//
// Runs both CQF versions (3.29.0 and 4.8.0), compares their output, and writes
// the canonical collapsed set to test/goldens/cqf/ if they are identical.
//
// Run from the repo root:
//
//	task parity:goldens
//
// Prerequisites: parity:jdk + parity:jar must have been run first so that the
// launchers exist under tools/cqframework/<version>/.
//
// The generated files are written to test/goldens/cqf/ and SHOULD be committed.
// They serve as the reference baseline for TestGoldenCorpus.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/artnerc/echo-elm/internal/parity"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	goldensDir := "test/goldens"
	tmpA := filepath.Join(goldensDir, ".tmp-3.29.0")
	tmpB := filepath.Join(goldensDir, ".tmp-4.8.0")
	canonical := filepath.Join(goldensDir, "cqf")

	defer func() {
		_ = os.RemoveAll(tmpA)
		_ = os.RemoveAll(tmpB)
	}()

	versions := []struct {
		ver string
		dir string
	}{
		{"3.29.0", tmpA},
		{"4.8.0", tmpB},
	}

	for _, v := range versions {
		fmt.Printf("Generating goldens for cqframework %s...\n", v.ver)
		cfg := parity.DefaultConfig(v.ver)
		n, err := parity.GenerateGoldensTo(cfg, v.dir)
		if err != nil {
			return fmt.Errorf("  error: %w", err)
		}
		fmt.Printf("  wrote %d golden file(s)\n", n)
	}

	// Fixtures marked versionDivergent in corpus.yaml are known to differ
	// between the pinned upstream versions; they take the newest version's output.
	corpus, err := parity.LoadCorpus(parity.DefaultConfig("4.8.0").CorpusDir)
	if err != nil {
		return fmt.Errorf("load corpus: %w", err)
	}
	divergent := make(map[string]bool)
	for _, fix := range corpus.Fixtures {
		if fix.VersionDivergent {
			divergent[fix.Path] = true
		}
	}

	fmt.Println("\nComparing versions...")
	diffs, err := parity.CompareVersionGoldens(goldensDir, ".tmp-3.29.0", ".tmp-4.8.0", divergent)
	if err != nil {
		return fmt.Errorf("compare error: %w", err)
	}

	if len(diffs) > 0 {
		fmt.Printf("⚠  %d file(s) differ between 3.29.0 and 4.8.0:\n", len(diffs))
		for _, d := range diffs {
			fmt.Printf("  %s\n", d)
		}
		return fmt.Errorf("versions diverged — canonical goldens NOT updated")
	}

	if len(divergent) > 0 {
		fmt.Printf("  (%d fixture(s) marked versionDivergent — taking 4.8.0 output)\n", len(divergent))
	}

	// Versions agree everywhere else — collapse to canonical set from 4.8.0 (latest).
	fmt.Println("✓ Versions identical — writing canonical set to test/goldens/cqf/")
	if err := os.RemoveAll(canonical); err != nil {
		return fmt.Errorf("remove %s: %w", canonical, err)
	}
	if err := copyDir(tmpB, canonical); err != nil {
		return fmt.Errorf("copy to cqf/: %w", err)
	}

	fmt.Println("Goldens written to test/goldens/cqf/ — commit to lock in the baseline.")
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
