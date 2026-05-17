# QI-Core 7.0.2 notes

## Source

- Home: <https://hl7.org/fhir/us/qicore/>

QI-Core 7.0.2 is a FHIR R4 STU7 implementation guide. It defines profiles, extensions, bindings, modelinfo, negation patterns, and quality-focused clinical data constraints used for CQL authoring and evaluation.

## Role in this workspace

QI-Core is the preferred US Realm clinical data model for FHIR-based quality measurement and decision support. It derives from and extends US Core where possible.

## Implementation concerns

- Keep QI-Core modelinfo versioned separately from FHIR R4 modelinfo.
- Use QI-Core retrieve names exactly as provided by the modelinfo, including quoted identifiers where needed.
- Support QI-Core negation patterns; do not translate QDM-style negation assumptions directly into FHIR without profile-specific rules.
- Validate FHIR resources against QI-Core profiles when a measure depends on profile-constrained semantics.
- For performance, decide whether profile conformance is enforced at ingestion, query, retrieve resolution, or pre-indexing.

## QDM relationship

QI-Core includes a QDM-to-QI-Core mapping. Use it for migration and cross-model comparison, but do not treat QDM datatype names and QI-Core profile names as interchangeable.

