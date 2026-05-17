# dQM engine implementation playbook

> **Status: reference only.** This file documents what a dQM /
> measure-evaluation engine needs to do. echo-elm is a translator and
> does **not** implement any of this. The companion project that will
> implement these capabilities is `echo-qm`, in a separate repository.
> The name "dqm-engine" is **not** used as a project name in this org.

## Target

Build a standards-based engine that can ingest FHIR quality measure packages, compile or load CQL/ELM, evaluate measures over FHIR/QI-Core/QDM-compatible data, and emit FHIR/DEQM-compatible reports.

## Runtime pipeline

1. **Artifact load**: ingest FHIR Bundles, NPM packages, directories, or canonical URLs.
2. **Canonical resolution**: resolve Measure, Library, ValueSet, CodeSystem, StructureDefinition, and dependencies by canonical URL and version.
3. **Content extraction**: decode CQL/ELM Library attachments, prefer embedded content over external URLs, validate content type.
4. **Translation**: compile CQL to ELM if executable ELM is absent or unsuitable.
5. **ELM suitability**: compare translator version, compatibility level, signature level, and semantic options before reuse.
6. **Data acquisition**: query/import FHIR resources for subject(s), measurement period, attribution, and data requirements.
7. **Terminology**: expand value sets and evaluate code membership with version manifests when available.
8. **Evaluation**: execute ELM by context, subject, library, and measure group.
9. **Measure scoring**: calculate population membership, stratifiers, supplemental data, observations, exclusions/exceptions, and score.
10. **Reporting**: emit MeasureReport and optional Bundles conforming to CQF Measures and DEQM profiles.
11. **Traceability**: keep per-expression evaluation traces, evaluated resources, criteria references, terminology versions, and diagnostics.

## Engine surfaces

- `evaluateMeasure(measure, period, options) -> MeasureReport`
- `evaluateSubject(measure, subject, period, options) -> MeasureReport`
- `collectDataRequirements(measure/library) -> DataRequirement[]`
- `packageMeasure(measure, capabilities, terminologyCapabilities) -> Bundle`
- `careGaps(params) -> Parameters<Bundle[]>`
- `validateArtifact(bundle/package) -> OperationOutcome`

Use `references\implementation\terminology.md` for terminology resolution, expansion, membership, cache, and trace requirements.

## Semantics to implement carefully

- CQL three-valued null logic.
- Date/time precision and timezone behavior.
- Interval boundaries and open/closed semantics.
- Quantity comparison and UCUM unit conversion.
- List equality/equivalence and null items.
- Context switching and retrieves.
- Terminology versioning and value set expansion dates.
- Population criteria order and scoring formula differences.
- Stratifier and supplemental data expansion.
- Composite measure aggregation.
- QI-Core negation and QDM negation differences.

## Data model adapters

Maintain separate adapters:

- `FHIR R4`: base FHIR resources and FHIRHelpers.
- `QI-Core`: profile-aware retrieves and authoring names.
- `US Core`: ingestion/validation foundation.
- `QDM 5.6`: legacy eCQM model adapter; do not mix directly with FHIR resources.

## Reporting modes

Support:

- individual reports,
- subject-list reports,
- summary reports,
- data-collection reports,
- DEQM individual/summary/subject-list profiles,
- DEQM gaps-in-care Bundles when `$care-gaps` is in scope.

## Acceptance criteria

- Evaluates official-style positive and negative test case Bundles.
- Produces deterministic MeasureReports from fixed inputs.
- Fails loudly with OperationOutcome/diagnostics when artifacts, terminology, modelinfo, or ELM are missing/unsuitable.
- Records enough provenance to explain why a subject did or did not meet each population.
- Validates generated reports against relevant profiles before claiming compatibility.

