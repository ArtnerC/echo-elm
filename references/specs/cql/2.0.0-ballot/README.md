# CQL 2.0.0-ballot future-facing notes

## Source

- Continuous build: <https://build.fhir.org/ig/HL7/cql/>

CQL 2.0.0-ballot is a trial-use ballot / continuous build and changes regularly. It is **not** the implementation baseline for this workspace. Use it only for future compatibility research, never as a default behavior target unless the project explicitly opts into CQL Release 2.

## Implementation stance

- Baseline parser/translator/runtime semantics should target CQL 1.5.3.
- Keep feature flags for CQL 2 behavior isolated by compatibility level.
- Add conformance tests per feature before enabling CQL 2 syntax or operators.
- Do not allow CQL 2 ballot behavior to change CQL 1.5.3 output.

