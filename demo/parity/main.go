// Parity demo — shows echo-elm results alongside cqframework 3.29.0 and 4.8.0 outputs.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/artnerc/echo-elm/internal/parity"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

func main() {
	box("echo-elm  Parity Demo  —  CQFramework comparison")

	// Verify tooling is in place.
	for _, v := range []string{"3.29.0", "4.8.0"} {
		launcher := launcherPath(v)
		if _, err := os.Stat(launcher); err != nil {
			fmt.Printf("  ✗ cqframework %s launcher not found: %s\n", v, launcher)
			fmt.Println("    Run: task parity:jdk && task parity:jar")
			os.Exit(1)
		}
		fmt.Printf("  ✓ cqframework %s launcher found\n", v)
	}
	fmt.Println()

	for _, version := range []string{"3.29.0", "4.8.0"} {
		fmt.Printf("── Parity vs cqframework %s ──\n\n", version)

		cfg := parity.DefaultConfig(version)
		cfg.TagFilter = "smoke"

		results, err := parity.Run(cfg, echoTranslate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		for _, r := range results {
			icon := statusIcon(r.Status)
			fmt.Printf("  %s  %-40s  %v\n", icon, r.Fixture, r.Status)
			if r.Status == parity.StatusDifferJSON && r.Diff != "" {
				// Show at most 8 diff lines to keep demo readable.
				lines := strings.Split(r.Diff, "\n")
				for i, l := range lines {
					if i >= 8 {
						fmt.Printf("         ... (%d more diff lines)\n", len(lines)-8)
						break
					}
					fmt.Printf("         %s\n", l)
				}
			}
			if r.Error != "" {
				fmt.Printf("     err: %s\n", r.Error)
			}
		}

		summary := parity.Summary(results)
		fmt.Printf("\n  Summary: match=%d  differ-json=%d  error=%d\n\n",
			summary[parity.StatusMatch],
			summary[parity.StatusDifferJSON],
			summary[parity.StatusEchoError]+summary[parity.StatusUpstreamError])
	}

	fmt.Println("Parity demo complete ✓")
	fmt.Println("  (Expression bodies will differ until Phase 4 expression builder is complete)")
}

// echoTranslate runs echo-elm on a CQL file and returns the JSON ELM.
func echoTranslate(cqlPath string) ([]byte, error) {
	data, err := os.ReadFile(cqlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cqlPath, err)
	}

	sourceName := filepath.Base(cqlPath)
	result, err := echoelm.Translate(data, sourceName, echoelm.WithAnnotations(true))
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(map[string]interface{}{"library": result.Library}, "", "   ")
}

func launcherPath(version string) string {
	return filepath.Join("tools", "cqframework", version, "run.bat")
}

func statusIcon(s parity.Status) string {
	switch s {
	case parity.StatusMatch:
		return "✓"
	case parity.StatusDifferJSON:
		return "≠"
	case parity.StatusEchoError, parity.StatusUpstreamError:
		return "✗"
	default:
		return "?"
	}
}

func box(title string) {
	pad := strings.Repeat("═", len(title)+4)
	fmt.Printf("╔%s╗\n║  %s  ║\n╚%s╝\n\n", pad, title, pad)
}
