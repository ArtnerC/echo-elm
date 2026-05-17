# echo-elm — Tasks

> Live project task list. Implementation progress may also be mirrored in the
> AI session's private todo tracker, but echo-elm itself has no SQL/database
> component.
> Status legend: `[ ]` pending · `[~]` in progress · `[x]` done · `[!]` blocked.

## Phase 0 — Bootstrap

- [ ] **bootstrap-git-init** — `git init`, `.gitignore`, branch `main`.
- [ ] **bootstrap-license** — Apache-2.0 `LICENSE`, `NOTICE`.
- [ ] **bootstrap-go-mod** — `go mod init`, pin Go 1.26.3 in `go.mod` `toolchain`.
- [ ] **bootstrap-taskfile** — `Taskfile.yml` with `build test lint fmt run ui mcp ui:dev ui:build ui:bundle-report install:deps install:jdk parity` targets.
- [ ] **bootstrap-lint** — `.golangci.yml`, `.editorconfig`, prettier config for `web/workbench/`.
- [ ] **bootstrap-ci** — `.github/workflows/ci.yml`: matrix Win/macOS/Linux, Go build+vet+test+lint.
- [ ] **bootstrap-parity-smoke** — Minimal CQFramework JDK/JAR install targets + tiny parity fixture set available before parser work begins.
- [ ] **bootstrap-readme** — Top-level `README.md` referencing AGENTS, spec, plan, tasks, contracts.
- [ ] **bootstrap-commit** — Conventional Commit `chore(repo): bootstrap workspace`.

## Phase 1 — Lexer & Parser

- [ ] **antlr-grammar-import** — Copy `cqlLexer.g4`, `cqlParser.g4` from CQL 1.5.3 spec ZIP into `internal/parser/grammar/`.
- [ ] **antlr-codegen** — Taskfile target generating Go visitor/listener via `antlr-4.13.x`.
- [ ] **lexer-ast** — AST nodes with `Locator` (sl,sc,el,ec) on every node.
- [ ] **parser-diagnostics** — Diagnostic collector; severity ladder; recovery.
- [ ] **parser-tests** — Corpus parse tests; round-trip parse expectations match upstream.

## Phase 2 — Symbols & types

- [ ] **scope-chain** — Lexical scoping, `using`/`include`/`define` semantics.
- [ ] **type-lattice** — System types + ModelInfo types; subtype relations.
- [ ] **conversions** — Implicit/explicit cast, list/interval promotion/demotion, all flag-gated.
- [ ] **overload-resolution** — Honors SignatureLevel; modern default `Overloads`, CQF CLI default `None`.
- [ ] **options-enum** — All 13 `CqlCompilerOptions.Options` values honored end-to-end.

## Phase 3 — ELM builder & serializers

- [ ] **elm-model** — Complete Go representation of ELM R1 schema; builder coverage may grow incrementally, but model/schema coverage is not partial.
- [ ] **elm-xml** — Serializer with xsi:type emission matching upstream.
- [ ] **elm-json** — JSON serializer matching upstream key ordering + types.
- [ ] **elm-validate** — XSD validation against bundled schemas.
- [ ] **annotations** — `EnableAnnotations` emits snippet annotations.
- [ ] **goldens-init** — Seed `test/golden/` with a curated subset.

## Phase 4 — Resolvers & options

- [ ] **library-source-fs** — Filesystem and `fs.FS` `LibrarySource` with CQFramework path semantics.
- [ ] **model-info-bundle** — Embed System + FHIR R4/FHIRHelpers ModelInfo; support QI-Core/US Core/QDM through explicit versioned providers/packages and fixtures.
- [ ] **terminology-iface** — `TerminologyProvider` interface + fake for tests.
- [ ] **ucum** — Port/wrap UCUM validator; `validateUnits` flag wired.
- [ ] **options-loader** — `cql-options.json` + `cqf-cqlOptions` extension loader.

## Phase 5 — CLI surfaces

- [ ] **cli-flags** — Modern `translate` flags + CQFramework-compatible `cqf translate` flags in `contracts/cli.md`.
- [ ] **stderr-banner** — Byte-exact banner (LF). Error format matches.
- [ ] **exit-codes** — `0` success / `1` errors / `2` usage.
- [ ] **cli-tests** — Modern CLI behavior tests + CQF golden-stderr tests + parity smoke vs upstream JAR.

## Phase 6 — Public Go API freeze

- [ ] **goapi-surface** — Final exports under `pkg/echoelm`; godoc complete.
- [ ] **goapi-examples** — `ExampleTranslator_*` tests.
- [ ] **goapi-audit** — Record v0.1.0 surface in `contracts/go-api.md`.

## Phase 7 — Local UI HTTP server

- [ ] **ui-skeleton** — `internal/ui` with chi/net-http, loopback-bind guard, request-id + slog middleware.
- [ ] **ui-workspace** — `/api/workspace` + `/api/libraries` + `/api/libraries/{path}` (filesystem walk).
- [ ] **ui-translate** — `POST /api/translate` (sync, calls `pkg/echoelm`).
- [ ] **ui-parity-viewer** — `/api/parity/runs[/{id}[/report.md|/diffs/...]]` (reads `parity/runs/`).
- [ ] **ui-meta** — `/api/healthz`, `/api/version`.
- [ ] **ui-tests** — `httptest` per endpoint + filesystem-fixture workspaces.

## Phase 8 — MCP server

- [ ] **mcp-skeleton** — `echo-elm mcp` stdio loop using the official Model Context Protocol Go SDK.
- [ ] **mcp-tools** — Implement tool inventory in `contracts/mcp-tools.md`.
- [ ] **mcp-tests** — Stdio harness round-trips for every tool.

## Phase 9 — Svelte workbench

- [ ] **ui-scaffold** — SvelteKit + Tailwind + CodeMirror 6 under `web/workbench/`; adapter-static.
- [ ] **ui-route-libraries** — `/` library browser.
- [ ] **ui-route-translate** — `/translate` editor + diagnostics + ELM tabs; locator click-to-jump.
- [ ] **ui-route-parity** — `/parity/:id` markdown viewer + per-fixture diff drawer.
- [ ] **ui-embed** — `//go:embed web/workbench/build` served by `ui`.
- [ ] **ui-bundle-budget** — CI check ≤ 150 KB gzipped initial JS.
- [ ] **ui-tests** — Vitest for `lib/`; one Playwright smoke exercising all three routes.

## Phase 10 — Parity harness

- [ ] **jdk-install** — `task install:jdk` fetches Temurin 17 into `tools\jdk\`.
- [ ] **jar-fetch** — Download upstream CLI JARs (3.29.0 + 4.8.0) into `tools\cqframework\`.
- [ ] **corpus-import** — Copy `clinical_quality_language` cql-to-elm tests + NOTICE/provenance.
- [ ] **cql-tests-import** — Conditional on license; otherwise wire as git submodule.
- [ ] **parity-driver** — `cmd/echo-elm/parity`: orchestrates runs, diffs, report writing.
- [ ] **parity-ci** — Tagged subset on PR; full nightly.

## Phase 11 — Hardening & release

- [ ] **perf-pass** — Profile parser/builder/serializer; reduce allocs.
- [ ] **release-binaries** — GoReleaser config for Win/macOS/Linux × amd64/arm64.
- [ ] **container-image** — Distroless image to GHCR.
- [ ] **docs-site** — Static docs from `docs/` + `contracts/`.
