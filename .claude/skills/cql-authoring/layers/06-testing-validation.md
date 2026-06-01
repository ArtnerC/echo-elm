# Layer 06 — Testing and Validation

This layer covers testing strategy for CQL quality measures: unit-level CQL test assertions,
FHIR-based test case bundles, structural validation, and golden file comparison.

---

## Testing pyramid for CQL measures

```
       ┌─────────────────────────────────┐
       │  E2E / Measure Evaluation       │  MeasureReport comparison against
       │  (slowest; validates full stack) │  known-good engine (CQF Ruler, etc.)
       ├─────────────────────────────────┤
       │  FHIR Test Case Bundles         │  Population-level: each case tests one
       │  (one bundle per test patient)   │  patient with specific expected outcome
       ├─────────────────────────────────┤
       │  CQL Expression Tests           │  Fast unit tests; assert individual
       │  (fastest)                      │  expression results given inline data
       └─────────────────────────────────┘
```

---

## CQL expression tests (inline)

CQL unit tests use the `Test` library pattern. Each test is an expression that returns
`true` (pass) or an informative value (inspect):

```cql
library CMS.CMS122-Tests version '12.0.000'

using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers
include CMS.CMS122 version '12.0.000' called Measure

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

context Patient

// ─── Helper: Build a minimal test patient ────────────────────────────────────

// (Tests rely on external test case FHIR bundles for Patient context;
//  these expressions reference measure definitions directly.)

// ─── IPP Tests ────────────────────────────────────────────────────────────────

/*
 * @test: IPP - Patient is in IPP when age 18-75, has diabetes, has qualifying encounter
 * @expected: true
 */
define "Test IPP Patient in Age Range":
  Measure."Patient Age 18 to 75 at Start"

/*
 * @test: IPP - Full IPP evaluation
 * @expected: true (when loaded with IPP test case bundle)
 */
define "Test Initial Population":
  Measure."Initial Population" = true

// ─── Denominator Tests ────────────────────────────────────────────────────────

/*
 * @test: Denominator equals IPP (this measure)
 * @expected: true when IPP is true
 */
define "Test Denominator Equals IPP":
  Measure."Denominator" = Measure."Initial Population"

// ─── Denominator Exclusion Tests ─────────────────────────────────────────────

/*
 * @test: Hospice exclusion fires correctly
 * @expected: true (when loaded with hospice exclusion test bundle)
 */
define "Test Hospice Exclusion":
  Measure."Has Hospice During Measurement Period" = true

// ─── Numerator Tests ──────────────────────────────────────────────────────────

/*
 * @test: Numerator fires when most recent HbA1c > 9.0%
 * @expected: true (with HbA1c result bundle where value = 9.5%)
 */
define "Test Numerator High HbA1c":
  Measure."Numerator" = true

/*
 * @test: Numerator does not fire when most recent HbA1c <= 9.0%
 * @expected: false (with HbA1c result bundle where value = 8.9%)
 */
define "Test Numerator Controlled HbA1c":
  Measure."Numerator" = false

/*
 * @test: HbA1c result value is correctly extracted
 * @expected: 9.5 '%'
 */
define "Test Most Recent HbA1c Value":
  Measure."Most Recent HbA1c Result"
```

---

## FHIR test case bundle structure

Each test case is a FHIR Bundle of type `collection` containing:
- One `Patient` resource
- All relevant clinical resources (Encounter, Condition, Observation, etc.)
- An expected `MeasureReport` (or expectations as a Parameters resource)

### Bundle template

```json
{
  "resourceType": "Bundle",
  "id": "TestCase-CMS122-IPP-True",
  "meta": {
    "profile": [
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/test-case-bundle-cqfm"
    ]
  },
  "type": "collection",
  "entry": [
    {
      "resource": {
        "resourceType": "Patient",
        "id": "patient-ipp-true",
        "meta": {
          "profile": ["http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-patient"]
        },
        "extension": [
          {
            "url": "http://hl7.org/fhir/us/core/StructureDefinition/us-core-race",
            "extension": [
              {
                "url": "ombCategory",
                "valueCoding": {
                  "system": "urn:oid:2.16.840.1.113883.6.238",
                  "code": "2106-3",
                  "display": "White"
                }
              },
              { "url": "text", "valueString": "White" }
            ]
          },
          {
            "url": "http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity",
            "extension": [
              {
                "url": "ombCategory",
                "valueCoding": {
                  "system": "urn:oid:2.16.840.1.113883.6.238",
                  "code": "2186-5",
                  "display": "Not Hispanic or Latino"
                }
              },
              { "url": "text", "valueString": "Not Hispanic or Latino" }
            ]
          }
        ],
        "identifier": [{ "value": "patient-ipp-true" }],
        "name": [{ "family": "TestPatient", "given": ["IPP"] }],
        "gender": "female",
        "birthDate": "1970-06-15"
      }
    },
    {
      "resource": {
        "resourceType": "Encounter",
        "id": "enc-office-visit",
        "meta": {
          "profile": ["http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-encounter"]
        },
        "status": "finished",
        "class": {
          "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
          "code": "AMB"
        },
        "type": [
          {
            "coding": [
              {
                "system": "http://snomed.info/sct",
                "code": "185463005",
                "display": "Visit out of hours"
              }
            ]
          }
        ],
        "subject": { "reference": "Patient/patient-ipp-true" },
        "period": {
          "start": "2024-03-15T09:00:00Z",
          "end": "2024-03-15T10:00:00Z"
        }
      }
    },
    {
      "resource": {
        "resourceType": "Condition",
        "id": "cond-diabetes",
        "meta": {
          "profile": [
            "http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-condition-problems-health-concerns"
          ]
        },
        "clinicalStatus": {
          "coding": [
            {
              "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
              "code": "active"
            }
          ]
        },
        "verificationStatus": {
          "coding": [
            {
              "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
              "code": "confirmed"
            }
          ]
        },
        "category": [
          {
            "coding": [
              {
                "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                "code": "problem-list-item"
              }
            ]
          }
        ],
        "code": {
          "coding": [
            {
              "system": "http://snomed.info/sct",
              "code": "44054006",
              "display": "Diabetes mellitus type 2"
            }
          ]
        },
        "subject": { "reference": "Patient/patient-ipp-true" },
        "onsetDateTime": "2015-01-01"
      }
    },
    {
      "resource": {
        "resourceType": "Observation",
        "id": "obs-hba1c-high",
        "meta": {
          "profile": [
            "http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-observation-lab"
          ]
        },
        "status": "final",
        "category": [
          {
            "coding": [
              {
                "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                "code": "laboratory"
              }
            ]
          }
        ],
        "code": {
          "coding": [
            {
              "system": "http://loinc.org",
              "code": "4548-4",
              "display": "Hemoglobin A1c/Hemoglobin.total in Blood"
            }
          ]
        },
        "subject": { "reference": "Patient/patient-ipp-true" },
        "effectiveDateTime": "2024-07-20T08:30:00Z",
        "valueQuantity": {
          "value": 9.5,
          "unit": "%",
          "system": "http://unitsofmeasure.org",
          "code": "%"
        }
      }
    },
    {
      "resource": {
        "resourceType": "MeasureReport",
        "id": "expected-result-ipp-true",
        "meta": {
          "profile": [
            "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/test-case-cqfm"
          ]
        },
        "extension": [
          {
            "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-isTestCase",
            "valueBoolean": true
          }
        ],
        "modifierExtension": [
          {
            "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-isIncremental",
            "valueBoolean": false
          }
        ],
        "status": "complete",
        "type": "individual",
        "measure": "https://example.org/fhir/Measure/CMS122",
        "subject": { "reference": "Patient/patient-ipp-true" },
        "period": {
          "start": "2024-01-01",
          "end": "2024-12-31"
        },
        "group": [
          {
            "population": [
              { "code": { "coding": [{ "code": "initial-population" }] }, "count": 1 },
              { "code": { "coding": [{ "code": "denominator" }] }, "count": 1 },
              { "code": { "coding": [{ "code": "denominator-exclusion" }] }, "count": 0 },
              { "code": { "coding": [{ "code": "numerator" }] }, "count": 1 }
            ]
          }
        ]
      }
    }
  ]
}
```

---

## Test case naming conventions

| Test scenario | File name pattern | Expected MeasureReport |
|--------------|------------------|------------------------|
| In IPP, In Denominator, In Numerator (passes measure) | `Pass-IPP-DENOM-NUMER.json` | IPP:1, DENOM:1, NUMER:1 |
| In IPP, In Denominator, Not in Numerator | `Fail-IPP-DENOM.json` | IPP:1, DENOM:1, NUMER:0 |
| Not in IPP | `Fail-NotIPP-Age.json` | IPP:0, all:0 |
| In Denominator Exclusion (hospice) | `Excl-DENEX-Hospice.json` | IPP:1, DENOM:1, DENEX:1 |
| Edge case: HbA1c exactly 9.0% | `Edge-HbA1c-9pct.json` | NUMER:0 (not > 9.0%) |
| Edge case: No HbA1c in period | `Edge-NoHbA1c.json` | NUMER:1 (null > 9.0% = null, scores as fail) |

---

## Population-level test coverage matrix

Every measure must have test cases covering:

| Category | Required cases |
|---------|---------------|
| IPP inclusion | 1+ positive; 1+ negative (age boundary, no encounter, no diagnosis) |
| Denominator inclusion | Same as IPP if DENOM = IPP; otherwise additional cases |
| Denominator exclusion | 1+ per exclusion criterion (hospice, frailty+dementia, palliative) |
| Denominator exception | 1+ if applicable |
| Numerator inclusion | 1+ positive (passes quality action) |
| Numerator exclusion | 1+ if applicable |
| Boundary values | Age boundary (17 vs 18, 75 vs 76); date boundaries (Jan 1, Dec 31) |
| Null / missing data | 1+ for each key data element (no HbA1c, no encounters) |
| SDE population | 1+ per race/ethnicity/payer/sex combination if SDE validation is required |

---

## Structural validation checklist

Run before committing any measure library:

```
□ Library name matches file name (CMS.CMS122 → CMS.CMS122.cql)
□ Library version matches FHIR Library resource version
□ All referenced value sets have a declaration in terminology library
□ All referenced code systems have a declaration
□ No value set OIDs used as string literals (must be canonical URL form)
□ UCUM units present on all Quantity literals
□ All population expressions are exported (no `private` modifier)
□ All @measure-component annotations present for IPP/DENOM/NUMER
□ context Patient declared exactly once per library
□ FHIRHelpers version matches FHIR model version (4.0.1)
□ No circular includes
□ ELM produced by echo-elm matches CQF Ruler accepted ELM (parity check)
□ All test cases have expected MeasureReport with population counts
```

---

## Golden file comparison workflow

For regression testing against a known-good ELM output:

1. Translate with CQF `cql-to-elm 4.8.0` → store as `goldens/<profile>/file.xml`
2. Translate with echo-elm → compare to golden (semantic diff, not text diff)
3. For options configurations:
   - `default`: no flags
   - `debug`: `--annotations --locators`
   - `strict`: `--disable-list-traversal --disable-list-demotion --disable-list-promotion`
   - `signatures-overloads`: `--signatures Overloads`
   - `compat-1.4`: `--compatibility-level 1.4`

```shell
# Run parity harness (echo-elm built-in)
echo-elm parity run --corpus test/corpus/cqframework --profile default
echo-elm parity run --corpus test/corpus/cqframework --profile debug
echo-elm parity run --corpus test/corpus/cqframework --profile strict
echo-elm parity run --corpus test/corpus/measures --profile default
```

---

## CQL quality validation rules

These SHOULD generate warnings or errors from your CI pipeline:

```
ERROR:   Undefined expression referenced
ERROR:   Missing required population criteria
ERROR:   Value set URL not in OID-based VSAC format (when VSAC is required)
ERROR:   Quantity literal missing units
WARNING: Denominator does not reference Initial Population expressions
WARNING: Population expression name does not match expected Title Case form
WARNING: Value set version pinned without documented reason
WARNING: Direct-reference code without display value
INFO:    Unused expression in library
INFO:    Unused include
```
