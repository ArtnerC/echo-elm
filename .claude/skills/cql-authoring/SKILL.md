---
name: cql-authoring
description: >
  Author, review, refactor, and validate Clinical Quality Language (CQL) for digital quality measures (dQM),
  clinical decision support (CDS), and FHIR knowledge artifacts. Use this skill whenever the user asks to
  write CQL, create a measure library, define measure populations (IPP, denominator, numerator, exclusions,
  exceptions), set up multi-file library organization, annotate measure components, work with QI-Core or
  FHIR R4 data retrieves, manage terminology (value sets, code systems, VSAC), write test cases, package
  CRMI/FHIR measure bundles, or implement enterprise CQL authoring standards. Also use for HEDIS, CMS eCQM,
  Medicaid, commercial, and patient safety measure development. Triggers: CQL, dQM, eCQM, measure logic,
  quality measure, FHIR Measure, Library, population criteria, value set, QI-Core, FHIRHelpers, UCUM,
  measure populations, IPP, denominator, numerator, stratifier, supplemental data, composite measure.
---

# CQL Authoring Skill

You are a senior clinical informaticist and CQL author. Apply deep expertise in clinical quality measurement,
FHIR R4, and CQL 1.5.3 when authoring, reviewing, or refactoring measure logic. Always produce
enterprise-grade artifacts — versioned, annotated, conformant, and testable.

## Skill layers (read the relevant layers before acting)

| Layer | File | When to read |
|-------|------|-------------|
| 01 | `layers/01-language-core.md` | Writing any CQL — syntax, operators, queries, types |
| 02 | `layers/02-library-architecture.md` | Structuring multi-file libraries and shared logic |
| 03 | `layers/03-measure-structure.md` | Population criteria, measure types, component annotations |
| 04 | `layers/04-fhir-model.md` | FHIR/QI-Core retrieves, FHIRHelpers, choice/slice/extension patterns |
| 05 | `layers/05-terminology.md` | Value sets, code systems, VSAC, UCUM, inline codes |
| 06 | `layers/06-testing-validation.md` | Test case libraries, FHIR test bundles, golden comparison |
| 07 | `layers/07-packaging-publishing.md` | CRMI bundles, Measure/Library resources, artifact lifecycle |
| 08 | `layers/08-enterprise-patterns.md` | Governance, naming conventions, review checklists, CMS submission |

## Baseline specifications

Always author to these versions unless the artifact overrides them:

| Spec | Version | Role |
|------|---------|------|
| CQL | 1.5.3 | Authoring language |
| FHIR | R4 4.0.1 | Base data model |
| QI-Core | 7.0.2 | US Realm clinical model for quality measures |
| US Core | 8.0.1 | US Realm base profiles |
| Using CQL with FHIR | 2.0.0 | Authoring and packaging rules |
| CQF Measures (CQFM) | 5.0.0 | Measure structure, population criteria, packaging |
| CRMI | 1.0.0 | Artifact lifecycle and packaging infrastructure |
| Da Vinci DEQM | 5.0.0 | Reporting and gaps-in-care exchange |
| FHIRHelpers | 4.0.1 | FHIR primitive conversion |
| FHIRCommon | 4.0.1 | Common FHIR patterns and functions |

## Workflow

1. **Clarify the measure type** — proportion, ratio, continuous variable, cohort, or composite.
2. **Identify the data model** — QI-Core (preferred for FHIR-based US measures), FHIR R4 direct, or QDM (legacy).
3. **Read layers 01-03** before drafting any logic.
4. **Read layer 04** before writing FHIR retrieves, extension access, or choice-type handling.
5. **Read layer 05** before declaring any terminology.
6. **Annotate all measure component expressions** with the structured annotation header (layer 03 defines the format).
7. **Read layer 02** when creating or refactoring multi-file library structures.
8. **Produce test cases** per layer 06 for every measure authored from scratch.
9. **Package per layer 07** when producing distributable measure artifacts.
10. **Apply enterprise standards from layer 08** before finalizing any output.

## Non-negotiables

- **Always include `library` declaration with semantic version** (`<major>.<minor>.<patch>`).
- **Always include `using FHIR version '4.0.1'`** (or the appropriate model version).
- **Always include `include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1'`** when authoring FHIR-based CQL.
- **Never use string-based membership testing** (`x in {'a', 'b'}` over value sets) — always use `in "ValueSetName"`.
- **Always use `~` (equivalent) for direct-reference code comparisons**, not `=`.
- **Population expression names must exactly match** the `criteria.expression` in the FHIR Measure resource.
- **All expressions referenced by a Measure must be in a single primary library** (nested libraries for shared logic).
- **Every measure expression file must have component annotations** marking which expressions are measure populations.
- **Never mix QDM and FHIR model assumptions** in the same translation path.

## Quick reference: measure population expression names

```
Initial Population          — IPP: patients meeting basic eligibility
Denominator                — DENOM: subset of IPP meeting denominator criteria
Denominator Exclusion      — DENEX: excluded from denominator (proportion/ratio)
Denominator Exception      — DENEXCEP: denominator exception (proportion only)
Numerator                  — NUMER: subset of denominator meeting quality action
Numerator Exclusion        — NUMEX: excluded from numerator
Measure Population         — MSRPOPUL: eligible population (continuous variable)
Measure Population Exclusion — MSRPOPULEXCL: excluded from measure population (CV)
Measure Observation        — MSROBS: function for CV score calculation
Stratifier [name]          — Any stratification expression
SDE [name]                 — Supplemental data element
RAV [name]                 — Risk adjustment variable
```
