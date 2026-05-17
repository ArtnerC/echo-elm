# FHIR Quality Measure / CQF Measures 5.0.0 notes

## Sources

- Home: <https://hl7.org/fhir/us/cqfmeasures/>
- Measure conformance: <https://hl7.org/fhir/us/cqfmeasures/measure-conformance.html>
- Using CQL: <https://hl7.org/fhir/us/cqfmeasures/using-cql.html>
- Packaging: <https://hl7.org/fhir/us/cqfmeasures/packaging.html>

The FHIR Quality Measure Implementation Guide 5.0.0 is the current published US Realm guide for representing quality measures with FHIR R4, CQL, and ELM.

## Measure structure

A FHIR-based quality measure consists of:

- a `Measure` resource conforming at least to CRMI shareable/publishable profiles as required,
- a primary `Library` containing CQL and optionally ELM,
- measure group/population/stratifier criteria that reference named expressions,
- terminology and data-requirement metadata,
- packaging metadata for computable or executable distribution.

## CQL requirements

Important conformance constraints:

- CQL must be contained in a CQL library.
- CQL libraries used by measures must include semantic versions in `<major>.<minor>.<patch>` form.
- Expressions directly referenced by a Measure should be contained in a single primary library to avoid requiring qualified identifiers in the Measure.
- All libraries and expressions used directly or indirectly by a measure must use FHIR-based data models.
- Data model declarations must include versions, for example `using FHIR version '4.0.1'`.
- Value sets usually use version-independent canonical URLs; version information is preferably managed externally by a terminology manifest.
- String-based membership testing is discouraged.

## CQL/ELM packaging

For CQL measures:

- computable packages include CQL Library resources,
- executable packages include ELM XML and/or ELM JSON Library resources,
- ELM should be provided for runtimes that cannot compile CQL,
- ELM must be semantically equivalent to the CQL.

Relevant content types:

- `text/cql`
- `application/elm+xml`
- `application/elm+json`

Library content is base64-encoded in FHIR `Attachment.data`.

## Measure packaging order

Measure bundles:

1. First entry: `Measure` resource.
2. Second entry: primary `Library` resource.
3. Optional entries: recursively required libraries.
4. Optional entries: required code systems and value sets.
5. Optional entries: test case bundles.

Library bundles:

1. First entry: primary `Library`.
2. Optional entries: referenced libraries.
3. Optional entries: referenced terminology.

Test case bundles:

1. First entry: expected-outcome `MeasureReport` conforming to the CQFM test case profile.
2. Include all resources needed to evaluate the test case.

## dQM engine implications

A compatible engine must understand proportion, ratio, continuous-variable, cohort, and composite patterns as represented in FHIR Measure. It should evaluate criteria in measure group order, compute population membership and scores, then produce individual, subject-list, summary, or data-collection reports compatible with CQF/DEQM profiles.

