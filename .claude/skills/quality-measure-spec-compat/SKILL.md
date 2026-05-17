---
name: quality-measure-spec-compat
description: Check, design, or fix compatibility with CQL, ELM, FHIR R4, CQF Measures, DEQM, QI-Core, US Core, QDM, CRMI, or CMS dQM specifications. Use this skill whenever the user asks whether an artifact, translator, engine, package, Measure, Library, MeasureReport, or implementation is spec-compatible, conformant, standards-based, or aligned to latest stable versions.
---

# Quality Measure Spec Compatibility Skill

Use this workflow for conformance reviews, compatibility audits, and version-alignment work.

## Baseline references

- Index: `references\README.md`
- Spec checklist: `references\implementation\spec-compatibility.md`
- Terminology: `references\implementation\terminology.md`
- CQL/ELM: `references\specs\cql\1.5.3\README.md`
- FHIR R4: `references\specs\fhir-r4\4.0.1\README.md`
- Using CQL with FHIR: `references\specs\using-cql-with-fhir\2.0.0\README.md`
- CRMI: `references\specs\crmi\1.0.0\README.md`
- CQF Measures: `references\specs\cqfmeasures\5.0.0\README.md`
- DEQM: `references\specs\davinci-deqm\5.0.0\README.md`
- QI-Core: `references\specs\qicore\7.0.2\README.md`
- US Core: `references\specs\us-core\8.0.1\README.md`
- QDM: `references\specs\qdm\5.6\README.md`

## Workflow

1. Identify the artifact type and declared package dependencies.
2. Build a compatibility matrix from actual artifact versions, not assumed latest versions.
3. Check canonical URLs, versions, profiles, content types, translator options, and modelinfo identity.
4. Validate CQL/ELM suitability before runtime execution.
5. Validate Measure, Library, MeasureReport, Bundle, and terminology resources against their profiles.
6. Report incompatibilities as actionable findings with the exact spec layer affected.

## Default stance

Use the latest stable baseline in `references\README.md`, but defer to explicit artifact package dependencies when evaluating a real measure package. Do not upgrade an artifact's assumed spec version merely because a newer guide exists.

