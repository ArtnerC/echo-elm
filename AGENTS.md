# AGENTS.md — echo-elm

This file is the entry point for AI coding agents working in the
`echo-elm` repository. It captures the project's identity, the
non-negotiable spec rules, the compatibility surface we maintain, and
pointers into the versioned reference library under `references\`.

## Project identity

- **Name:** `echo-elm`.
- **What it is:** a Go implementation of a Clinical Quality Language
  (CQL) → Expression Logical Model (ELM) translator.
- **Form factors:** (1) a CLI (`cmd/echo-elm`), (2) an importable Go
  package (`pkg/echoelm`), (3) a loopback-only workbench (`echo-elm ui`),
  and (4) an MCP server (`echo-elm mcp`). All must be first-class.
- **Goal:** full conformance to **CQL 1.5.3** (ANSI/HL7 CQLANG R1-2020
  (R2025), published 2025-03-07) and the ELM R1 schema. The main
  `echo-elm translate` interface is modern XML/JSON output for current SDKs
  and echo-qm; CQFramework `cql-to-elm` compatibility for
  `org.cqframework:cql-to-elm-cli:5.0.0` lives under
  `echo-elm cqf translate`.
- **Companion project (separate repo):** `echo-qm` — the dQM / measure
  evaluation engine. Anything related to ELM execution, terminology
  expansion, data retrieval, MeasureReport generation, etc. lives there,
  not here.
- **Scope boundary:** this repo builds only the CQL→ELM translator and
  translator-adjacent tooling (CLI, Go API, local workbench, MCP, parity
  harness). It does **not** execute CQL/ELM, calculate measures, retrieve
  patient data, or implement echo-qm.

### Naming rules

- Use **echo-elm** as the project name everywhere.
- Do **not** use "cql-to-elm-translator" or "dqm-engine" as project
  names. They're fine as generic descriptions of the *kind* of software
  but not as identifiers for any project in this org.
- Use **echo-qm** when referring to the downstream engine. Treat
  `references\implementation\dqm-engine.md` and the
  `echo-qm-reference` skill as reference material only.

## Spec baseline (always default to these)

| Area | Version | Local reference |
| --- | --- | --- |
| CQL language + ELM schema | 1.5.3 | `references\specs\cql\1.5.3\README.md` |
| CQL 2 future work | 2.0.0-ballot (future/experimental, not baseline) | `references\specs\cql\2.0.0-ballot\README.md` |
| FHIR base | R4 4.0.1 | `references\specs\fhir-r4\4.0.1\README.md` |
| Using CQL with FHIR | 2.0.0 STU2 | `references\specs\using-cql-with-fhir\2.0.0\README.md` |
| CRMI | 1.0.0 STU1 | `references\specs\crmi\1.0.0\README.md` |
| QI-Core | 7.0.2 STU7 | `references\specs\qicore\7.0.2\README.md` |
| US Core | 8.0.1 STU8 | `references\specs\us-core\8.0.1\README.md` |
| QDM | 5.6 | `references\specs\qdm\5.6\README.md` |
| CQFramework `cql-to-elm` (current) | `org.cqframework:cql-to-elm-jvm:4.8.0` | `references\tools\cqframework-cql\4.8.0\README.md` |
| CQFramework `cql-to-elm` (legacy) | `info.cqframework:cql-to-elm:3.29.0` | `references\tools\cqframework-cql\3.29.0\README.md` |

Versioning convention: new versions go under
`references\specs\<spec>\<version>\` or
`references\tools\<tool>\<version>\` — never overwrite an existing
version directory. Update `references\README.md` only when a new version
becomes the baseline.

## Spec rules echo-elm must honor

These are the non-negotiable items. Detailed sources live in the
versioned references; the bullets below are pulled in here so that an
agent doesn't need to leave this file to make safe decisions.

### CQL/ELM physical representation

- ELM namespaces: `urn:hl7-org:cql:r1` and `urn:hl7-org:elm:r1`.
- ELM XSDs (load from `references\specs\cql\1.5.3\` once cached):
  `expression.xsd`, `clinicalexpression.xsd`, `library.xsd`,
  `modelinfo.xsd`, `types.xsd`.
- Media types echo-elm must round-trip cleanly: `text/cql`,
  `text/cql-identifier`, `text/cql-expression`, `application/elm+xml`,
  `application/elm+json`.
- ELM JSON mirrors XML; polymorphic discrimination uses the `type`
  attribute, qualified type names use `{namespace}Name`, and there is
  no mixed content.

### Translator option defaults (modern/FHIR artifacts)

From `CqlCompilerOptions.defaultOptions()` and *Using CQL with FHIR 2.0.0*:

- `EnableAnnotations`, `EnableLocators`, `DisableListDemotion`,
  `DisableListPromotion`.
- Method invocation **enabled** (no `DisableMethodInvocation`).
- List traversal **enabled** (no `DisableListTraversal`).
- `validateUnits = true`, `compatibilityLevel = "1.5"`,
  `errorLevel = Info`.
- `SignatureLevel`:
  - **Library API + modern `translate` default:** `Overloads`.
  - **CQF compatibility CLI default:** `None` (matches the CQFramework CLI
    default for `--signatures`). Preserve this
    asymmetry only inside `echo-elm cqf translate`.
- An artifact's FHIR `cqf-cqlOptions` extension on `Library` overrides
  these defaults.

### ELM header requirements (for ELM suitability)

Always record on the ELM `Library`: translator version, compatibility
level, signature level, and the semantically relevant options
(list traversal / demotion / promotion, interval demotion / promotion,
method invocation, `RequireFromKeyword`, `SignatureLevel`,
`validateUnits`).

### ModelInfo

- ModelInfo is versioned to the FHIR version (not the IG version), e.g.
  `Library/FHIR-ModelInfo|4.0.1`; `include hl7.fhir.uv.cql.FHIRHelpers
  version '4.0.1'`.
- Profile-informed vs. derived ModelInfo are distinct strategies and
  must never be silently mixed; record the chosen strategy on the
  artifact.

### Diagnostics

- Severity ladder: `Trace` / `Info` / `Warning` / `Error`.
- Locator format used on stderr: `[startLine:startChar,
  endLine:endChar]`; use `[n/a]` when the locator is missing.
- Per-message line shape: `<Severity>:[<sl>:<sc>, <el>:<ec>] <message>`.
- CQFramework-compatible banner per file (`echo-elm cqf translate`):

  ```
  ================================================================================
  TRANSLATE <inputPath>
  Translation completed successfully.
  ELM output written to: <outputPath>
  ```

  On failure, use `Translation failed due to errors:` followed by the
  diagnostic lines.

### CQL 2 stance

CQL 2.0.0-ballot features are future-facing only. They must never affect 1.5.3
output. Do not add them to the main interface until an explicit experimental
mode is designed.

## CQFramework CLI compatibility (`echo-elm cqf translate`)

The full flag map is in
`references\implementation\cqframework-compatibility.md`. Highlights:

- Required: `--input`. Optional IO: `--output`, `--format`
  (`XML`/`JSON`/`COFFEE`), `--model`, `--root-dir`.
- Toggle flags map 1:1 to `CqlCompilerOptions.Options` values; e.g.
  `--annotations` → `EnableAnnotations`,
  `--disable-list-promotion` → `DisableListPromotion`,
  `--require-from-keyword` → `RequireFromKeyword`,
  `--disable-default-modelinfo-load` → `DisableDefaultModelInfoLoad`,
  `--enable-interval-demotion` → `EnableIntervalDemotion`, etc.
- Diagnostic flags: `--error-level {Trace|Info|Warning|Error}` (default
  `Info`), `--signatures {None|Differing|Overloads|All}` (CLI default
  `None`), `--compatibility-level {1.3|1.4|1.5}` (default `1.5`).
- Composite flags:
  - `--debug` expands to `--annotations --locators --result-types`.
  - `--strict` expands to `--disable-list-traversal
    --disable-list-demotion --disable-list-promotion
    --disable-method-invocation --require-from-keyword` (and in 4.x
    also `--validate-units`).
- Output extension chosen from format when `--output` is a directory.
- `--verify` parses + analyses without writing any ELM file.

The option enum and defaults are **identical** between 3.29.0 and
4.8.0; the deltas worth recording are the Maven group change
(`info.cqframework` → `org.cqframework`), the Java → Kotlin port, and
the 4.x-only `reportSelectivity` library knob.

## Working in this repo

- Always use **Windows-style paths** (`\`) when writing examples; the
  primary development environment is Windows + `pwsh`.
- Prefer ecosystem tools (`go mod`, `go test`, `go vet`,
  `golangci-lint`) over manual changes.
- Don't add new lint/build/test tools unless the task requires it.
- When adding a new spec or tool version, create
  `references\<area>\<name>\<version>\README.md` next to the existing
  ones; never overwrite an older version's directory.
- When in doubt about behavior, read the relevant versioned reference
  before changing code.

## Pointers

- Implementation playbook: `references\implementation\echo-elm.md`
- CQFramework CLI compatibility matrix:
  `references\implementation\cqframework-compatibility.md`
- Spec-compatibility checklist:
  `references\implementation\spec-compatibility.md`
- Terminology playbook:
  `references\implementation\terminology.md`
- Spec/tool index: `references\README.md`
- Per-area agent files:
  - `agents\echo-elm.md` — translator/Go-package agent
  - `agents\cqframework-compatibility.md` — compat-mode agent
  - `agents\spec-references.md` — spec lookup agent
  - `agents\local-ui.md` — `echo-elm ui` local API + workbench server
  - `agents\frontend-workbench.md` — SvelteKit workbench agent
  - `agents\mcp-server.md` — MCP server agent
  - `agents\parity-harness.md` — CQFramework parity-test agent
- Project-local skills:
  - `.claude\skills\echo-elm\SKILL.md`
  - `.claude\skills\quality-measure-spec-compat\SKILL.md`
  - `.claude\skills\echo-qm-reference\SKILL.md` (reference only)

## Planning artifacts (live documents)

These are the operational source of truth during implementation. Keep them
updated at every phase boundary or material decision.

- `specs\echo-elm\spec.md` — product & system specification
- `plan.md` — phased implementation plan + decision log
- `tasks.md` — granular project task list
- `contracts\`
  - `go-api.md` — `pkg/echoelm` exported surface
  - `cli.md` — CLI flag/banner parity surface
  - `local-api.md` — loopback HTTP endpoints serving the workbench
  - `mcp-tools.md` — MCP tool inventory
  - `frontend.md` — Svelte workbench surface
  - `parity-harness.md` — CQFramework parity-test contract
- `docs\`
  - `architecture.md` — layering & dependency rules
  - `dev-environment.md` — toolchains & common loops
  - `testing-strategy.md` — test layers & DOD for public surfaces

## Tech stack (locked)

- **Language:** Go 1.26.3 (installed). One binary, multiple subcommands
  (`translate`, `cqf translate`, `ui`, `mcp`, `parity`, `version`).
- **Frontend:** SvelteKit with `adapter-static`, embedded via `//go:embed`.
  Lightweight stack — Tailwind + CodeMirror 6 + native `fetch`; no design
  system, no codegen, no state library.
- **State/persistence:** no database and no service-owned state. CQL
  libraries are folders on disk (CQFramework convention). CLI translations
  write ELM files next to inputs (or to `--output`). Parity runs write reports
  to `<workspace>/parity/runs/<id>/`. The `echo-elm ui` server itself is
  stateless and loopback-only.
- **Python:** managed with `uv` (`uvx` for one-shots) when needed for
  helper scripts. Never required at Go runtime.
- **MCP:** `echo-elm mcp` subcommand over stdio.
- **MCP SDK:** use the official Model Context Protocol Go SDK
  (`github.com/modelcontextprotocol/go-sdk/mcp`, unless the official module
  path changes before implementation). Do not use community/third-party MCP
  packages for the protocol layer.
- **Java (parity only):** Temurin 17 LTS, project-local under `tools\jdk\`,
  installed via `task install:jdk`. Required only when running the parity
  harness.
- **Task runner:** [Taskfile.dev](https://taskfile.dev) — `Taskfile.yml`.
- **Commit policy:** Conventional Commits, milestone cadence, **always**
  include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`
  when an AI agent made the commit.

## Dependency policy

Use libraries where they materially improve correctness, compatibility, or
maintenance. This project is **not** stdlib-only. Prefer small, mature,
well-maintained packages with stable APIs and modest transitive dependency
graphs. Before adding a dependency, record why it is needed and inspect
transitives; avoid packages that pull in large frameworks or broad dependency
trees for narrow tasks. Keep the translator core's dependency surface smaller
than CLI/UI/MCP tooling where possible.

## Test-corpus & licensing

- Copy upstream `cqframework/clinical_quality_language` cql-to-elm tests
  **verbatim** into `test\corpus\cqframework\`. Preserve `NOTICE` and add
  `PROVENANCE.md` (source repo, commit SHA, copy date, modifications=none).
- Conditionally import `cqframework/cql-tests` if its license permits a
  vendored copy; otherwise clone at test time under `tools\cql-tests\`
  (gitignored) and reference at runtime.
- echo-elm itself: Apache-2.0.
