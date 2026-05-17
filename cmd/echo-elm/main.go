// Command echo-elm is a CQL→ELM translator.
package main

import (
	"fmt"
	"os"
)

// Version is set at build time via -ldflags.
var Version = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("echo-elm %s\n", Version)
		return
	}
	fmt.Fprintf(os.Stderr, "echo-elm %s\n", Version)
	fmt.Fprintln(os.Stderr, "Usage: echo-elm <command> [flags]")
	fmt.Fprintln(os.Stderr, "  translate   Translate CQL to ELM")
	fmt.Fprintln(os.Stderr, "  cqf         CQFramework-compatible subcommands")
	fmt.Fprintln(os.Stderr, "  ui          Start local workbench UI")
	fmt.Fprintln(os.Stderr, "  mcp         Start MCP server")
	fmt.Fprintln(os.Stderr, "  version     Print version")
	os.Exit(1)
}
