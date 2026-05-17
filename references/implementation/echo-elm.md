# echo-elm implementation playbook

`echo-elm` is a Go implementation of a CQL→ELM translator. It ships as a CLI,
an importable Go package, a loopback-only workbench (`echo-elm ui`), and an MCP
server. It targets full conformance to the CQL 1.5.3 specification
(ANSI/HL7 CQLANG R1-2020 (R2025)) and provides a CQFramework
**compatibility mode** that matches the JVM `cql-to-elm` CLI byte-for-byte
where reasonable. The companion runtime/engine project is `echo-qm` and lives
in a separate repository.

## Goals

1. **Spec compliance** — implement CQL 1.5.3 and emit ELM R1 that validates
   against the published XSDs (`expression.xsd`, `clinicalexpression.xsd`,
   `library.xsd`) and the JSON form described in *Using CQL with FHIR 2.0.0*.
2. **Modern CLI + CQF compatibility** — provide a modern `echo-elm translate`
   interface for XML/JSON ELM generation, plus `echo-elm cqf translate` with
   the same flag names, defaults, and exit/IO semantics as the CQFramework CLI
   for both `info.cqframework:cql-to-elm` 3.29.0 and
   `org.cqframework:cql-to-elm-jvm` 4.8.0 (and current `main`). See
   `references\implementation\cqframework-compatibility.md`.
3. **Library API** — expose a stable, idiomatic Go API
   (`pkg/echoelm`) for embedding in other Go programs (notably `echo-qm`)
   without going through the CLI.
4. **Local workbench** — expose a stateless, loopback-only `echo-elm ui`
   workbench for browsing CQL libraries, running interactive translations, and
   viewing parity reports.
5. **MCP server** — expose safe filesystem-scoped tools so AI agents can drive
   translation, validation, parity runs, and local build/test loops.
6. **Determinism** — given the same inputs, options and resolved
   dependencies, produce identical ELM XML/JSON.

## Repository shape (target)

```
echo-elm/
  cmd/echo-elm/             # CLI entry point (cobra or stdlib flag)
  pkg/echoelm/              # public Go API
  internal/
    lex/                    # ANTLR4-derived lexer (Go target)
    parse/                  # parser + parse tree
    sem/                    # symbol tables, scopes, name resolution
    types/                  # CQL type system, conversions, promotion/demotion
    elm/                    # ELM AST + builder
    serialize/xml,json      # ELM physical-representation serializers
    modelinfo/              # ModelInfo loader, FHIR/QDM/QICore/USCore providers
    library/                # LibrarySourceProvider, NPM package loader
    terminology/            # ValueSet/CodeSystem/membership resolver
    ucum/                   # units validation
    diagnostics/            # error/warning/info with locators + severity
    ui/                     # loopback HTTP API + embedded workbench
    mcp/                    # stdio MCP server
    parity/                 # CQFramework parity harness
  web/workbench/            # SvelteKit workbench, adapter-static
  contracts/                # public surface contracts
  docs/                     # architecture, dev environment, testing strategy
  references/               # spec + tool references (versioned)
  agents/                   # agent markdown files for AI work
  AGENTS.md
```

## Architecture

1. **Source intake** — accept `text/cql` from files, directories (recursive,
   `*.cql` and `*.CQL`), stdin, `[]byte`, FHIR `Library.content` of media
   type `text/cql`, or canonical URL through a `LibrarySourceProvider`.
2. **Lex/parse** — ANTLR4 CQL 1.5.3 grammar compiled to the Go target.
   Retain token start/end line+char for every node so locators and
   annotations are emitable.
3. **Symbols & scopes** — build symbol tables for libraries, includes,
   parameters, valuesets, codesystems, codes, concepts, contexts, functions
   (with overloads), and expressions.
4. **Resolver layer** — pluggable interfaces:
   - `LibrarySourceProvider` (canonical/name/version → CQL or ELM bytes)
   - `ModelInfoProvider` (model name+version → `ModelInfo`)
   - `TerminologyProvider` (membership, expansion, lookup)
   - `UcumService` (unit validation when `validate-units` is enabled)
   - `OptionsProvider` (`cql-options.json` and FHIR `cqf-cqlOptions`)
5. **Type system** — full CQL type lattice (System types, intervals, lists,
   tuples, choice types, class types, generics, implicit conversions,
   list/interval promotion & demotion gated by options, overload
   resolution honoring `SignatureLevel`).
6. **ELM builder** — produce semantic ELM nodes including `resultTypeName`
   / `resultTypeSpecifier`, signatures (per `SignatureLevel`), annotations
   (per `EnableAnnotations`), locators (per `EnableLocators`).
7. **Serializer** — XML (default) and JSON, both bound to the canonical
   namespaces `urn:hl7-org:cql:r1` and `urn:hl7-org:elm:r1`. JSON encoding
   mirrors XML but uses the `type` attribute for polymorphic discriminators
   and `{namespace}Name` for qualified type names; no mixed content.
8. **Diagnostics** — stable severity ladder
   (`Trace`/`Info`/`Warning`/`Error`) with `[startLine:startChar,
   endLine:endChar]` locators, matching the CQFramework CLI message format.

## Default options for FHIR artifacts

Per *Using CQL with FHIR 2.0.0* and matching `CqlCompilerOptions.defaultOptions()`:

- `EnableAnnotations`
- `EnableLocators`
- `DisableListDemotion`
- `DisableListPromotion`
- `MethodInvocation` enabled (no `DisableMethodInvocation`)
- `ListTraversal` enabled (no `DisableListTraversal`)
- `SignatureLevel` — **library API and modern CLI** default `Overloads`
  (matches engine/FHIR suitability guidance); **CQF compatibility CLI** default
  `None` (matches the CQFramework `--signatures` default for both 3.29.0 and
  4.8.0). Allow callers to override.
- `validateUnits = true`
- `compatibilityLevel = "1.5"`
- `errorLevel = Info`

An artifact's `cqf-cqlOptions` extension on a `Library`, when present, takes
precedence over these defaults.

## Required resolvers

- `LibrarySourceProvider` — at minimum: `DefaultLibrarySourceProvider`
  (filesystem, same directory as the source), `FhirLibrarySourceProvider`
  (bundled FHIRHelpers, FHIRCommon), `NpmLibrarySourceProvider` (FHIR IG
  packages).
- `ModelInfoProvider` — `DefaultModelInfoProvider` (filesystem),
  `NpmModelInfoProvider`, and a registry seeded with bundled
  `System`, `FHIR-ModelInfo`, `QICore-ModelInfo`, `USCore-ModelInfo`,
  `QDM-ModelInfo`.
- `TerminologyProvider` — interface only in echo-elm proper; the engine
  (`echo-qm`) plugs in real VSAC/TX server clients.
- `UcumService` — interface + a bundled `ucum-essence.xml` table.

## Acceptance tests

- Parser golden tests for representative CQL snippets.
- Semantic diagnostic tests for every error/warning code we emit.
- ELM golden snapshots for the CQL examples shipped with `clinical_quality_language/Examples/`.
- Schema validation of XML output against `library.xsd` and JSON output
  against the equivalent JSON shape.
- **Compatibility parity** — for the same CQL + options + resolved
  dependencies, diff our ELM JSON against the CQFramework CLI output and
  fail on semantic differences (ignoring whitespace, annotation text
  formatting, and the `translatorVersion` attribute).
- FHIR Library packaging tests for `text/cql`, `application/elm+xml`,
  `application/elm+json` round-trips.
- CQL spec operator coverage: null behavior, intervals, lists, datetime
  arithmetic, quantities, terminology operators, queries, aggregates.

## Non-negotiable compatibility details

- Preserve library identifier, version, and `using`/`include` aliases
  exactly as they appear in source.
- Never silently pick an overload when ambiguity exists — emit a
  diagnostic and refuse to produce ELM (matching CQFramework behavior).
- Record translation options on the ELM header so downstream runtimes can
  run ELM suitability checks.
- Keep CQL 2 ballot features out of the main interface until an explicit
  experimental mode is designed; never leak them at CQL 1.5.
- Treat the chosen ModelInfo strategy (profile-informed vs. derived) as
  part of artifact identity — record it, do not mix.

## See also

- `references\implementation\cqframework-compatibility.md`
- `references\implementation\spec-compatibility.md`
- `references\implementation\terminology.md`
- `references\specs\cql\1.5.3\README.md`
- `references\specs\using-cql-with-fhir\2.0.0\README.md`
- `references\tools\cqframework-cql\4.8.0\README.md`
- `references\tools\cqframework-cql\3.29.0\README.md`
