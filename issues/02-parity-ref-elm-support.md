# Issue: `parity` — Accept Pre-Built Reference ELM Without CQF JAR

**Priority:** Enhancement  
**Labels:** parity, cli, feature  

---

## Summary

The `echo-elm parity` command currently requires the upstream CQF Java CLI (downloaded via `task install:cqframework`) to generate reference ELM. Add a `--ref-dir` flag so parity can instead read pre-built reference ELM from disk, and a `--lib-dir` flag to extend the library resolver search path. This enables parity runs against FHIR bundle ELM, CI environments where Java is unavailable, and committed golden directories.

---

## Current Behavior

`echo-elm parity` always invokes the upstream CQF JAR (`tools/cqframework/<version>/run.bat|sh`) via `runUpstream()`. If the JAR is not present, every fixture reports `upstream-error` and the run is useless. There is no way to use pre-existing ELM as the reference.

---

## Proposed Changes

### New CLI flags

```
--ref-dir <path>    Use pre-built ELM from <ref-dir>/<profile>/<fixture>.json
                    instead of running the CQF JAR.
--lib-dir <path>    Additional CQL library search directory (supplements the
                    fixture's own directory for cross-library include resolution).
--profile <name>    Run only fixtures matching this option profile name.
```

### `--ref-dir` layout

```
<ref-dir>/
  <profile-name>/        # one dir per option_profile in corpus.yaml
    <fixture>.json       # normalized ELM JSON; path mirrors corpus fixture path
    sub/dir/Lib.json
```

This mirrors the existing goldens layout at `test/goldens/cqf/`.

### Config struct additions (Go)

```go
// Config holds harness configuration.
type Config struct {
    // ... existing fields ...

    // RefDir, when non-empty, skips the upstream CQF JAR and reads
    // pre-built reference ELM from <RefDir>/<profile>/<fixture>.json.
    RefDir string

    // LibDir, when non-empty, is added as an additional DirSource for
    // cross-library CQL include resolution.
    LibDir string

    // ProfileFilter, if non-empty, runs only fixtures with this profile name.
    ProfileFilter string
}
```

### `runFixture` change

```go
func runFixtureWithRef(..., refDir, profileName string) FixtureResult {
    // When refDir is set, read reference ELM from disk instead of running JAR:
    if refDir != "" {
        refPath := filepath.Join(refDir, profileName,
            strings.TrimSuffix(fix.Path, ".cql")+".json")
        data, err := os.ReadFile(refPath)
        if err != nil {
            r.Status = StatusUpstreamError
            r.Error = fmt.Sprintf("ref ELM not found: %v", err)
            return r
        }
        r.UpstreamJSON = normalizeJSON(string(data))
    } else {
        // existing JAR path ...
    }
    // ... rest unchanged ...
}
```

---

## Implementation Notes Already Committed

This change has been prototyped locally (`C:\Dev\echo-elm`):

- `internal/parity/runner.go`: `Config.RefDir`, `Config.LibDir`, `runFixtureWithRef()`, `translateWithProfileAndLibDir()` added.
- `cmd/echo-elm/main.go`: `--ref-dir`, `--lib-dir`, `--profile` flags wired into `runParity()`.

The prototype was validated against 383 (2025.1.0) and 382 (2025.2.1) HEDIS measure libraries extracted from NCQA bundles. See [issue-03-cqf-parity-gaps.md](03-cqf-parity-gaps.md) for the results.

---

## Extraction Helper

Alongside the flag, a companion script (`extract-bundles.py` or a new `echo-elm bundle extract` subcommand — see [issue-01-fhir-bundle-cql-elm-support.md](01-fhir-bundle-cql-elm-support.md)) populates the corpus and ref-dir structure from a FHIR bundle directory:

```
parity/hedis/<version>/
  corpus/           ← .cql files + corpus.yaml
  ref/
    default/        ← reference ELM (CQF defaults, stripped of locators/metadata)
    locators/       ← same ELM used for locators-profile comparison
```

Run command:

```powershell
echo-elm parity `
  --corpus parity/hedis/2025.2.1/corpus `
  --ref-dir parity/hedis/2025.2.1/ref `
  --lib-dir parity/hedis/2025.2.1/corpus `
  --out parity/hedis/2025.2.1/runs `
  --run-id hedis-2025.2.1
```

---

## Related

- [Issue 01 — Bundle support](01-fhir-bundle-cql-elm-support.md): once bundle support is added, `parity --bundle` could replace this two-step workflow.
- [Issue 03 — CQF parity gaps](03-cqf-parity-gaps.md): analysis produced using this flag.
