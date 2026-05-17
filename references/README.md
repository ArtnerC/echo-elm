# echo-elm reference index

This workspace tracks the latest stable implementation baseline for
building **echo-elm**: a Go CQL→ELM translator (CLI + importable
package) that is fully spec-compliant and CQFramework-CLI-compatible.

Background material about ELM runtimes / FHIR quality-measure
evaluation is kept here as reference only — that work belongs in the
separate `echo-qm` project, not in this repo.

## Latest stable baseline

Use these versions unless an artifact or program explicitly requires
older content:

| Area | Latest stable / current published version | Local reference |
| --- | --- | --- |
| CQL language and ELM schema | CQL 1.5.3, ANSI/HL7 CQLANG R1-2020 (R2025), published 2025-03-07 | `specs\cql\1.5.3\README.md` |
| CQL 2 future work | 2.0.0-ballot, continuous build / trial-use ballot, not the baseline | `specs\cql\2.0.0-ballot\README.md` |
| FHIR base | R4 4.0.1 for current quality-measure IGs | `specs\fhir-r4\4.0.1\README.md` |
| Using CQL with FHIR | 2.0.0 STU2 | `specs\using-cql-with-fhir\2.0.0\README.md` |
| CRMI canonical artifact infrastructure | 1.0.0 STU1 | `specs\crmi\1.0.0\README.md` |
| QI-Core | 7.0.2 STU7 | `specs\qicore\7.0.2\README.md` |
| US Core | 8.0.1 STU8 | `specs\us-core\8.0.1\README.md` |
| QDM | 5.6, published January 2021 | `specs\qdm\5.6\README.md` |
| CQFramework translator (current) | `org.cqframework:cql-to-elm-jvm:4.8.0` | `tools\cqframework-cql\4.8.0\README.md` |
| CQFramework translator (legacy) | `info.cqframework:cql-to-elm:3.29.0` | `tools\cqframework-cql\3.29.0\README.md` |
| JavaScript CQL execution (engine-side ref) | `cql-execution@3.3.0` | `tools\cql-execution\3.3.0\README.md` |

The following are kept as reference only — they describe downstream
ELM-runtime / measure-evaluation behavior that lives in the separate
`echo-qm` project:

| Area | Version | Local reference |
| --- | --- | --- |
| FHIR Quality Measure / CQF Measures | 5.0.0 STU5 | `specs\cqfmeasures\5.0.0\README.md` |
| Da Vinci DEQM | 5.0.0 STU5 | `specs\davinci-deqm\5.0.0\README.md` |
| CMS digital quality measures | Current CMS dQM definition | `specs\cms-dqm\current\README.md` |

## Implementation playbooks

- `implementation\echo-elm.md` — echo-elm architecture, conformance,
  acceptance criteria (this is the primary translator playbook).
- `implementation\cqframework-compatibility.md` — CQFramework
  `cql-to-elm` CLI compatibility matrix for 3.29.0 and 4.8.0+.
- `implementation\spec-compatibility.md` — wider spec compatibility
  matrix, packaging rules, and validation checklist.
- `implementation\terminology.md` — terminology resolution, value-set
  expansion, and version handling (echo-elm exposes the interface;
  resolution itself is pluggable).
- `implementation\dqm-engine.md` — **reference only**, kept for
  `echo-qm` background. Do not adopt "dqm-engine" as a project name.

## Project-local skills

Project-local skills live in `.claude\skills`:

- `.claude\skills\echo-elm\SKILL.md` — translator implementation skill.
- `.claude\skills\quality-measure-spec-compat\SKILL.md` — compatibility
  audit skill.
- `.claude\skills\echo-qm-reference\SKILL.md` — reference-only skill
  for the downstream engine (separate project).

Skills are intentionally small and point back into `references\...` for
versioned detail.

## Agent files

- `..\AGENTS.md` — repo-wide agent entry point.
- `..\agents\echo-elm.md` — translator/Go-package agent.
- `..\agents\cqframework-compatibility.md` — compat-mode agent.
- `..\agents\spec-references.md` — spec lookup agent.
- `..\agents\local-ui.md` — loopback UI API + embedded workbench server.
- `..\agents\frontend-workbench.md` — SvelteKit workbench agent.
- `..\agents\mcp-server.md` — MCP server agent.
- `..\agents\parity-harness.md` — CQFramework parity harness agent.

## Versioning convention

References are stored under `specs\<spec-name>\<version>\` or
`tools\<tool-name>\<version>\`. Add future versions side-by-side; do
not overwrite prior version folders. Add a new row to this index when a
new version becomes the implementation baseline.
