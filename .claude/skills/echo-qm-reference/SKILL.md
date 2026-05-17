---
name: echo-qm-reference
description: Reference-only knowledge about digital quality measure (dQM) / CQL+ELM execution engines, FHIR Measure evaluation, MeasureReport generation, DEQM reporting, terminology expansion, and gaps-in-care. Use when the user asks background questions about how an ELM runtime or quality-measure engine works in the context of echo-qm (a separate project that will consume echo-elm output). Do NOT use this skill to drive implementation work inside the echo-elm repository — echo-elm is a translator only.
---

# echo-qm reference skill (reference only)

This skill captures background knowledge about ELM runtimes and FHIR
quality-measure evaluation. It exists for the convenience of contributors
who need to reason about how echo-elm's output is consumed downstream by
`echo-qm` (a separate companion project).

**Scope guardrail:** echo-elm is a translator. It does **not** evaluate
ELM, expand terminology, fetch data, or produce MeasureReports. If a task
needs those capabilities, it belongs in `echo-qm`, not here.

## When to consult this skill

- The user is asking conceptual questions about how downstream engines
  use ELM produced by echo-elm.
- The user is reasoning about translator features that exist *because*
  the runtime needs them (e.g. ELM suitability, annotations, locators,
  result types, signature level).
- The user is sketching the shape of the future `echo-qm` project.

## Background references

- Engine background notes: `references\implementation\dqm-engine.md`
  (legacy filename, reference only — do not adopt "dqm-engine" as a
  project name).
- FHIR R4 runtime: `references\specs\fhir-r4\4.0.1\README.md`
- CQF Measures: `references\specs\cqfmeasures\5.0.0\README.md`
- DEQM: `references\specs\davinci-deqm\5.0.0\README.md`
- CMS dQM definition: `references\specs\cms-dqm\current\README.md`
- Terminology: `references\implementation\terminology.md`
- QI-Core: `references\specs\qicore\7.0.2\README.md`
- QDM: `references\specs\qdm\5.6\README.md`

## Implications for echo-elm

- An ELM runtime needs translator options, compatibility level, and
  signature level on the ELM header to perform ELM suitability checks —
  echo-elm must always emit these.
- Engines treat the chosen ModelInfo strategy as part of artifact
  identity; echo-elm must record it and never mix strategies silently.
- Engines depend on accurate locators and annotations for traceability;
  echo-elm should default these on for FHIR artifacts.
