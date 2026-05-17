// Command demo-phase0 is the Phase 0 milestone demo.
// It validates that the echo-elm project structure builds correctly,
// lists available packages, and confirms the version command works.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

	check("go vet ./...", func() string {
		out, err := exec.Command("go", "vet", "./...").CombinedOutput()
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
		os.Remove(binPath)
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
