# echo-elm — Implementation Plan

> Live document. Update at every phase boundary or material change.
> See `specs/echo-elm/spec.md` for the spec, `tasks.md` for granular work,
> and `contracts/*.md` for the frozen interface surfaces each phase delivers.

## Operating principles

- **Test-first for public surfaces.** Every `pkg/echoelm` export, local UI API endpoint, and MCP tool ships with tests in the same commit.
- **Dependency discipline.** Use libraries where they add reliability or
  compatibility, but keep transitive dependency graphs intentional. MCP uses the
  official Model Context Protocol Go SDK, not community protocol packages.
- **Parity is a budget, not an aspiration.** Minimal CQFramework parity tooling
  starts in Phase 0/1 so every compiler phase can report parity-vs-CQFramework
  delta. A delta regression blocks the phase.
- **Commit cadence.** Conventional Commits, one commit per coherent slice, always with `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`. Each phase ends with a tagged checkpoint (`phaseN-done`).
- **Planning docs stay live.** Tick tasks, update phase status, append decisions to a `Decision Log` section at the bottom of this file.

## Phase 0 — Bootstrap

- `git init`, `.gitignore`, `LICENSE` (Apache-2.0), `NOTICE`.
- `go mod init github.com/<org>/echo-elm` (org TBD; placeholder `echo-health` until set).
- `Taskfile.yml` with targets: `build test lint fmt run ui mcp ui:dev ui:build ui:bundle-report install:deps install:jdk parity`.
- `.editorconfig`, `.golangci.yml`, `pre-commit` (gofmt + golangci-lint + prettier for `web/workbench`).
- Wire planning artifacts into `AGENTS.md` (already drafted).
- CI scaffold (`.github/workflows/ci.yml`): build, vet, test, lint on Win/macOS/Linux.
- Minimal CQFramework parity smoke harness: project-local JDK/JAR install
  targets plus a tiny fixture set, expanded later in Phase 10.

## Phase 1 — Lexer & Parser (CQL 1.5.3)

- ANTLR4 Go target for the official `cqlLexer.g4` + `cqlParser.g4` (pinned to spec ZIP).
- Parse-tree → AST with full locator retention.
- Diagnostic collector with severity ladder; error recovery.
- Tests: round-trip every file in `test/corpus/cqframework/cql-to-elm` parses without error or matches upstream's error list.

## Phase 2 — Symbol resolution & type system

- Scope chain, qualified-name resolution, `include`/`using` semantics.
- Type lattice (System types + ModelInfo types), conversions, implicit/explicit cast.
- List promotion/demotion, interval promotion/demotion (option-gated, defaults match spec).
- Overload resolution honoring `SignatureLevel` (None/Differing/Overloads/All).
- `cqlCompilerOptions.Options` enum: all 13 values honored.

## Phase 3 — ELM builder + serializers

- AST → complete ELM R1 model representation (`urn:hl7-org:elm:r1`), with
  builder coverage growing incrementally but model/schema coverage complete.
- Serializers: XML (with xsi:type quirks matching upstream) and JSON.
- Schema validation against bundled XSDs.
- `Annotation` emission when `EnableAnnotations` (snippet shape matches upstream).
- Goldens: `test/golden/<corpus-id>/{Library.xml,Library.json}`.

## Phase 4 — Resolvers & options

- `LibrarySource` interface + filesystem and `fs.FS` impls (matches CQFramework `DefaultLibrarySourceProvider` semantics).
- `ModelInfoProvider` + bundled System and FHIR R4/FHIRHelpers ModelInfo. QI-Core/US Core/QDM are supported through explicit versioned providers/packages and test fixtures, not hard-coded language behavior.
- `TerminologyProvider` interface (no implementation in v0; tests use a fake).
- UCUM validator (port or wrap; decision: prefer port for single-binary).
- Options loader: `cql-options.json` (CQFramework convention) + `Measure.extension.cqf-cqlOptions` (per CQF spec).

## Phase 5 — CLI surfaces

- Subcommands: `translate` (default), `cqf translate`, `ui`, `mcp`, `parity`, `version`.
- `translate` is modern echo-elm XML/JSON output; `cqf translate` is the upstream-compatible surface (see `contracts/cli.md`).
- `cqf translate` stderr banner byte-exact (LF only). CQF exit codes match upstream (`0` success, `1` translation errors, `2` usage).
- Parity smoke tests in CI: run echo-elm + upstream JAR on a small fixture set; diff stderr.

## Phase 6 — Public Go API freeze

- Finalize `pkg/echoelm`: `Translator`, `Options`, `Result`, `Diagnostic`, `LibrarySource`, `ModelInfoProvider`, `TerminologyProvider`.
- godoc complete; examples for each exported type.
- API audit task records the v0.1.0 surface in `contracts/go-api.md`.

## Phase 7 — Local UI HTTP server

- `internal/ui` with `chi` or `net/http` + minimal middlewares (logging, recover, request-id).
- Loopback-only by default; refuses non-loopback bind without `--allow-remote`.
- Endpoints per `contracts/local-api.md`: workspace, libraries, translate, parity reports, meta.
- All endpoints stateless — backed by `pkg/echoelm` + filesystem under `--workspace`.
- `httptest` integration coverage for every endpoint.

## Phase 8 — MCP server

- `echo-elm mcp` over stdio using the official Model Context Protocol Go SDK. Tool inventory in `contracts/mcp-tools.md`.
- All tools return structured results matching documented JSON schemas.
- Tests: stdio harness exercising each tool end-to-end.

## Phase 9 — Svelte workbench

- SvelteKit with `adapter-static`, Tailwind, CodeMirror 6.
- Three routes: `/` (library browser), `/translate`, `/parity/:id`.
- Build emits to `web/workbench/build/`; embedded via `//go:embed`.
- Vitest + one Playwright smoke against `echo-elm ui --workspace test/fixtures/workspace`.
- Bundle budget: ≤ 150 KB gzipped initial JS.

## Phase 10 — Parity harness

- `task install:jdk` installs Temurin 17 into `tools\jdk\` (Adoptium API).
- Fetch upstream CLI JARs: 3.29.0 (`info.cqframework`) and 4.8.0 (`org.cqframework`).
- `cmd/echo-elm/parity` driver: for each corpus fixture, run echo-elm and both upstream JARs, normalize stderr (timestamps stripped), diff ELM (XML + JSON), produce a `report.json` and `report.md` per run.
- Import `clinical_quality_language` cql-to-elm tests verbatim under `test/corpus/cqframework/` with NOTICE + provenance.
- Import `cqframework/cql-tests` if license permits; otherwise reference at runtime.
- CI: parity job runs against a tagged subset; full sweep runs nightly.

## Phase 11 — Hardening & release

- Performance pass: parser/AST allocations, ELM serialization, parallel translation.
- Release artifacts: `echo-elm` binaries (Win/macOS/Linux × amd64/arm64) attached to GitHub Release; container image `ghcr.io/<org>/echo-elm`.
- Docs site (mkdocs or static) from `docs/` + `contracts/`.

---

## Decision Log

| Date | Decision | Rationale |
|---|---|---|
| pending | DuckLake/Iceberg excluded | user clarified copy-paste error |
| pending | Postgres excluded entirely | translator is stateless; UI is filesystem-backed; no multi-user use case in v0 |
| pending | Local UI server is loopback-only, no auth | single-user developer workbench, not an ops console |
| pending | Svelte workbench scope: library browser + translator + parity viewer only | "lightweight and self-contained" per user |
| pending | SvelteKit `adapter-static` + `//go:embed` | single binary distribution |
| pending | Taskfile.yml as task runner | cross-platform, Windows-friendly |
| pending | Temurin 17 project-local | reproducibility for parity tests |
| pending | Main CLI is modern; CQF parity lives under `echo-elm cqf translate` | keep current SDK/downstream-consumer workflow clean while retaining exact legacy compatibility |
| pending | `COFFEE` output only in CQF compatibility mode | legacy CommonJS wrapper not used by modern ELM consumers |
| pending | CQL 1.5 only in main interface; CQL 2 deferred to future experimental mode | avoid ballot behavior leaking into stable output |
| pending | Minimal parity smoke starts in Phase 0/1 | avoid late compiler rewrites |
