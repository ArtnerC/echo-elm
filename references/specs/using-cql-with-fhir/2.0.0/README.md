# Using CQL with FHIR 2.0.0 implementation notes

## Source

- Home: <https://hl7.org/fhir/uv/cql/>
- Using ELM: <https://hl7.org/fhir/uv/cql/using-elm.html>
- Using ModelInfo: <https://hl7.org/fhir/uv/cql/using-modelinfo.html>

Using CQL with FHIR 2.0.0 is a universal-realm STU2 implementation guide based on FHIR R4. It defines common CQL, ELM, modelinfo, packaging, and evaluation support for FHIR knowledge artifacts.

## Recommended translator options for FHIR knowledge artifacts

Use these defaults when no artifact-specific `cqf-cqlOptions` extension is present:

| Option | Recommendation |
| --- | --- |
| `EnableAnnotations` | SHOULD use |
| `EnableLocators` | SHOULD use |
| `DisableListDemotion` | SHOULD use |
| `DisableListPromotion` | SHOULD use |
| `DisableMethodInvocation` | SHOULD NOT use for FHIR artifacts |
| `EnableDateRangeOptimization` | MAY use |
| `EnableResultTypes` | MAY use |
| `EnableDetailedErrors` | SHOULD NOT use for distributed artifacts |
| `DisableListTraversal` | SHOULD NOT use |
| `SignatureLevel` | SHOULD be `Overloads` or `All` |

When `cqf-cqlOptions` is present on a Library, asset collection, or implementation guide, it drives translation and takes precedence according to the IG.

## ELM suitability checks

Before executing packaged ELM:

1. If overloaded functions exist, require `SignatureLevel` of `Overloads` or `All`.
2. Ensure compatibility level is consistent with the translator/runtime.
3. Prefer matching translator versions between generation and evaluation.
4. Require semantically relevant options to match: list traversal, list demotion, list promotion, interval demotion/promotion, method invocation, `RequireFromKeyword`, and `SignatureLevel`.
5. Use annotations, locators, and result types for authoring/debugging but do not rely on them for runtime semantics.

## FHIR ModelInfo

FHIR R4 modelinfo is versioned with the FHIR version, not the IG version. For FHIR R4, reference:

- `http://hl7.org/fhir/uv/cql/Library/FHIR-ModelInfo|4.0.1`
- `include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1'`

FHIR primitive values are represented as FHIR structures; authors often need `.value` unless FHIRHelpers implicit conversions are available.

## Derived and profile-informed ModelInfo

Derived ModelInfo creates CQL classes for profiles derived from base resources and can preserve reuse across IGs. Profile-informed ModelInfo adds first-class profile slice/extension access and translator target mappings so generated ELM evaluates against base FHIR resources. Existing QI-Core and US Core authoring has used profile-informed approaches; newer tooling may prefer derived modelinfo for reuse.

For implementation, keep modelinfo generation strategy explicit and versioned. Do not mix derived and profile-informed assumptions in one compiled artifact without recording translator options and modelinfo identity.

