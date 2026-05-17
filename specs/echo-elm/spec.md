# echo-elm — Product & System Specification

> **Status:** Draft v0.1 (planning only — pending sign-off)
> **Owner:** repo maintainers + AI agents
> **Companion:** `plan.md` (phases), `tasks.md` (granular work), `contracts/*.md` (surfaces), `docs/*.md` (architecture & ops)

---

## 1. Vision

**echo-elm** is a fully spec-compliant CQL → ELM translator implemented in Go.
It ships as:

1. A **modern CLI** for producing ELM XML/JSON for current SDKs and downstream
   CQL/ELM consumers,
   plus a CQFramework compatibility namespace (`echo-elm cqf ...`) with
   byte-for-stderr-banner parity with the upstream CQFramework `cql-to-elm`
   CLI (both `info.cqframework:cql-to-elm:3.29.0` and
   `org.cqframework:cql-to-elm-jvm:4.8.0+`).
2. An **importable Go package** (`pkg/echoelm`) that exposes the translator
   pipeline as a stable, tested public API.
3. A **local workbench** (`echo-elm ui`) — loopback-only HTTP server with an
   embedded SvelteKit SPA for browsing CQL libraries in a workspace, running
   translations interactively with a diagnostic-aware editor, and viewing
   parity reports. Stateless; no database; no auth.
4. An **MCP server** (`echo-elm mcp`) so AI agents can drive translation,
   validation, parity tests, and local build/test loops as first-class tools.

echo-elm is the companion translator for **echo-qm** (separate project), but
this repository only builds the translator and translator-adjacent tooling.
"dqm-engine" is not a name used in this repo.

## 2. Non-goals

- **Measure evaluation.** That belongs to `echo-qm`.
- **Terminology service.** echo-elm consumes terminology metadata; expansion is
  external (FHIR `$expand` / VSAC).
- **CQL execution / runtime.** echo-elm produces ELM; execution is downstream.
- **CQL ⇄ FHIR mapping authoring.** Out of scope.

## 3. Conformance targets

| Surface | Target | Source of truth |
|---|---|---|
| CQL grammar | **CQL 1.5.3** (ANSI/HL7 CQLANG R1-2020 R2025) | `references/specs/cql/1.5.3/` |
| CQL 2.0 ballot | Future-facing, explicitly experimental; no effect on 1.5 output | `references/specs/cql/2.0.0-ballot/` |
| ELM schemas | `urn:hl7-org:elm:r1` (`expression.xsd`, `clinicalexpression.xsd`, `library.xsd`, `modelinfo.xsd`, `types.xsd`) | spec ZIP |
| Main CLI flags | Modern echo-elm interface for XML/JSON ELM generation | `contracts/cli.md` |
| CQFramework CLI flags | Identical set to upstream `cql-to-elm` 3.29.0 **and** 4.8.0, isolated under `echo-elm cqf translate` | `agents/cqframework-compatibility.md` |
| Compiler options enum | `CqlCompilerOptions.Options` (13 values) | `references/implementation/cqframework-compatibility.md` |
| Defaults (modern/FHIR artifact translation mode) | `EnableAnnotations`, `EnableLocators`, `DisableListDemotion`, `DisableListPromotion`, method invocation enabled, list traversal enabled, `validateUnits=true`, `compatibilityLevel=1.5`, `errorLevel=Info`, signature level=`Overloads` | spec |
| Media types | `text/cql`, `text/cql-identifier`, `text/cql-expression`, `application/elm+xml`, `application/elm+json` | spec |
| Stderr banner | LF-only, exact format below | empirical from upstream JARs |

```
================================================================================
TRANSLATE <inputPath>
Translation completed successfully.
ELM output written to: <outputPath>
```

Error banner:
```
================================================================================
TRANSLATE <inputPath>
Translation failed due to errors:
<Severity>:[<sl>:<sc>, <el>:<ec>] <message>
```
`[n/a]` is emitted when locator is missing.

`echo-elm translate` defaults to modern ELM output: XML/JSON only, CQL 1.5,
and FHIR/ELM-suitability defaults. CQFramework legacy shorthands (`--strict`,
`--debug`), legacy compatibility levels (`1.3`, `1.4`), and `COFFEE` output
belong to `echo-elm cqf translate`.

## 4. Users & use cases

| User | Use case | Surface |
|---|---|---|
| Measure author | Translate a `.cql` library and inspect ELM | CLI / Svelte workbench |
| Toolchain integrator | Embed translation in a Go pipeline | `pkg/echoelm` |
| CI / regression author | Compare echo-elm ELM with CQFramework ELM | `echo-elm parity` + harness |
| Measure author / dev | Browse libraries, run translations, inspect diagnostics & parity | Svelte workbench (`echo-elm ui`) |
| AI dev agent | Drive build/test/translate/diff from chat | MCP server |

## 5. High-level architecture

```
┌────────────────────────────────────────────────────────────────────┐
│ Frontends                                                          │
│   CLI (echo-elm)        Svelte workbench (//go:embed)              │
│   MCP server (stdio)    Local UI API clients (loopback)            │
└──────┬──────────────────────┬─────────────────────┬────────────────┘
       │                      │                     │
┌──────▼──────────────────────▼─────────────────────▼────────────────┐
│ Application layer                                                  │
│   cmd/echo-elm/{translate,cqf,ui,mcp,parity,version}               │
│   internal/ui (HTTP+SPA) │ internal/mcp │ internal/parity          │
└──────┬─────────────────────────────────────────────────────────────┘
       │
┌──────▼─────────────────────────────────────────────────────────────┐
│ Translator core (pkg/echoelm)                                      │
│   lexer → parser (ANTLR4) → AST → symbol/type → ELM builder        │
│   → serializers (XML, JSON) → schema validation                    │
│   options · ucum · model-info · library-source · terminology iface │
└────────────────────────────────────────────────────────────────────┘

State: filesystem only. CQL libraries are folders on disk (CQFramework
convention) or supplied through Go `fs.FS` / in-memory sources. CLI translation
writes ELM next to inputs (or to --output). Parity runs write reports to
<workspace>/parity/runs/<id>/. No database.

Parity harness (internal/parity): drives upstream CQFramework JARs (3.29.0,
4.8.0) via Temurin 17 (project-local under tools/jdk/) and diffs stderr + ELM.
```

## 6. Repository layout (target)

```
echo-elm/
├── AGENTS.md
├── README.md
├── LICENSE  NOTICE
├── Taskfile.yml
├── go.mod  go.sum
├── plan.md  tasks.md
├── specs/echo-elm/spec.md          ← this file
├── contracts/
│   ├── go-api.md  cli.md  mcp-tools.md
│   ├── local-api.md  frontend.md  parity-harness.md
├── docs/architecture.md  dev-environment.md  testing-strategy.md
├── cmd/echo-elm/                   ← single binary, subcommands
├── pkg/echoelm/                    ← public Go API
├── internal/
│   ├── lexer parser ast symtab types elm/{xml,json}
│   ├── options modelinfo libsrc terminology ucum
│   ├── ui  mcp  parity
├── web/workbench/                  ← SvelteKit, adapter-static, //go:embed
├── tools/jdk/                      ← project-local Temurin 17 (gitignored)
├── test/
│   ├── corpus/cqframework/         ← copied from upstream (Apache-2.0)
│   ├── corpus/cql-tests/           ← if license permits
│   ├── golden/                     ← XML+JSON ELM goldens
│   ├── integration/                ← ui httptest, mcp stdio harness
│   └── parity/                     ← runs upstream JARs + diffs
├── agents/                         ← per-domain agent playbooks
└── references/                     ← versioned spec/tool references (existing)
```

## 7. Cross-cutting requirements

- **Determinism.** Translator output is byte-deterministic given (input, options, model-info bundle). XML+JSON ELM goldens are diffed in CI.
- **Locator fidelity.** All AST nodes carry `[sl:sc, el:ec]`; serialized when `EnableLocators`.
- **Annotations.** When `EnableAnnotations`, ELM `annotation` elements include source snippet, matching upstream's `TraceabilityAnnotation` shape.
- **Diagnostics.** Severity ladder `Trace < Info < Warning < Error`. `errorLevel` filters what propagates; reaching `Error` aborts the pipeline.
- **Cross-platform.** Builds on Windows, macOS, Linux (amd64 + arm64). CLI line endings: LF on stderr banner (matches upstream).
- **Public API stability.** `pkg/echoelm` follows semver; breaking changes only on major. All exported symbols have godoc + tests.
- **Single binary.** The `ui` subcommand bundles the Svelte workbench (`//go:embed`). CLI, UI, MCP, parity, and version are all subcommands.
- **All public APIs tested.** Go `testing` + `httptest`; Vitest + Playwright for the workbench. No database dependency.
- **ModelInfo boundary.** CQL/ELM compliance is language-level, but translating
  retrieves against FHIR, QI-Core, QDM, or other models requires ModelInfo.
  echo-elm core is model-agnostic. It bundles System + FHIR R4/FHIRHelpers for
  common use, exposes external ModelInfo providers, and keeps profile-specific
  QI-Core/US Core/QDM modelinfo as versioned packages/fixtures rather than
  hard-coded language behavior.
- **Reproducible toolchain.** `task install:deps` brings Go modules, JDK 17 (for parity), Node + pnpm (for UI), and Python via `uv` (for any helper scripts; `uvx` for one-shots).
- **License hygiene.** echo-elm: Apache-2.0. Copied test corpus retains upstream NOTICE + provenance file.
- **Dependency hygiene.** Use reliable third-party libraries when they improve
  compatibility or reduce implementation risk, but keep transitive dependency
  graphs intentional and small. Use the official Model Context Protocol Go SDK
  for MCP, not community protocol packages.

## 8. Open questions (post sign-off)

1. Distribute via Homebrew / Scoop / apt? Deferred.

## 9. Definition of done (per phase)

A phase is "done" when: code merged, `task test` green, `task lint` clean,
contracts updated, golden files refreshed (if applicable), parity-vs-upstream
delta unchanged or improved, and a Conventional-Commits commit is recorded
with the Copilot Co-authored-by trailer.
