# Agent: spec references

Use this file as the lookup index when you need spec-level facts to
make a decision in echo-elm. It surfaces the bits you'll reach for most
often, with pointers into the versioned reference library under
`references\` for the long-form material.

## CQL 1.5.3 (baseline)

- Source of truth: `references\specs\cql\1.5.3\README.md`.
- Official: ANSI/HL7 CQLANG R1-2020 (R2025), published 2025-03-07.
- Authoring + Developer guides at <https://cql.hl7.org/>.
- ELM namespaces: `urn:hl7-org:cql:r1`, `urn:hl7-org:elm:r1`.
- Schemas: `expression.xsd`, `clinicalexpression.xsd`, `library.xsd`,
  `modelinfo.xsd`, `types.xsd`.
- Media types: `text/cql`, `text/cql-identifier`,
  `text/cql-expression`, `application/elm+xml`, `application/elm+json`.
- ELM JSON mirrors XML; polymorphic types use the `type` attribute,
  qualified type names use `{namespace}Name`, no mixed content.

## CQL 2.0.0-ballot (future, experimental)

- Source of truth: `references\specs\cql\2.0.0-ballot\README.md`.
- Continuous build, trial-use ballot. Not baseline.
- Do not activate ballot behavior in CQL 1.5 output. Main-interface CQL 2
  support is deferred until an explicit experimental mode is designed.

## Using CQL with FHIR 2.0.0 STU2

- Source of truth:
  `references\specs\using-cql-with-fhir\2.0.0\README.md`.
- Defines the recommended translator options for FHIR artifacts that
  echo-elm uses as defaults: `EnableAnnotations`, `EnableLocators`,
  `DisableListDemotion`, `DisableListPromotion`; method invocation
  enabled; list traversal enabled; `SignatureLevel >= Overloads` for
  library callers; `validateUnits = true`.
- Defines **ELM suitability** rules — the runtime compares translator
  version, compatibility level, signature level, and the
  semantically-relevant options (list traversal/demotion/promotion,
  interval demotion/promotion, method invocation, `RequireFromKeyword`,
  `SignatureLevel`). echo-elm must record all of these on the ELM
  header.
- FHIR ModelInfo is versioned to the FHIR version, not the IG. e.g.
  `Library/FHIR-ModelInfo|4.0.1`, `include hl7.fhir.uv.cql.FHIRHelpers
  version '4.0.1'`.
- The `cqf-cqlOptions` extension on a `Library` overrides our defaults.

## FHIR R4 4.0.1

- Source of truth: `references\specs\fhir-r4\4.0.1\README.md`.
- Base FHIR version for current quality-measure IGs.

## CRMI 1.0.0

- Source of truth: `references\specs\crmi\1.0.0\README.md`.
- Canonical artifact infrastructure that the wider quality-measure
  ecosystem assumes for packaging and versioning of Library / Measure /
  ValueSet / CodeSystem resources.

## CQF Measures 5.0.0 STU5

- Reference (consumer-side only — echo-elm does not evaluate measures):
  `references\specs\cqfmeasures\5.0.0\README.md`.
- Tells you what shape a Library packaged for FHIR Measure execution
  takes; relevant when echo-elm is invoked as part of an authoring
  pipeline.

## QI-Core 7.0.2 STU7 / US Core 8.0.1 STU8 / QDM 5.6

- `references\specs\qicore\7.0.2\README.md`
- `references\specs\us-core\8.0.1\README.md`
- `references\specs\qdm\5.6\README.md`
- Used when constructing or loading ModelInfo for typed CQL.

## CQFramework tooling

- 4.8.0 current line: `references\tools\cqframework-cql\4.8.0\README.md`
- 3.29.0 legacy line: `references\tools\cqframework-cql\3.29.0\README.md`
- Treat these as the reference implementation for CLI semantics and
  default options (see `agents\cqframework-compatibility.md`).

## Lookup heuristics

- Need defaults for a translator option? → Using CQL with FHIR 2.0.0
  + the CQFramework `CqlCompilerOptions.defaultOptions()` snapshot in
  `references\tools\cqframework-cql\4.8.0\README.md`.
- Need CLI flag semantics? →
  `references\implementation\cqframework-compatibility.md`.
- Need ELM physical-representation rules? → CQL 1.5.3 reference + the
  XSDs cached alongside it.
- Need a CQL operator's defined behavior? → the CQL 1.5.3 spec (HTML
  at <https://cql.hl7.org/02-authorsguide.html> and
  <https://cql.hl7.org/09-b-cqlreference.html>).
- Need ModelInfo guidance? → Using CQL with FHIR 2.0.0 reference.
- Need terminology contract? →
  `references\implementation\terminology.md`.

## When refs disagree

CQL spec > Using CQL with FHIR > CRMI > CQFramework tool behavior >
echo-elm convention. Record any deliberate divergence from the
reference implementation in
`references\implementation\cqframework-compatibility.md` with a
rationale.
