// showdiff reports where the two pinned CQF versions disagree for one golden.
//
// The goldens generator refuses to write a canonical set while 3.29.0 and 4.8.0
// differ, and marking a fixture versionDivergent in corpus.yaml suppresses that
// check — this shows what the disagreement actually is, so the marker can be
// justified rather than guessed at.
//
//	task parity:showdiff -- default/ra-measure/RAMeasure-1.0.0.json
//
// Both version trees are generated on first use and cached under
// test/goldens/.tmp-<version>/; delete those to force a regeneration.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/artnerc/echo-elm/internal/parity"
)

const defaultTarget = "default/ra-measure/RAMeasure-1.0.0.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "showdiff: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	target := defaultTarget
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	const goldensDir = "test/goldens"
	versions := []string{"3.29.0", "4.8.0"}
	paths := make([]string, len(versions))

	for i, ver := range versions {
		tmp := filepath.Join(goldensDir, ".tmp-"+ver)
		if _, statErr := os.Stat(tmp); os.IsNotExist(statErr) {
			fmt.Fprintf(os.Stderr, "Generating goldens for cqframework %s...\n", ver)
			n, genErr := parity.GenerateGoldensTo(parity.DefaultConfig(ver), tmp)
			if genErr != nil {
				return fmt.Errorf("generate %s: %w", ver, genErr)
			}
			fmt.Fprintf(os.Stderr, "  wrote %d file(s)\n", n)
		}
		paths[i] = filepath.Join(tmp, target)
	}

	older, err := os.ReadFile(paths[0])
	if err != nil {
		return fmt.Errorf("read %s: %w", paths[0], err)
	}
	newer, err := os.ReadFile(paths[1])
	if err != nil {
		return fmt.Errorf("read %s: %w", paths[1], err)
	}

	// Normalize exactly as the golden comparison does, so what shows up here is
	// the disagreement the generator refuses to collapse — not the localId and
	// annotation noise that normalization already accounts for.
	a := parity.NormalizeForGolden(string(older))
	b := parity.NormalizeForGolden(string(newer))

	fmt.Printf("%s\n", target)
	if a == b {
		fmt.Printf("  %s and %s agree once normalized\n", versions[0], versions[1])
		return nil
	}
	fmt.Printf("  - %s\n  + %s\n\n", versions[0], versions[1])
	fmt.Println(parity.SimpleDiff(a, b))
	return nil
}
