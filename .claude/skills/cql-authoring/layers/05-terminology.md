# Layer 05 — Terminology: Value Sets, Code Systems, and Operators

This layer covers all aspects of terminology in CQL: VSAC value set declarations,
code system canonical URLs, direct-reference codes, terminology operators, version
pinning, and UCUM units.

---

## Value set declaration

### From VSAC (OID-based canonical URL)
```cql
// VSAC canonical URL format: http://cts.nlm.nih.gov/fhir/ValueSet/<OID>
valueset "Diabetes": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020'
valueset "HbA1c Laboratory Test": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.198.12.1013'
valueset "Inpatient Encounter": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.666.5.307'
```

### From HL7 FHIR (HL7-maintained value sets)
```cql
// HL7 base URLs
valueset "ActCode": 'http://terminology.hl7.org/ValueSet/v3-ActCode'
valueset "Encounter Status": 'http://hl7.org/fhir/ValueSet/encounter-status'
```

### From project-local or IG value sets
```cql
// Project-defined (not in VSAC)
valueset "Elective Inpatient Procedure": 'https://example.org/fhir/ValueSet/elective-inpatient'
```

### Version-pinned value set
Only pin versions when the submission program requires it. Avoid otherwise — unpinned
value sets receive updates transparently:

```cql
// Pinned (use only when required by eCQM submission rules)
valueset "Diabetes" version '20240101':
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020'
```

---

## Code system declarations

### Standard clinical code systems (canonical URLs)

```cql
// Primary code systems
codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "LOINC": 'http://loinc.org'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "CPT": 'http://www.ama-assn.org/go/cpt'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "ICD10PCS": 'http://www.cms.gov/Medicare/Coding/ICD10'
codesystem "ICD9CM": 'http://hl7.org/fhir/sid/icd-9-cm'
codesystem "HCPCS": 'https://www.cms.gov/Medicare/Coding/HCPCSReleaseCodeSets'
codesystem "NDC": 'http://hl7.org/fhir/sid/ndc'
codesystem "CVX": 'http://hl7.org/fhir/sid/cvx'

// HL7 terminology code systems
codesystem "ActCode": 'http://terminology.hl7.org/CodeSystem/v3-ActCode'
codesystem "AdministrativeGender": 'http://hl7.org/fhir/administrative-gender'
codesystem "ConditionClinicalStatus": 'http://terminology.hl7.org/CodeSystem/condition-clinical'
codesystem "ConditionVerificationStatus": 'http://terminology.hl7.org/CodeSystem/condition-ver-status'
codesystem "ObservationStatus": 'http://hl7.org/fhir/observation-status'
codesystem "NullFlavors": 'http://terminology.hl7.org/CodeSystem/v3-NullFlavor'
codesystem "RoleCode": 'http://terminology.hl7.org/CodeSystem/v3-RoleCode'

// US-specific
codesystem "UB04RevenueCodes": 'https://www.nubc.org/CodeSystem/RevenueCodes'
codesystem "POSCode": 'https://www.cms.gov/Medicare/Coding/place-of-service-codes/Place_of_Service_Code_Set'
codesystem "OmbRaceCategories": 'urn:oid:2.16.840.1.113883.6.238'
```

---

## Direct-reference codes (DRC)

Use DRCs for specific, non-value-set-member codes — especially LOINC vital sign codes
and SNOMED clinical findings:

```cql
// Vital sign LOINC codes
code "Systolic blood pressure": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic blood pressure": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "Body mass index": '39156-5' from "LOINC" display 'Body mass index'
code "Birth date": '21112-8' from "LOINC" display 'Birth date'
code "Tobacco use status": '72166-2' from "LOINC" display 'Tobacco smoking status'

// SNOMED clinical concepts
code "Left": '7771000' from "SNOMEDCT" display 'Left'
code "Right": '24028007' from "SNOMEDCT" display 'Right'
code "Bilateral": '51440002' from "SNOMEDCT" display 'Bilateral'

// Discharge dispositions
code "Discharged to Home": '01' from "UB04RevenueCodes" display 'Discharged to Home'
code "Discharged to SNF": '03' from "UB04RevenueCodes" display 'Skilled Nursing Facility'
```

---

## Terminology operators

### `in` — value set membership
```cql
// Condition code in value set (prefer this — pushes filter to server retrieve)
[Condition: "Diabetes"]

// Post-retrieve membership check (when conditional logic is needed)
exists (
  [Condition] C
    where C.code in "Diabetes"
      and C.clinicalStatus ~ FHIRCommon."active"
)
```

### `~` (equivalent) — direct-reference code comparison
```cql
// Compare a FHIR.Coding or FHIR.CodeableConcept to a CQL Code
// Does NOT check system version or display; compares system + code only
Observation.code ~ "Systolic blood pressure"

// Correct pattern for BP components
singleton from (
  bp.component C
    where C.code ~ "Systolic blood pressure"
).value as FHIR.Quantity

// WRONG: = (equals) requires identical objects; use ~ for code comparison
Observation.code = "Systolic blood pressure"   // Will not work as intended
```

### `AnyInValueSet` — list membership check
```cql
// True if any coding in a codeable concept is in a value set
AnyInValueSet(Condition.code, "Diabetes")

// Equivalent but more idiomatic when inside a query:
[Condition] C where C.code in "Diabetes"
```

### `InValueSet` — single code membership
```cql
// True if a specific code is in a value set
InValueSet(
  Code { system: 'http://snomed.info/sct', code: '44054006' },
  "Diabetes"
)
```

---

## Negation terminology patterns

### NotDoneValueSet (QI-Core negation reason)
```cql
// "Not done" with reason documented via value set reference
define "Colonoscopy Not Done Due to Refusal":
  [Procedure: "Colonoscopy"] P
    where P.status = 'not-done'
      and FHIRCommon.GetExtension(P,
            'http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-notDoneValueSet'
          ) is not null
      and (
        FHIRCommon.GetExtension(P,
          'http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-notDoneValueSet'
        ).value as FHIR.canonical
      ) = 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.600.1.1503'

// Simpler: check statusReason code
define "Colonoscopy Not Done with Medical Reason":
  [Procedure: "Colonoscopy"] P
    where P.status = 'not-done'
      and P.statusReason in "Medical Reason"
```

### Not Ordered (MedicationRequest / ServiceRequest negation)
```cql
define "Statin Not Ordered Due to Allergy":
  [MedicationRequest: "Statin Medications"] MR
    where MR.status = 'cancelled'
      and MR.statusReason in "Allergy to Statin"

define "Referral Not Ordered":
  [ServiceRequest: "Cardiology Referral"] SR
    where SR.status = 'revoked'
      and SR.reasonCode in "Medical Reason"
```

---

## UCUM units

CQL Quantity comparisons require compatible UCUM units. Use the canonical UCUM form:

| Clinical measurement | UCUM unit | CQL Quantity literal |
|--------------------|-----------|---------------------|
| Blood pressure (mmHg) | `mm[Hg]` | `140 'mm[Hg]'` |
| HbA1c (%) | `%` | `9.0 '%'` |
| BMI (kg/m²) | `kg/m2` | `30 'kg/m2'` |
| Weight (kg) | `kg` | `70 'kg'` |
| Weight (lb) | `[lb_av]` | `154 '[lb_av]'` |
| Height (cm) | `cm` | `180 'cm'` |
| Height (in) | `[in_i]` | `71 '[in_i]'` |
| LDL (mg/dL) | `mg/dL` | `100 'mg/dL'` |
| Creatinine (mg/dL) | `mg/dL` | `1.5 'mg/dL'` |
| eGFR (mL/min/1.73m²) | `mL/min/{1.73_m2}` | `60 'mL/min/{1.73_m2}'` |
| Age (years) | `a` | `18 'a'` (use AgeInYearsAt for patient age) |
| Duration (days) | `d` | `90 'd'` |
| Duration (hours) | `h` | `4 'h'` |
| Duration (minutes) | `min` | `30 'min'` |
| Frequency | `/d` | `2 '/d'` (twice daily) |
| Pack years | `{PackYears}` | `10 '{PackYears}'` |

### Unit comparison rules
```cql
// VALID: same units
(Observation.value as FHIR.Quantity) > 9.0 '%'

// VALID: compatible units (CQL will convert)
(Observation.value as FHIR.Quantity) > 140 'mm[Hg]'

// INVALID: incompatible units — runtime error
(Observation.value as FHIR.Quantity) > 140 'mg/dL'   // if obs is in mm[Hg]
```

---

## Terminology manifest (CQL Terminology Service pattern)

For programmatic terminology access, use a manifest to pin value sets:

```json
{
  "resourceType": "Library",
  "type": { "coding": [{ "code": "asset-collection" }] },
  "useContext": [
    {
      "code": { "code": "program" },
      "valueCodeableConcept": {
        "coding": [{ "code": "ep-ec", "display": "EP/EC Quality Measures" }]
      }
    }
  ],
  "relatedArtifact": [
    {
      "type": "depends-on",
      "resource": "http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020|20240101"
    }
  ]
}
```

---

## Common terminology mistakes to avoid

| Mistake | Correct approach |
|---------|-----------------|
| Using `=` for code comparison | Use `~` for FHIR.Coding / FHIR.CodeableConcept |
| Using `in` for single code comparison | Use `~` for DRC comparison; `in` for value set membership |
| Pinning value set versions unnecessarily | Omit version; let the terminology server resolve latest |
| Including code system in value set name | Value set name should be clinical, not technical |
| Forgetting UCUM units on Quantity literals | Always include units: `9.0 '%'` not `9.0` |
| Comparing Quantity across incompatible units | Ensure consistent units or use unit conversion |
| Using OID format in canonical URL | Always use full VSAC canonical URL `http://cts.nlm.nih.gov/...` |
| Mixing ICD-10-CM and ICD-10-PCS in one value set | Keep diagnosis (CM) and procedure (PCS) sets separate |

---

## Value set governance rules (enterprise)

1. **Own nothing in VSAC** unless you are the designated eCQM steward.
2. **Reference VSAC by canonical URL only** — never hardcode OIDs as string literals in CQL logic.
3. **Value set change requests** must go through VSAC authoring or your internal value set management system.
4. **Effective date alignment**: when CMS requires a specific annual version, use version-pinned declarations in your terminology library and document the reason.
5. **Custom value sets** (not in VSAC): publish to your organization's terminology server with a FHIR ValueSet resource; use your org's canonical URL.
6. **Local code systems** (custom): define a FHIR CodeSystem resource with your org's canonical URL; register with your terminology server.
