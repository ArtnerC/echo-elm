# CMS digital quality measure notes

## Source

- CMS/eCQI dQM page: <https://ecqi.healthit.gov/dqm>

CMS defines digital quality measures (dQMs) as quality measures that:

- use standardized digital data from one or more health information sources,
- are captured and exchanged via interoperable systems,
- apply standards-based quality measure specifications that use code packages,
- are computable in an integrated environment.

## Implications for this project

A dQM engine is more than a CQL evaluator. It needs:

- standards-based artifact ingestion,
- FHIR/API data acquisition or import,
- terminology services and version manifests,
- CQL-to-ELM translation or executable ELM ingestion,
- deterministic measure calculation,
- FHIR MeasureReport and DEQM-compatible output,
- test-case execution and traceability,
- support for quality improvement workflows, not only retrospective reporting.

eCQMs are a subset of dQMs. The forward-looking path is FHIR-based, interoperable, and reusable across sources such as EHRs, labs, devices, assessments, patient applications, registries, and HIEs.

