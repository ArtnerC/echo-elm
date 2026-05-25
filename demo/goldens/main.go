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
	goldensDir := "test/goldens"
	tmpA := filepath.Join(goldensDir, ".tmp-3.29.0")
	tmpB := filepath.Join(goldensDir, ".tmp-4.8.0")
	canonical := filepath.Join(goldensDir, "cqf")

	defer func() {
		os.RemoveAll(tmpA)
		os.RemoveAll(tmpB)
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
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  wrote %d golden file(s)\n", n)
	}

	fmt.Println("\nComparing versions...")
	diffs, err := parity.CompareVersionGoldens(goldensDir, ".tmp-3.29.0", ".tmp-4.8.0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "compare error: %v\n", err)
		os.Exit(1)
	}

	if len(diffs) > 0 {
		fmt.Printf("⚠  %d file(s) differ between 3.29.0 and 4.8.0:\n", len(diffs))
		for _, d := range diffs {
			fmt.Printf("  %s\n", d)
		}
		fmt.Println("\nVersions diverged — canonical goldens NOT updated.")
		os.Exit(1)
	}

	// Versions are identical — collapse to canonical set from 4.8.0 (latest).
	fmt.Println("✓ Versions identical — writing canonical set to test/goldens/cqf/")
	if err := os.RemoveAll(canonical); err != nil {
		fmt.Fprintf(os.Stderr, "remove %s: %v\n", canonical, err)
		os.Exit(1)
	}
	if err := copyDir(tmpB, canonical); err != nil {
		fmt.Fprintf(os.Stderr, "copy to cqf/: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Goldens written to test/goldens/cqf/ — commit to lock in the baseline.")
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
