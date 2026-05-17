---
name: echo-elm
description: Build, debug, or design the echo-elm Go CQL-to-ELM translator (CLI + importable package) targeting full CQL 1.5.3 compliance and CQFramework cql-to-elm CLI compatibility for both info.cqframework 3.29.0 and org.cqframework 4.8.0+. Use this skill whenever the user mentions echo-elm, CQL parsing, CQL-to-ELM translation, ELM XML/JSON output, ELM suitability, translator options, ModelInfo, FHIRHelpers, ANTLR CQL grammar, jopt-simple flag parity, signature levels, compatibility level, or the cql-to-elm CLI.
---

# echo-elm skill

Use this workflow for any implementation work on `echo-elm`: the Go
implementation of a fully spec-compliant CQL→ELM translator that ships as a
modern CLI, an importable Go package, a loopback workbench, and an MCP server.
CQFramework `cql-to-elm` CLI compatibility is available through the
`echo-elm cqf translate` surface so legacy behavior does not leak into the
main interface.

## Project naming

- **echo-elm** — this project. Go CQL→ELM translator.
- **echo-qm** — separate companion project (different repo). It will be
  the dQM / measure-evaluation engine that consumes ELM produced by
  echo-elm.
- Never use the names "cql-to-elm-translator" or "dqm-engine" to refer to
  these projects in new content; those terms remain only as generic
  descriptions of the *kind* of software.

## Baseline references

Always read the relevant versioned references before changing behavior:

- Project-wide instructions: `AGENTS.md`
- echo-elm playbook: `references\implementation\echo-elm.md`
- CQFramework CLI compatibility: `references\implementation\cqframework-compatibility.md`
- CQL/ELM: `references\specs\cql\1.5.3\README.md`
- Using CQL with FHIR: `references\specs\using-cql-with-fhir\2.0.0\README.md`
- CQFramework 4.8.0 tool: `references\tools\cqframework-cql\4.8.0\README.md`
- CQFramework 3.29.0 tool: `references\tools\cqframework-cql\3.29.0\README.md`

CQL 2.0.0-ballot content is future-facing only — do not let it leak into
1.5 behavior.

## Workflow

1. Identify the affected layer: lex/parse, semantic analysis, type
   system, ELM build, serialization, resolver, diagnostics, CLI, or
   public Go API.
2. Confirm the CQL version (default 1.5.3), data model, output format,
   options, and whether the task uses modern `translate` or CQFramework
   compatibility mode (`cqf translate`).
3. Load the relevant versioned references; do not change behavior from
   memory.
4. Preserve semantic equivalence between source CQL and emitted ELM.
5. Record translator options and compatibility level on the ELM header.
6. Keep library, modelinfo, terminology, and UCUM resolution behind
   pluggable interfaces; do not hard-code providers.
7. Validate generated ELM against the published XSD and against a
   golden corpus diffed with the CQFramework CLI output.

## Recommended defaults

For modern ELM/FHIR knowledge-artifact output:
`EnableAnnotations`, `EnableLocators`, `DisableListDemotion`,
`DisableListPromotion`, method invocation enabled, list traversal
enabled, `validateUnits = true`, `compatibilityLevel = "1.5"`,
`errorLevel = Info`. SignatureLevel default is `Overloads` for the library API
and modern `translate`; CQFramework compatibility mode uses `None` to match
the upstream `--signatures` default.

## Watch-outs

- Do not silently fall back when a library, modelinfo, value set, code
  system, or unit cannot be resolved — emit a diagnostic.
- Do not mix QDM and FHIR model assumptions in one translation path.
- Do not let CQL 2 ballot behavior affect CQL 1.5.3 output.
- Do not conflate the CQFramework tool version with the CQL spec version.
- CQFramework compatibility defaults differ from modern echo-elm defaults;
  preserve the asymmetry only in `echo-elm cqf translate`.
