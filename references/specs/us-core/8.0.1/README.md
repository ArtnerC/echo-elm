# US Core 8.0.1 notes

## Source

- Home: <https://hl7.org/fhir/us/core/>

US Core 8.0.1 is the current published US Realm STU8 guide based on FHIR R4. It defines the floor for patient clinical data access and profiles used by US Realm FHIR implementation guides.

## Role in dQM work

US Core is the base interoperability layer that QI-Core builds on. A dQM engine should be able to ingest and normalize US Core conformant resources even when a measure uses QI-Core retrieve/profile names.

## Implementation concerns

- Track US Core and QI-Core versions independently; QI-Core 7.0.2 aligns with US Core STU7, while current US Core is 8.0.1.
- Do not assume every current US Core profile exists in the QI-Core baseline used by a measure.
- Use canonical profile URLs and package dependencies to determine actual required profile versions.
- Validate Must Support and mandatory elements according to the IG when claiming conformance; for measure evaluation, also validate measure-specific data requirements.

