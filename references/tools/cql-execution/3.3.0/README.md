# cql-execution 3.3.0 notes

## Sources

- Repository: <https://github.com/cqframework/cql-execution>
- Latest release: <https://github.com/cqframework/cql-execution/releases/tag/v3.3.0>
- npm package: <https://registry.npmjs.org/cql-execution/latest>

`cql-execution` is a TypeScript/JavaScript library for executing JSON ELM. Version 3.3.0 was published in August 2025.

## Scope

`cql-execution` implements core CQL logical constructs. It does not by itself provide robust data-model or terminology support. Use:

- a PatientSource/data provider for the data model,
- a CodeService/terminology provider,
- `cql-exec-fhir` for FHIR patient data,
- `cql-exec-vsac` for VSAC terminology,
- `cqm-execution` for QDM eCQM execution patterns.

## Known limitations to consider

The project notes limitations including:

- no direct robust support for any specific data model in the core package,
- evolving PatientSource, CodeService, and Results APIs,
- JavaScript Number precision differences from CQL Integer/Decimal semantics,
- incomplete support for some CQL 1.5 trial-use features and advanced retrieves,
- incomplete support for external functions and related/unfiltered context retrieves.

## Engine implementation lesson

Do not equate ELM evaluation with a full dQM engine. A full engine must add:

- precise CQL type semantics,
- FHIR/QI-Core/QDM model adapters,
- terminology expansion/membership,
- measure scoring and aggregation,
- MeasureReport/DEQM packaging,
- deterministic execution traces and diagnostics.

