# Corpus Provenance

## Source: cqframework/clinical_quality_language

| Field | Value |
|---|---|
| **Repository** | https://github.com/cqframework/clinical_quality_language |
| **Commit SHA** | (fetched from `main` branch, ~2025-05) |
| **Copy date** | 2025-05 |
| **License** | Apache-2.0 (see NOTICE below) |
| **Modifications** | None — verbatim copies |
| **Source path** | `Src/java/cql-to-elm/src/jvmTest/resources/org/cqframework/cql/cql2elm/` |

## Files imported

```
cqf-tests/
  DateTimeLiteralTest.cql       — DateTime/time literal expressions
  DefaultContext.cql            — Default (unnamed) library, null handling
  InTest.cql                    — In operator, Interval range expressions
  ParameterTest.cql             — Tuple/list parameter expressions
  QuantityLiteralTest.cql       — Quantity literals and conversions
  RatioLiteralTest.cql          — Ratio literals
  TestComments.cql              — Comment handling with FHIR model
  TestPatientContext.cql        — Patient context with QDM model
  TenDividedByTwo.cql           — Division and implicit promotion
  TupleDifferentKeys.cql        — Tuple equality with different keys
smoke/
  Minimal.cql                   — echo-elm authored smoke fixture
  IncludeTest.cql               — echo-elm authored smoke fixture
```

## Purpose

These fixtures are used by the parity harness (`echo-elm parity`) to compare
echo-elm's ELM output against the upstream `org.cqframework:cql-to-elm-jvm`
and `info.cqframework:cql-to-elm` JARs. Any parity gap is tracked as a
`differ-json` result in the parity report.
