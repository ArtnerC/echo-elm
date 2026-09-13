// goldens — generate committed CQF reference golden files.
//
// Runs the pinned CQF version over the corpus and writes the result to
// test/goldens/cqf/.
//
// Run from the repo root:
//
//	task parity:goldens
//
// Prerequisites: parity:jdk + parity:jar must have been run first so that the
// launcher exists under tools/cqframework/<version>/.
//
// The generated files are written to test/goldens/cqf/ and SHOULD be committed.
// They serve as the reference baseline for TestGoldenCorpus.
//
// echo-elm used to pin two CQF versions and collapse their goldens into one set,
// which only worked because the harness normalized away everywhere they
// disagreed. Those normalizations were load-bearing for the collapse and were
// hiding real differences, so the pin is now a single version — see
// parity.PinnedCQFVersion.
package main

import (
	"fmt"
	"os"

	"github.com/artnerc/echo-elm/internal/parity"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	canonical := "test/goldens/cqf"

	fmt.Printf("Generating goldens for cqframework %s...\n", parity.PinnedCQFVersion)
	if err := os.RemoveAll(canonical); err != nil {
		return fmt.Errorf("remove %s: %w", canonical, err)
	}

	cfg := parity.DefaultConfig(parity.PinnedCQFVersion)
	n, err := parity.GenerateGoldensTo(cfg, canonical)
	if err != nil {
		return fmt.Errorf("  error: %w", err)
	}
	fmt.Printf("  wrote %d golden file(s)\n", n)

	fmt.Println("Goldens written to test/goldens/cqf/ — commit to lock in the baseline.")
	return nil
}
