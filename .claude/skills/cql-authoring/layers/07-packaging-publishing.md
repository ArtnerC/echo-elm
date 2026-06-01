# Layer 07 — Packaging and Publishing (CRMI, FHIR IG, eCQM Submission)

This layer covers how to package CQL measures as FHIR knowledge artifacts for distribution,
IG publication, CMS eCQM submission, and integration with downstream engines (echo-qm).

---

## FHIR artifact hierarchy

```
Measure (root artifact)
  └── primary Library (contains all population CQL/ELM)
        ├── terminology Library (value sets + code declarations)
        ├── helper Library (shared functions)
        └── patient-characteristics Library (SDE logic)
```

All of the above are FHIR `Library` resources. They are packaged together
as a FHIR `Bundle` for distribution.

---

## Measure resource template

```json
{
  "resourceType": "Measure",
  "id": "CMS122",
  "meta": {
    "profile": [
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/computable-measure-cqfm",
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/publishable-measure-cqfm",
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/executable-measure-cqfm"
    ]
  },
  "language": "en",
  "extension": [
    {
      "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-populationBasis",
      "valueCode": "Patient"
    },
    {
      "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-scoringUnit",
      "valueCodeableConcept": {
        "coding": [{ "system": "http://unitsofmeasure.org", "code": "/1000.d" }]
      }
    },
    {
      "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-effectivePeriod",
      "valuePeriod": { "start": "2024-01-01", "end": "2024-12-31" }
    },
    {
      "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-softwaresystems",
      "extension": [
        {
          "url": "software",
          "valueCoding": {
            "system": "http://hl7.org/fhir/us/cqfmeasures/CodeSystem/software-system-type",
            "code": "translator"
          }
        },
        { "url": "version", "valueString": "3.0.0" }
      ]
    }
  ],
  "url": "https://madie.cms.gov/Measure/CMS122",
  "identifier": [
    {
      "use": "usual",
      "type": {
        "coding": [{ "system": "http://hl7.org/fhir/us/cqfmeasures/CodeSystem/identifier-type",
                     "code": "short-name" }]
      },
      "value": "CMS122"
    },
    {
      "use": "official",
      "type": {
        "coding": [{ "system": "http://terminology.hl7.org/CodeSystem/v2-0203",
                     "code": "VERS" }]
      },
      "system": "urn:ietf:rfc:3986",
      "value": "urn:hl7ii:2.16.840.1.113883.10.20.28.6.1:2024-01-01"
    }
  ],
  "version": "12.0.000",
  "name": "DiabetesHemoglobinA1cHbA1cPoorControl9",
  "title": "Diabetes: Hemoglobin A1c (HbA1c) Poor Control (> 9%)",
  "status": "active",
  "experimental": false,
  "date": "2024-06-01",
  "publisher": "Centers for Medicare & Medicaid Services (CMS)",
  "contact": [
    {
      "telecom": [{ "system": "url", "value": "https://www.cms.gov" }]
    }
  ],
  "description": "Percentage of patients 18–75 years with diabetes who had hemoglobin A1c > 9.0% (poor control) during the measurement period.",
  "useContext": [
    {
      "code": {
        "system": "http://terminology.hl7.org/CodeSystem/usage-context-type",
        "code": "program"
      },
      "valueCodeableConcept": {
        "coding": [
          {
            "system": "http://hl7.org/fhir/us/cqfmeasures/CodeSystem/quality-programs",
            "code": "ep-ec",
            "display": "EP/EC"
          }
        ]
      }
    }
  ],
  "jurisdiction": [
    {
      "coding": [{ "system": "urn:iso:std:iso:3166", "code": "US" }]
    }
  ],
  "purpose": "The purpose of this measure is to identify patients 18–75 years with diabetes who had hemoglobin A1c test results indicating poor glycemic control.",
  "copyright": "...",
  "approvalDate": "2024-05-01",
  "lastReviewDate": "2024-05-01",
  "effectivePeriod": { "start": "2024-01-01", "end": "2024-12-31" },
  "topic": [
    {
      "coding": [
        {
          "system": "http://loinc.org",
          "code": "57024-2",
          "display": "Health Quality Measure Document"
        }
      ]
    }
  ],
  "library": ["https://madie.cms.gov/Library/CMS122"],
  "disclaimer": "...",
  "scoring": {
    "coding": [
      {
        "system": "http://terminology.hl7.org/CodeSystem/measure-scoring",
        "code": "proportion",
        "display": "Proportion"
      }
    ]
  },
  "type": [
    {
      "coding": [
        {
          "system": "http://terminology.hl7.org/CodeSystem/measure-type",
          "code": "process",
          "display": "Process"
        }
      ]
    }
  ],
  "improvementNotation": {
    "coding": [
      {
        "system": "http://terminology.hl7.org/CodeSystem/measure-improvement-notation",
        "code": "decrease",
        "display": "Decreased score indicates improvement"
      }
    ]
  },
  "group": [
    {
      "id": "group-1",
      "population": [
        {
          "id": "initial-population",
          "code": {
            "coding": [
              {
                "system": "http://terminology.hl7.org/CodeSystem/measure-population",
                "code": "initial-population",
                "display": "Initial Population"
              }
            ]
          },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Initial Population"
          }
        },
        {
          "id": "denominator",
          "code": {
            "coding": [
              {
                "system": "http://terminology.hl7.org/CodeSystem/measure-population",
                "code": "denominator",
                "display": "Denominator"
              }
            ]
          },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Denominator"
          }
        },
        {
          "id": "denominator-exclusion",
          "code": {
            "coding": [
              {
                "system": "http://terminology.hl7.org/CodeSystem/measure-population",
                "code": "denominator-exclusion",
                "display": "Denominator Exclusion"
              }
            ]
          },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Denominator Exclusion"
          }
        },
        {
          "id": "numerator",
          "code": {
            "coding": [
              {
                "system": "http://terminology.hl7.org/CodeSystem/measure-population",
                "code": "numerator",
                "display": "Numerator"
              }
            ]
          },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Numerator"
          }
        }
      ],
      "stratifier": [
        {
          "id": "strat-age-18-44",
          "code": { "text": "Age 18-44" },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Stratifier Age 18-44"
          }
        },
        {
          "id": "strat-age-45-64",
          "code": { "text": "Age 45-64" },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Stratifier Age 45-64"
          }
        },
        {
          "id": "strat-age-65-75",
          "code": { "text": "Age 65-75" },
          "criteria": {
            "language": "text/cql-identifier",
            "expression": "Stratifier Age 65-75"
          }
        }
      ]
    }
  ],
  "supplementalData": [
    {
      "id": "sde-ethnicity",
      "usage": [
        {
          "coding": [
            {
              "system": "http://terminology.hl7.org/CodeSystem/measure-data-usage",
              "code": "supplemental-data"
            }
          ]
        }
      ],
      "criteria": {
        "language": "text/cql-identifier",
        "expression": "SDE Ethnicity"
      }
    },
    {
      "id": "sde-race",
      "usage": [
        {
          "coding": [
            {
              "system": "http://terminology.hl7.org/CodeSystem/measure-data-usage",
              "code": "supplemental-data"
            }
          ]
        }
      ],
      "criteria": {
        "language": "text/cql-identifier",
        "expression": "SDE Race"
      }
    },
    {
      "id": "sde-sex",
      "usage": [
        {
          "coding": [
            {
              "system": "http://terminology.hl7.org/CodeSystem/measure-data-usage",
              "code": "supplemental-data"
            }
          ]
        }
      ],
      "criteria": {
        "language": "text/cql-identifier",
        "expression": "SDE Sex"
      }
    },
    {
      "id": "sde-payer",
      "usage": [
        {
          "coding": [
            {
              "system": "http://terminology.hl7.org/CodeSystem/measure-data-usage",
              "code": "supplemental-data"
            }
          ]
        }
      ],
      "criteria": {
        "language": "text/cql-identifier",
        "expression": "SDE Payer"
      }
    }
  ]
}
```

---

## Library resource template (with embedded CQL and ELM)

```json
{
  "resourceType": "Library",
  "id": "CMS122",
  "meta": {
    "profile": [
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/computable-library-cqfm",
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/publishable-library-cqfm",
      "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/executable-library-cqfm"
    ]
  },
  "extension": [
    {
      "url": "http://hl7.org/fhir/us/cqfmeasures/StructureDefinition/cqfm-softwaresystems",
      "extension": [
        {
          "url": "software",
          "valueCoding": { "code": "translator" }
        },
        { "url": "version", "valueString": "3.0.0" }
      ]
    }
  ],
  "url": "https://madie.cms.gov/Library/CMS122",
  "version": "12.0.000",
  "name": "CMS122",
  "title": "Diabetes: Hemoglobin A1c (HbA1c) Poor Control (> 9%) - Library",
  "status": "active",
  "experimental": false,
  "type": {
    "coding": [
      {
        "system": "http://terminology.hl7.org/CodeSystem/library-type",
        "code": "logic-library"
      }
    ]
  },
  "date": "2024-06-01",
  "publisher": "CMS",
  "relatedArtifact": [
    {
      "type": "depends-on",
      "resource": "http://fhir.org/guides/cqf/common/Library/FHIRHelpers|4.0.1"
    },
    {
      "type": "depends-on",
      "display": "Diabetes",
      "resource": "http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020"
    }
  ],
  "parameter": [
    {
      "name": "Measurement Period",
      "use": "in",
      "min": 0,
      "max": "1",
      "type": "Period"
    },
    {
      "name": "Patient",
      "use": "out",
      "min": 0,
      "max": "1",
      "type": "Patient"
    }
  ],
  "dataRequirement": [
    {
      "type": "Encounter",
      "profile": ["http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-encounter"],
      "mustSupport": ["type", "status", "period"],
      "codeFilter": [
        {
          "path": "type",
          "valueSet": "http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1240"
        }
      ]
    },
    {
      "type": "Condition",
      "profile": [
        "http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-condition-problems-health-concerns"
      ],
      "mustSupport": ["code", "clinicalStatus", "verificationStatus", "onset"],
      "codeFilter": [
        {
          "path": "code",
          "valueSet": "http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020"
        }
      ]
    },
    {
      "type": "Observation",
      "profile": [
        "http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-observation-lab"
      ],
      "mustSupport": ["code", "status", "effective", "value"],
      "codeFilter": [
        {
          "path": "code",
          "valueSet": "http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.198.12.1013"
        }
      ]
    }
  ],
  "content": [
    {
      "contentType": "text/cql",
      "data": "<base64-encoded CQL source>"
    },
    {
      "contentType": "application/elm+xml",
      "data": "<base64-encoded ELM XML>"
    },
    {
      "contentType": "application/elm+json",
      "data": "<base64-encoded ELM JSON>"
    }
  ]
}
```

---

## Measure bundle assembly order

```json
{
  "resourceType": "Bundle",
  "type": "transaction",
  "entry": [
    { "resource": { "resourceType": "Measure", ... } },
    { "resource": { "resourceType": "Library", "id": "CMS122", ... } },
    { "resource": { "resourceType": "Library", "id": "CMS122-Terminology", ... } },
    { "resource": { "resourceType": "Library", "id": "ClinicalHelpers", ... } },
    { "resource": { "resourceType": "Library", "id": "FHIRHelpers", ... } },
    { "resource": { "resourceType": "Library", "id": "FHIRCommon", ... } },
    { "resource": { "resourceType": "ValueSet", ... } },
    { "resource": { "resourceType": "ValueSet", ... } }
  ]
}
```

---

## Artifact lifecycle states

| FHIR status | Meaning | Allowed operations |
|-------------|---------|-------------------|
| `draft` | Under development | Any changes |
| `active` | Published and in use | Breaking changes require new version |
| `retired` | No longer in use | Read-only; no changes |

Use `experimental: true` for test/development measures not intended for production.

---

## CMS eCQM submission requirements

For CMS program submissions (MIPS, Medicaid, EP/EC):

1. **Canonical URL format**: `https://madie.cms.gov/Measure/<CMS-ID>`
2. **Library version**: `<major>.<minor>.<patch>` — use `000` patch for initial release
3. **Both CQL and ELM** must be embedded in Library.content (base64)
4. **ELM produced with**: annotations + locators + `compatibilityLevel 1.5`
5. **Value set versions** must be pinned to the specific annual VSAC release
6. **Translator version** must be recorded in Library.extension (softwaresystems)
7. **DataRequirement** elements must be present for all retrieved resource types
8. **RelatedArtifact** must list all value sets as `depends-on`
9. **MADIE** (Measure Authoring Development Integrated Environment) is the CMS-approved tooling

---

## CRMI profiles used

| Profile | URL | Use |
|---------|-----|-----|
| Computable Measure | `cqfmeasures/computable-measure-cqfm` | Measure with full logic |
| Publishable Measure | `cqfmeasures/publishable-measure-cqfm` | Published to a server |
| Executable Measure | `cqfmeasures/executable-measure-cqfm` | Ready for evaluation |
| Computable Library | `cqfmeasures/computable-library-cqfm` | Logic library |
| Publishable Library | `cqfmeasures/publishable-library-cqfm` | Published library |
| Executable Library | `cqfmeasures/executable-library-cqfm` | Library with ELM |
| Test Case Bundle | `cqfmeasures/test-case-bundle-cqfm` | Test bundle |
| Test Case (MeasureReport) | `cqfmeasures/test-case-cqfm` | Expected results |

---

## NPM IG packaging

FHIR Implementation Guides are packaged as npm modules:

```
ig/
  package.json          # IG metadata (id, version, fhirVersions, dependencies)
  package/
    Measure-CMS122.json
    Library-CMS122.json
    Library-CMS122-Terminology.json
    ...
```

```json
// package.json
{
  "name": "cms.ecqm.cms122",
  "version": "12.0.0",
  "canonical": "https://madie.cms.gov",
  "title": "CMS122 Diabetes HbA1c Poor Control",
  "description": "CMS122 eCQM package for 2024 program year",
  "fhirVersions": ["4.0.1"],
  "dependencies": {
    "hl7.fhir.uv.cql": "2.0.0",
    "hl7.fhir.us.qicore": "7.0.2",
    "hl7.fhir.us.core": "8.0.1"
  }
}
```
