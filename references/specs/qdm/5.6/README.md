# QDM 5.6 notes

## Source

- QDM page: <https://ecqi.healthit.gov/qdm>
- PDF: <https://ecqi.healthit.gov/sites/default/files/QDM-v5.6-508.pdf>

Quality Data Model (QDM) 5.6 is the current QDM version in use, published January 2021. QDM is a conceptual information model for electronic quality performance measurement.

## Role in this workspace

FHIR-based dQMs should be the forward baseline. QDM remains important for:

- legacy eCQMs,
- QDM CQL translator support,
- migration to QI-Core/FHIR,
- comparison with CMS historical measure logic,
- validating cross-model semantics.

## Translator implications

To support QDM CQL:

- include QDM modelinfo for version 5.6,
- resolve QDM datatype names and attributes exactly,
- support QDM-specific timing and negation semantics where represented in modelinfo and CQL,
- test QDM and FHIR measures separately because retrieves, datatypes, and terminology patterns differ.

## Engine implications

A QDM engine path must map source patient data to QDM datatypes before ELM evaluation. Do not feed FHIR resources directly to a QDM ELM runtime without a model adapter.

