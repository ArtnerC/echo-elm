// parity_diff — print full diff for a single fixture
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/artnerc/echo-elm/internal/parity"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: parity_diff <fixture-substring>")
		os.Exit(1)
	}
	target := os.Args[1]
	translateFn := parity.CQFTranslateFunc()
	cfg := parity.DefaultConfig(parity.PinnedCQFVersion)
	cfg.TagFilter = "cqf-corpus"
	results, err := parity.Run(cfg, translateFn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i := range results {
		r := &results[i]
		if !strings.Contains(r.Fixture, target) {
			continue
		}
		fmt.Printf("== %s (%s) ==\n", r.Fixture, r.Status)
		if r.Diff != "" {
			fmt.Println(r.Diff)
		}
		if r.UpstreamJSON != "" {
			_ = os.WriteFile("_up.json", []byte(r.UpstreamJSON), 0644)
			_ = os.WriteFile("_echo.json", []byte(r.EchoJSON), 0644)
		}
		if r.Error != "" {
			fmt.Println("ERR:", r.Error)
		}
	}
}
