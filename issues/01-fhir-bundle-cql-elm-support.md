# Issue: FHIR Measure Bundle CQL/ELM Read and Write Support

**Priority:** Enhancement
**Labels:** feature, cli, bundle, fhir
**Status:** Implemented — `pkg/bundle` and `echo-elm bundle extract|translate`.

---

## Implementation notes

The package is `pkg/bundle`; the API differs slightly from the sketch below:

- `bundle.Load(io.Reader)` rather than `LoadBundle`.
- `bundle translate` takes `--format json|xml` and the same translator flags as
  `echo-elm translate` (they share one flag set, so they cannot drift). `--verify`
  is rejected with `--format XML`: the bundled reference attachment is
  `application/elm+json`, so there would be nothing comparable to check against.
- `(*Bundle).SetELM(name, version, elmJSON)` rather than `WriteELM(name, elmJSON)` —
  a bundle can hold two versions of a library, so the version participates in the
  lookup. An empty version matches when only one library carries that name.
- `(*Bundle).LibrarySource()` returns a `*bundle.Source`, which satisfies
  `resolver.LibrarySource`. This is the part that matters: a measure's `include`
  declarations resolve against sibling libraries **in the bundle**, so nothing has
  to be written to disk first.
- `Marshal` round-trips through the original JSON, so every field the package does
  not model — `id`, `meta`, `fullUrl`, `status`, and anything else — survives a
  write-back. `TestMarshalPreservesUnmodelledFields` covers that.

Dependency order is not computed. It is not needed: `LibrarySource` resolves
includes on demand during translation, which is how the translator already works.

`echo-elm parity --bundle` is implemented too, without changing `parity.Run`'s
contract: `MaterializeBundle` writes the bundle's CQL into a temporary corpus with
a generated `corpus.yaml` and its ELM into a reference directory, then runs the
ordinary `--ref-dir` / `--lib-dir` path over it. A library carrying CQL but no ELM
is still extracted, so its dependents can resolve includes to it, but it does not
become a fixture — there is nothing to compare it against.

One caveat is worth stating: a bundle carries exactly one compiled ELM per library,
produced with whatever options its publisher used, so comparing it against several
option profiles is not meaningful. `--bundle` therefore runs a single profile,
`default` unless `--profile` names another.

---

## Summary

Add native support for reading CQL and ELM content from FHIR Measure Bundle JSON files and for writing generated ELM back into those bundles. This would allow `echo-elm` to operate directly on the artifact format used by NCQA, HL7, and other measure publishers without a pre-processing step.

---

## Motivation

HEDIS and other FHIR-based quality measure programs distribute CQL and pre-compiled ELM as base64-encoded content attachments inside FHIR `Library` resources, which are in turn collected in FHIR `Bundle` resources. Today, working with these requires extracting the content manually before `echo-elm` can operate on it, and there is no path to write generated ELM back into the bundle.

Two concrete use cases that would be enabled:

1. **Validation** — translate CQL found in a bundle, compare the generated ELM against the bundled reference ELM, and report any deviations.
2. **Re-generation** — translate CQL from a bundle with different options (e.g., annotations, updated compatibility level) and emit a new bundle or a patch containing the updated ELM content.

---

## Bundle Structure Reference

A FHIR Measure Bundle (e.g., `AAB_Reporting-2025.2.1.bundle.json`) contains one or more `Library` resources in its `entry[]`. Each `Library` has a `content[]` array with:

```json
{
  "resourceType": "Library",
  "name": "AAB_Reporting",
  "version": "2025.2.1",
  "content": [
    {
      "contentType": "text/cql",
      "data": "<base64-encoded CQL source>"
    },
    {
      "contentType": "application/elm+json",
      "data": "<base64-encoded ELM JSON>"
    }
  ]
}
```

A single reporting bundle typically contains 30–40 `Library` resources (the measure library plus all shared dependencies). The cross-library dependency graph must be resolved in declaration order (topological sort by `include` statements).

---

## Proposed Design

### New subcommand: `echo-elm bundle`

```
echo-elm bundle <subcommand> [flags]
```

#### `echo-elm bundle translate`

Translate CQL content from a bundle and compare or write ELM.

```
echo-elm bundle translate \
  --input  <bundle.json>     # FHIR Bundle JSON file
  --output <out.bundle.json> # Optional: write updated bundle with new ELM
  --format json|xml          # ELM output format (default: json)
  --verify                   # Compare generated ELM against existing bundled ELM and report differences
  --library <name>           # Translate only this Library (default: all)
  [translator flags]         # Same as echo-elm translate
```

#### `echo-elm bundle extract`

Extract CQL or ELM from a bundle to individual files.

```
echo-elm bundle extract \
  --input  <bundle.json>
  --out-dir <dir>
  --content cql|elm|both     # Which content types to extract (default: both)
```

---

## Go Package API

The bundle support should be exposed as an importable package so it can be used in tests and tooling:

```go
// pkg/bundle — proposed package

// Bundle represents a FHIR Bundle JSON document.
type Bundle struct { ... }

// LoadBundle reads and parses a FHIR Bundle from a file path or io.Reader.
func LoadBundle(r io.Reader) (*Bundle, error)

// Libraries returns all Library resources in the bundle that have CQL content.
func (b *Bundle) Libraries() []Library

// Library holds CQL source and optional reference ELM from a Bundle Library resource.
type Library struct {
    Name        string
    Version     string
    CQLSource   []byte
    ReferenceELM []byte // nil if no application/elm+json content present
}

// LibrarySource returns a resolver.LibrarySource that resolves CQL include
// statements from the bundle's Library resources.
func (b *Bundle) LibrarySource() resolver.LibrarySource

// WriteELM writes (or replaces) the application/elm+json content attachment
// for the named library with the provided ELM JSON bytes.
func (b *Bundle) WriteELM(libraryName string, elmJSON []byte) error

// Marshal serialises the (modified) bundle back to JSON.
func (b *Bundle) Marshal() ([]byte, error)
```

---

## Cross-Library Resolution

The most important detail: a measure bundle can contain 30–40 interdependent libraries. When translating any one of them, `echo-elm` must resolve `include` statements from sibling libraries in the same bundle (not from the filesystem). The proposed `Bundle.LibrarySource()` method returns a `resolver.LibrarySource` backed by the bundle's in-memory CQL content, keyed by `Name|version`.

Dependency resolution order:

1. Parse `include` statements in CQL headers.
2. Build a directed acyclic graph (DAG) across all bundle libraries.
3. Translate in reverse topological order (dependencies first).

---

## Testing Approach

- Unit tests using synthetic minimal bundles (2–3 libraries, no NCQA content).
- Integration tests against a public CQF sample bundle (if available under Apache license) or hand-authored test bundles committed to `test/`.
- Parity tests: after translating bundle CQL, compare generated ELM against the bundled reference ELM using `parity.NormalizeForGolden` to strip volatile fields before comparison.

---

## Relation to `parity` Command

Once bundle support exists, `echo-elm parity` could accept `--bundle <path>` as an alternative to `--corpus <dir>`, automatically extracting CQL, treating bundled ELM as the reference, and reporting match/differ per library. See also: [issue-02-parity-ref-elm-support.md](02-parity-ref-elm-support.md).
