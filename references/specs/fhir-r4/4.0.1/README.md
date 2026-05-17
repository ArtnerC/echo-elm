# FHIR R4 4.0.1 quality runtime notes

## Sources

- Measure: <https://hl7.org/fhir/R4/measure.html>
- Library: <https://hl7.org/fhir/R4/library.html>
- MeasureReport: <https://hl7.org/fhir/R4/measurereport.html>
- `$evaluate-measure`: <https://hl7.org/fhir/R4/measure-operation-evaluate-measure.html>

Current quality-measure implementation guides in this workspace are based on **FHIR R4 4.0.1**.

## Key resources

- `Measure` defines the quality measure and references logic libraries. The Measure usually contains population criteria references, not the full logic.
- `Library` packages CQL, ELM, modelinfo, parameters, dependencies, and data requirements. Embedded `content.data` takes precedence over external `content.url`.
- `MeasureReport` represents evaluation output at individual, subject-list, summary, or data-collection level.

## `$evaluate-measure`

Operation URL patterns:

- `[base]\Measure\$evaluate-measure`
- `[base]\Measure\[id]\$evaluate-measure`

Important parameters:

- `periodStart` and `periodEnd` are required.
- `measure` is needed when invoked on the type rather than an instance.
- `reportType` can be `subject`, `subject-list`, or `population` in base R4; output maps to MeasureReport report types used by guides.
- `subject` constrains evaluation to one or more subjects.
- `practitioner` constrains subjects by practitioner relationship when supported.
- `lastReceivedOn` is for patient-level result filtering.

Implementation requirements:

- Build a `Measurement Period` parameter from `periodStart` and `periodEnd` using FHIR date boundary semantics.
- Return a `MeasureReport`, directly if it is the only out parameter.
- Return pending/error statuses explicitly if asynchronous or failed evaluation is supported.
- Do not silently persist generated reports unless server behavior documents that side effect.

## Runtime implications

A dQM engine needs:

- canonical resolution for Measure and Library URLs with optional versions,
- dependency expansion through `relatedArtifact`,
- CQL/ELM attachment decoding by content type,
- FHIR resource query/patient-source abstraction,
- terminology expansion and membership testing,
- report assembly that respects FHIR Measure and MeasureReport cardinalities.

