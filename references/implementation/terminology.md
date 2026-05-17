# Terminology implementation playbook

## Target

Provide deterministic terminology behavior for CQL translation and dQM evaluation, including code system resolution, value set expansion, direct-reference code validation, and membership testing.

## Required capabilities

- Resolve `codesystem`, `valueset`, `code`, and `concept` declarations from CQL.
- Resolve FHIR `CodeSystem` and `ValueSet` canonical URLs with optional versions.
- Expand value sets at a declared expansion time or with a declared terminology manifest.
- Test membership for `Code`, `Coding`, `CodeableConcept`, and model-specific coded elements.
- Preserve system, code, display, version, and value set version details in traces.
- Fail explicitly when terminology cannot be resolved, unless the caller selected a documented permissive validation mode.

## CQL/FHIR conventions

- CQL value set declarations use canonical URLs; VSAC value sets commonly use `http://cts.nlm.nih.gov/fhir/ValueSet/<OID>`.
- A version-specific value set may be represented with `url|version`.
- Prefer external terminology manifests for measure packages rather than hardcoding every value set version in CQL.
- String-based membership testing is discouraged for quality measures.
- UCUM units are terminology-like dependencies for quantities; validate with a UCUM service when translation/evaluation requires it.

## Runtime architecture

1. **Canonical resolver**: maps URL/version to local package resources or remote terminology service calls.
2. **Expansion cache**: stores expansion results by URL, version, parameters, and expansion date.
3. **Membership service**: evaluates `InValueSet`, `InCodeSystem`, direct-reference codes, and CodeableConcept membership.
4. **Manifest support**: pins code system and value set versions for a measure evaluation run.
5. **Trace recorder**: records which terminology versions and expansions were used.

## Acceptance criteria

- Same measure, data, and terminology manifest produces identical population results.
- Missing value sets, unsupported code systems, and expansion failures surface as diagnostics or OperationOutcome issues.
- Direct-reference codes are validated against declared code systems when validation mode requires it.
- Expansion cache keys include enough inputs to avoid reusing incompatible expansions.
- Results include terminology provenance suitable for debugging measure differences.

