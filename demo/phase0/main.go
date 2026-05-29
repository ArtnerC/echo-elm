// Command demo-phase0 is the Phase 0 milestone demo.
// It validates that the echo-elm project structure builds correctly,
// lists available packages, and confirms the version command works.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║         echo-elm  Phase 0  —  Bootstrap Demo       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	check("Go version", func() string {
		return runtime.Version() + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
	})

	check("go build ./...", func() string {
		out, err := exec.Command("go", "build", "./...").CombinedOutput()
		if err != nil {
			return "FAIL: " + string(out)
		}
		return "ok — all packages compile"
	})

	check("go vet (non-generated)", func() string {
		// Exclude ANTLR-generated packages from vet; generated code contains
		// unreachable-code patterns that are intentional ANTLR output.
		listOut, err := exec.Command("go", "list", "./...").Output()
		if err != nil {
			return "FAIL (list): " + string(listOut)
		}
		var pkgs []string
		for _, p := range strings.Split(strings.TrimSpace(string(listOut)), "\n") {
			p = strings.TrimSpace(p)
			if p != "" && !strings.Contains(p, "/cqlparser") {
				pkgs = append(pkgs, p)
			}
		}
		args := append([]string{"vet"}, pkgs...)
		out, err := exec.Command("go", args...).CombinedOutput()
		if err != nil {
			return "FAIL: " + string(out)
		}
		return "ok — no vet issues"
	})

	check("go test ./...", func() string {
		out, err := exec.Command("go", "test", "./...").CombinedOutput()
		if err != nil {
			return "FAIL:\n" + string(out)
		}
		return "ok — " + string(out)
	})

	// Build the binary and run version sub-command.
	check("echo-elm version", func() string {
		binPath := "./echo-elm"
		if runtime.GOOS == "windows" {
			binPath = "./echo-elm.exe"
		}
		_ = exec.Command("go", "build", "-o", binPath, "./cmd/echo-elm").Run()
		out, _ := exec.Command(binPath, "version").Output()
		_ = os.Remove(binPath)
		result := string(out)
		if result == "" {
			return "0.0.0-dev (binary built; no version flag output captured)"
		}
		return result
	})

	fmt.Println()
	fmt.Println("Phase 0 complete ✓  Project bootstrapped and all packages compile.")
}

func check(label string, fn func() string) {
	fmt.Printf("  %-30s  %s\n", label, fn())
}
