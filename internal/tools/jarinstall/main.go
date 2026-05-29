// jarinstall downloads the cqframework cql-to-elm-cli JARs and their full
// transitive dependency classpaths using Coursier, then writes launcher
// scripts into tools/cqframework/<version>/.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// cqfVersions lists the cqframework cql-to-elm-cli coordinates to install.
var cqfVersions = []cqfVersion{
	{
		Slug:       "3.29.0",
		Coordinate: "info.cqframework:cql-to-elm-cli:3.29.0",
		MainClass:  "org.cqframework.cql.cql2elm.cli.Main",
	},
	{
		Slug:       "4.8.0",
		Coordinate: "org.cqframework:cql-to-elm-cli:4.8.0",
		MainClass:  "org.cqframework.cql.cql2elm.cli.Main",
	},
}

type cqfVersion struct {
	Slug       string
	Coordinate string
	MainClass  string
}

const (
	coursierDir        = "tools/coursier"
	cqframeworkBaseDir = "tools/cqframework"
)

func main() {
	fmt.Println("=== echo-elm jarinstall ===")

	csBin, err := ensureCoursier()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  coursier: %s\n", csBin)

	javaBin := findJava()
	fmt.Printf("  java:     %s\n", javaBin)

	for _, v := range cqfVersions {
		fmt.Printf("\n── %s (%s) ──\n", v.Slug, v.Coordinate)
		if err := installVersion(javaBin, csBin, v); err != nil {
			fmt.Fprintf(os.Stderr, "error installing %s: %v\n", v.Slug, err)
			os.Exit(1)
		}
	}

	fmt.Println("\n✓ All cqframework CLI versions installed.")
	fmt.Println("  Run with: task parity:run")
}

// findJava returns the java binary path: prefers the one written by jdkinstall,
// then falls back to whatever is on PATH.
func findJava() string {
	if data, err := os.ReadFile("tools/jdk/jdk.env"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if after, ok := strings.CutPrefix(line, "JAVA_BIN="); ok {
				p := strings.TrimSpace(after)
				if p != "" {
					return p
				}
			}
		}
	}
	if p, err := exec.LookPath("java"); err == nil {
		return p
	}
	return "java"
}

// ensureCoursier returns path to the coursier bootstrap JAR, downloading if needed.
func ensureCoursier() (string, error) {
	if err := os.MkdirAll(coursierDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", coursierDir, err)
	}

	// On non-Windows: check for native cs on PATH first.
	if runtime.GOOS != "windows" {
		if path, err := exec.LookPath("cs"); err == nil {
			return path, nil
		}
	}

	// Use the JVM bootstrap launcher — works on all platforms via `java -jar`.
	localBin := filepath.Join(coursierDir, "cs-bootstrap.jar")
	if _, err := os.Stat(localBin); err == nil {
		fmt.Printf("  using cached coursier at %s\n", localBin)
		return localBin, nil
	}

	// Download the Coursier JVM bootstrap launcher (~167KB).
	dlURL := "https://github.com/coursier/launchers/raw/master/coursier"
	fmt.Printf("  downloading coursier bootstrap from %s…\n", dlURL)
	if err := downloadFile(dlURL, localBin); err != nil {
		return "", fmt.Errorf("download coursier: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(localBin, 0o755)
	}
	fmt.Printf("  coursier saved to %s\n", localBin)
	return localBin, nil
}

func installVersion(javaBin, csBin string, v cqfVersion) error {
	dir := filepath.Join(cqframeworkBaseDir, v.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	classpathFile := filepath.Join(dir, "classpath.txt")

	// Check if already installed.
	if _, err := os.Stat(classpathFile); err == nil {
		fmt.Printf("  ✓ already installed (delete %s to reinstall)\n", classpathFile)
		return nil
	}

	// Use coursier fetch --classpath to get the full dependency classpath.
	fmt.Printf("  resolving %s…\n", v.Coordinate)

	// Use project-local coursier cache.
	cacheDir := filepath.Join(coursierDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("mkdir cache: %w", err)
	}

	// Determine how to invoke coursier.
	// If csBin is a jar (ends in .jar or is the bootstrap), invoke as: java -jar csBin
	// If it's a native binary (cs on PATH), invoke directly.
	var csArgs []string
	if strings.HasSuffix(csBin, ".jar") || strings.Contains(csBin, "bootstrap") {
		csArgs = []string{"-jar", csBin}
	}
	csArgs = append(csArgs, "fetch",
		"--cache", cacheDir,
		"--classpath",
		v.Coordinate,
	)

	out, err := runCmdWithBin(javaBin, csArgs...)
	if err != nil {
		return fmt.Errorf("coursier fetch: %w\nOutput: %s", err, out)
	}

	// Extract the classpath line (last non-empty line, skip Coursier progress lines).
	cp := extractClasspath(out)
	if cp == "" {
		return fmt.Errorf("coursier returned empty classpath; output: %s", out)
	}

	// Write classpath.txt.
	if err := os.WriteFile(classpathFile, []byte(cp+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("  wrote %s\n", classpathFile)

	// Write launcher scripts.
	if err := writeLaunchers(javaBin, dir, cp, v.MainClass); err != nil {
		return err
	}

	// Also copy the thin CLI jar if not already present.
	thinJar := filepath.Join(dir, "cql-to-elm-cli.jar")
	if _, err := os.Stat(thinJar); os.IsNotExist(err) {
		jarURL := mavenJarURL(v.Coordinate)
		if err := downloadFile(jarURL, thinJar); err != nil {
			fmt.Printf("  warning: could not download thin jar: %v\n", err)
		}
	}

	fmt.Printf("  ✓ %s ready\n", v.Slug)
	return nil
}

// extractClasspath extracts the classpath from coursier fetch output.
// Coursier prints progress lines like "Downloaded N missing file(s) / M"
// and then emits the classpath as the last line.
func extractClasspath(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Scan from end for a line that looks like a classpath (contains path separators).
	sep := string(os.PathListSeparator) // ":" on Unix, ";" on Windows
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" && (strings.Contains(l, sep) || strings.HasSuffix(l, ".jar")) {
			return l
		}
	}
	return ""
}

func writeLaunchers(javaBin, dir, classpath, mainClass string) error {
	// Windows .bat launcher.
	bat := filepath.Join(dir, "run.bat")
	batContent := fmt.Sprintf(`@echo off
"%s" -cp "%s" %s %%*
`, javaBin, classpath, mainClass)
	if err := os.WriteFile(bat, []byte(batContent), 0o644); err != nil {
		return fmt.Errorf("write run.bat: %w", err)
	}
	fmt.Printf("  wrote %s\n", bat)

	// Unix shell launcher.
	sh := filepath.Join(dir, "run.sh")
	shContent := fmt.Sprintf(`#!/usr/bin/env bash
exec "%s" -cp "%s" %s "$@"
`, javaBin, classpath, mainClass)
	if err := os.WriteFile(sh, []byte(shContent), 0o755); err != nil {
		return fmt.Errorf("write run.sh: %w", err)
	}
	fmt.Printf("  wrote %s\n", sh)

	return nil
}

// mavenJarURL returns the Maven Central URL for a thin jar given coordinates.
func mavenJarURL(coordinate string) string {
	// coordinate format: group:artifact:version
	parts := strings.Split(coordinate, ":")
	if len(parts) != 3 {
		return ""
	}
	group := strings.ReplaceAll(parts[0], ".", "/")
	artifact := parts[1]
	version := parts[2]
	return fmt.Sprintf(
		"https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.jar",
		group, artifact, version, artifact, version,
	)
}

// runCmdWithBin runs bin with args and returns combined stdout+stderr.
func runCmdWithBin(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// downloadFile downloads url to dst atomically.
func downloadFile(url, dst string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".download-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
