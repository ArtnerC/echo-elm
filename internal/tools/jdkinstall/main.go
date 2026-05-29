// jdkinstall checks for a suitable Java runtime (≥17) on PATH and, if not
// found, downloads Temurin 17 into tools/jdk/. On success it writes
// tools/jdk/jdk.env with JAVA_BIN pointing at the java executable.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	temurinVersion = 17
	temurinAPIBase = "https://api.adoptium.net/v3"
	toolsJDKDir    = "tools/jdk"
	jdkEnvFile     = "tools/jdk/jdk.env"
)

func main() {
	fmt.Println("=== echo-elm jdkinstall ===")

	// Check system java first.
	if javaBin, ver, err := systemJava(); err == nil && ver >= temurinVersion {
		fmt.Printf("✓ System Java %d found at %s\n", ver, javaBin)
		writeEnv(javaBin)
		return
	}

	// Check project-local JDK.
	if localBin, err := localJavaBin(); err == nil {
		fmt.Printf("✓ Project-local JDK found at %s\n", localBin)
		writeEnv(localBin)
		return
	}

	// Need to download Temurin.
	fmt.Printf("Downloading Temurin %d for %s/%s…\n", temurinVersion, runtime.GOOS, runtime.GOARCH)
	if err := downloadTemurin(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	localBin, err := localJavaBin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: downloaded JDK but java binary not found: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Temurin %d installed at %s\n", temurinVersion, localBin)
	writeEnv(localBin)
}

func writeEnv(javaBin string) {
	if err := os.MkdirAll(toolsJDKDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create %s: %v\n", toolsJDKDir, err)
	}
	content := fmt.Sprintf("JAVA_BIN=%s\n", javaBin)
	if err := os.WriteFile(jdkEnvFile, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write %s: %v\n", jdkEnvFile, err)
	} else {
		fmt.Printf("  wrote %s\n", jdkEnvFile)
	}
}

// systemJava returns the path and major version of the java binary on PATH.
func systemJava() (string, int, error) {
	javaBin, err := exec.LookPath("java")
	if err != nil {
		return "", 0, err
	}
	out, err := exec.CommandContext(context.Background(), javaBin, "-version").CombinedOutput()
	if err != nil {
		return "", 0, err
	}
	ver, err := parseMajorVersion(string(out))
	if err != nil {
		return "", 0, err
	}
	return javaBin, ver, nil
}

// localJavaBin returns the path to the project-local java binary.
func localJavaBin() (string, error) {
	bin := "java"
	if runtime.GOOS == "windows" {
		bin = "java.exe"
	}
	// Temurin extracts to tools/jdk/jdk-17*/bin/java
	dirs, _ := filepath.Glob(filepath.Join(toolsJDKDir, "jdk-*", "bin", bin))
	if len(dirs) > 0 {
		return dirs[0], nil
	}
	// Flat layout: tools/jdk/bin/java
	flat := filepath.Join(toolsJDKDir, "bin", bin)
	if _, err := os.Stat(flat); err == nil {
		return flat, nil
	}
	return "", fmt.Errorf("no local JDK found under %s", toolsJDKDir)
}

var javaVersionRE = regexp.MustCompile(`(?:version\s+"?)(\d+)`)

func parseMajorVersion(output string) (int, error) {
	m := javaVersionRE.FindStringSubmatch(output)
	if m == nil {
		return 0, fmt.Errorf("cannot parse java version from: %q", output)
	}
	major, _ := strconv.Atoi(m[1])
	// Old versioning scheme: 1.8 → major=1, real version 8.
	if major == 1 {
		// e.g. "1.8.0_392" — extract second segment.
		parts := strings.SplitN(m[1], ".", 3)
		_ = parts // m[1] here is the leading "1", we need to re-match differently
		re2 := regexp.MustCompile(`1\.(\d+)`)
		m2 := re2.FindStringSubmatch(output)
		if m2 != nil {
			major, _ = strconv.Atoi(m2[1])
		}
	}
	return major, nil
}

func adoptiumURL() (string, error) {
	goOS := runtime.GOOS
	goArch := runtime.GOARCH

	osMap := map[string]string{
		"linux":   "linux",
		"darwin":  "mac",
		"windows": "windows",
	}
	archMap := map[string]string{
		"amd64": "x64",
		"arm64": "aarch64",
		"386":   "x86",
	}

	adoptOS, ok := osMap[goOS]
	if !ok {
		return "", fmt.Errorf("unsupported OS: %s", goOS)
	}
	adoptArch, ok := archMap[goArch]
	if !ok {
		return "", fmt.Errorf("unsupported arch: %s", goArch)
	}

	apiURL := fmt.Sprintf(
		"%s/binary/latest/%d/ga/%s/%s/jdk/hotspot/normal/eclipse",
		temurinAPIBase, temurinVersion, adoptOS, adoptArch,
	)

	// HEAD request to resolve the redirect URL.
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Head(apiURL) //nolint:noctx // context not required for tool download helper
	if err != nil {
		return "", fmt.Errorf("adoptium HEAD: %w", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		loc := resp.Header.Get("Location")
		if loc != "" {
			return loc, nil
		}
	}
	if resp.StatusCode == http.StatusOK {
		return apiURL, nil
	}

	// Fall back to the JSON info endpoint to get binary_link.
	infoURL := fmt.Sprintf(
		"%s/info/release_files/%d/ga/%s/%s/jdk/hotspot/normal/eclipse",
		temurinAPIBase, temurinVersion, adoptOS, adoptArch,
	)
	infoURL += "?page_size=1"
	infoResp, err := http.Get(infoURL) //nolint:noctx // context not required for tool download helper
	if err != nil {
		return "", fmt.Errorf("adoptium info GET: %w", err)
	}
	defer func() { _ = infoResp.Body.Close() }()

	var assets []struct {
		Binary struct {
			Package struct {
				Link string `json:"link"`
				Name string `json:"name"`
			} `json:"package"`
		} `json:"binary"`
	}
	if err := json.NewDecoder(infoResp.Body).Decode(&assets); err != nil {
		return "", fmt.Errorf("adoptium parse: %w", err)
	}
	if len(assets) == 0 {
		return "", fmt.Errorf("no adoptium assets found")
	}
	return assets[0].Binary.Package.Link, nil
}

func downloadTemurin() error {
	dlURL, err := adoptiumURL()
	if err != nil {
		return fmt.Errorf("resolve download URL: %w", err)
	}

	fmt.Printf("  URL: %s\n", dlURL)

	if err := os.MkdirAll(toolsJDKDir, 0o755); err != nil {
		return err
	}

	// Stream download to a temp file.
	tmpFile, err := os.CreateTemp(toolsJDKDir, "jdk-download-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	resp, err := http.Get(dlURL) //nolint:noctx // context not required for tool download helper
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("stream: %w", err)
	}
	_ = tmpFile.Close()
	fmt.Printf("  downloaded %.1f MB\n", float64(written)/1e6)

	// Extract based on file extension.
	lower := strings.ToLower(dlURL)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(tmpFile.Name(), toolsJDKDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(tmpFile.Name(), toolsJDKDir)
	default:
		return fmt.Errorf("unknown archive format: %s", dlURL)
	}
}

func extractZip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		path := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dst)+string(os.PathSeparator)) {
			continue // zip slip guard
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			_ = out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		_ = rc.Close()
		_ = out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		path := filepath.Join(dst, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dst)+string(os.PathSeparator)) {
			continue // tar slip guard
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			_ = out.Close()
			if err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		}
	}
	return nil
}
