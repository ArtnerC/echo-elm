// parity_full — run full cqf-corpus parity suite (not committed to repo, temp tool).
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/artnerc/echo-elm/internal/parity"
)

func main() {
	translateFn := parity.CQFTranslateFunc()

	for _, version := range []string{parity.PinnedCQFVersion} {
		fmt.Printf("══ cqframework %s ══\n", version)
		cfg := parity.DefaultConfig(version)
		cfg.TagFilter = "cqf-corpus"

		results, err := parity.Run(cfg, translateFn)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for i := range results {
			r := &results[i]
			icon := "✓"
			if r.Status != parity.StatusMatch {
				icon = "✗"
			}
			fmt.Printf("  %s  %-50s  %s\n", icon, r.Fixture, r.Status)
			if r.Status == parity.StatusDifferJSON && r.Diff != "" {
				for i, l := range strings.Split(r.Diff, "\n") {
					if i >= 15 {
						fmt.Printf("         ... (%d more)\n", strings.Count(r.Diff, "\n")-15)
						break
					}
					fmt.Printf("         %s\n", l)
				}
			}
			if r.Error != "" {
				fmt.Printf("       err: %s\n", r.Error)
			}
			if r.Status == parity.StatusUpstreamError && r.UpstreamStderr != "" {
				lines := strings.Split(strings.TrimSpace(r.UpstreamStderr), "\n")
				if len(lines) > 5 {
					lines = lines[:5]
				}
				for _, l := range lines {
					fmt.Printf("       cqf: %s\n", l)
				}
			}
		}

		sum := parity.Summary(results)
		fmt.Printf("\n  match=%d differ-json=%d echo-error=%d upstream-error=%d\n\n",
			sum[parity.StatusMatch], sum[parity.StatusDifferJSON],
			sum[parity.StatusEchoError], sum[parity.StatusUpstreamError])
	}
}
