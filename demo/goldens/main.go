// goldens — generate committed CQF reference golden files for both supported
// CQFramework versions (3.29.0 and 4.8.0).
//
// Run from the repo root:
//
//	task parity:goldens
//
// Prerequisites: parity:jdk + parity:jar must have been run first so that the
// launchers exist under tools/cqframework/<version>/.
//
// The generated files are written to test/goldens/<version>/ and SHOULD be
// committed. They serve as the reference baseline for TestGoldenCorpus.
package main

import (
	"fmt"
	"os"

	"github.com/artnerc/echo-elm/internal/parity"
)

func main() {
	goldensDir := "test/goldens"

	for _, version := range []string{"3.29.0", "4.8.0"} {
		fmt.Printf("Generating goldens for cqframework %s...\n", version)
		cfg := parity.DefaultConfig(version)

		n, err := parity.GenerateGoldens(cfg, goldensDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  wrote %d golden file(s) → %s/%s/\n", n, goldensDir, version)
	}

	fmt.Println("\nGoldens generated. Commit test/goldens/ to lock in the baseline.")
}
