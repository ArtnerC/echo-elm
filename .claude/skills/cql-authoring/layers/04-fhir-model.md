# Layer 04 — FHIR Model, QI-Core Profiles, and Data Retrieval

This layer covers correct use of FHIR R4 and QI-Core 7.0.2 in CQL, including resource
retrieval syntax, profile filtering, choice type handling, extension access, and common
clinical resource patterns.

---

## Model declaration

Always declare the FHIR version and include FHIRHelpers as the first include:

```cql
using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers
include hl7.fhir.uv.cql.FHIRCommon version '4.0.1' called FHIRCommon
```

FHIRHelpers provides implicit conversion functions (Coding→Code, Period→Interval, etc.)
FHIRCommon provides `GetExtension`, `resolve()`, and common code comparisons.

---

## Resource retrieval syntax

### Basic retrieve
```cql
// All encounters (no profile filter)
[Encounter]

// Filtered by QI-Core profile (preferred for eCQMs)
[Encounter: "Office Visit"]         // value set filter on type
[Encounter: class = "AMB"]          // code filter on a specific field
```

### Retrieve with QI-Core profile binding
QI-Core profiles use FHIR profile URLs. CQL retrieves by value set or code:

```cql
// QI-Core: QICore-Encounter (inpatient)
[Encounter: "Inpatient Encounter"]

// QI-Core: QICore-Condition (active problem list)
[Condition: "Diabetes"]

// QI-Core: QICore-Observation (lab result)
[Observation: "HbA1c Laboratory Test"]

// QI-Core: QICore-MedicationRequest
[MedicationRequest: "ACE Inhibitor or ARB or ARNI"]

// QI-Core: QICore-Procedure
[Procedure: "Colonoscopy"]

// QI-Core: QICore-DiagnosticReport-lab
[DiagnosticReport: "Mammography"]

// QI-Core: QICore-Coverage
[Coverage: type in "Payer Type"]
```

### Retrieve with multiple conditions
```cql
[Encounter: "Office Visit"] E
  where E.status = 'finished'
    and E.period during "Measurement Period"
    and E.class ~ "AMB"
```

---

## Common resource patterns

### Encounter

```cql
define "Qualifying Encounters":
  [Encounter: "Outpatient Encounter"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"

// Telehealth encounter (check for virtual component)
define "Telehealth Encounters":
  [Encounter: "Telehealth Services"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"

// Inpatient LOS calculation
define "Encounter Length of Stay"(encounter FHIR.Encounter):
  duration in days of encounter.period
```

### Condition

```cql
// Active, confirmed diagnosis
define "Active Diabetes":
  [Condition: "Diabetes"] C
    where C.clinicalStatus ~ FHIRCommon."active"
      and C.verificationStatus ~ FHIRCommon."confirmed"

// Onset before measurement period (prevalent condition)
define "Pre-existing Hypertension":
  [Condition: "Essential Hypertension"] C
    where C.clinicalStatus ~ FHIRCommon."active"
      and C.onset.toInterval() before end of "Measurement Period"

// Historical (resolved) condition
define "History of Myocardial Infarction":
  [Condition: "Myocardial Infarction"] C
    where C.clinicalStatus in { FHIRCommon."inactive", FHIRCommon."resolved" }
      or C.verificationStatus ~ FHIRCommon."entered-in-error" is false
```

### Observation

```cql
// Most recent qualifying lab result
define "Most Recent HbA1c":
  Last(
    [Observation: "HbA1c Laboratory Test"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and O.effective.toInterval() during "Measurement Period"
      sort by O.effective.effectiveDateTime() ascending
  )

// Vital sign (blood pressure — multi-component)
define "Blood Pressure Observations":
  [Observation: "Blood Pressure"] BP
    where BP.status in { 'final', 'amended' }
      and BP.effective.toInterval() during "Measurement Period"

define fluent function systolic(bp FHIR.Observation):
  singleton from (
    bp.component C
      where C.code ~ "Systolic blood pressure"
  ).value as FHIR.Quantity

define fluent function diastolic(bp FHIR.Observation):
  singleton from (
    bp.component C
      where C.code ~ "Diastolic blood pressure"
  ).value as FHIR.Quantity

// Most recent BP with adequate control
define "Blood Pressure Controlled":
  exists (
    "Blood Pressure Observations" BP
      where BP.systolic() < 140 'mm[Hg]'
        and BP.diastolic() < 90 'mm[Hg]'
  )
```

### MedicationRequest

```cql
// Active prescription ordered during measurement period
define "Statin Therapy Ordered":
  [MedicationRequest: "Statin Medications"] MR
    where MR.status in { 'active', 'completed' }
      and MR.intent = 'order'
      and MR.authoredOn during "Measurement Period"

// Long-term medication (no end date or end date after period)
define "Long-term Anticoagulation":
  [MedicationRequest: "Anticoagulant Medications"] MR
    where MR.status = 'active'
      and MR.intent = 'order'
      and Coalesce(
            MR.dispenseRequest.validityPeriod.end.value,
            end of "Measurement Period"
          ) >= end of "Measurement Period"
```

### MedicationDispense

```cql
// Filled prescription (pharmacy claim proxy)
define "Dispensed Antihypertensives":
  [MedicationDispense: "Antihypertensive Medications"] MD
    where MD.status = 'completed'
      and MD.whenHandedOver during "Measurement Period"
```

### Procedure

```cql
// Completed procedure during measurement period
define "Colonoscopy Performed":
  [Procedure: "Colonoscopy"] P
    where P.status = 'completed'
      and P.performed.toInterval() during "Measurement Period"

// Procedure during relevant 10-year window
define "Colonoscopy in Last 10 Years":
  [Procedure: "Colonoscopy"] P
    where P.status = 'completed'
      and P.performed.toInterval() starts after
          (start of "Measurement Period" - 10 years)
```

### DiagnosticReport

```cql
// Radiology/imaging report
define "Mammography Performed":
  [DiagnosticReport: "Mammography"] D
    where D.status in { 'final', 'amended', 'corrected' }
      and D.effective.toInterval() starts after
          (start of "Measurement Period" - 27 months)
```

### ServiceRequest

```cql
// Order placed during measurement period
define "Palliative Care Order":
  [ServiceRequest: "Palliative Care Intervention"] SR
    where SR.status in { 'active', 'completed' }
      and SR.intent = 'order'
      and SR.occurrence.toInterval() overlaps "Measurement Period"
```

### Coverage

```cql
// Patient had coverage during the full measurement period
define "Continuously Enrolled":
  exists (
    [Coverage: type in "Payer Type"] C
      where C.status = 'active'
        and C.period includes "Measurement Period"
  )
```

### AllergyIntolerance

```cql
define "Documented Drug Allergy":
  [AllergyIntolerance: "Drug Allergy"] A
    where A.clinicalStatus ~ FHIRCommon."active"
      and A.verificationStatus ~ FHIRCommon."confirmed"
      and A.type = 'allergy'
```

---

## Choice type handling

FHIR resources frequently use `[x]` polymorphic types. Access them using `as`:

```cql
// Condition.onset[x] — dateTime, Age, Period, Range, string
define fluent function toInterval(
  choice Choice<FHIR.dateTime, FHIR.Age, FHIR.Period, FHIR.Range, FHIR.string>
):
  case
    when choice is FHIR.dateTime then
      Interval[FHIRHelpers.ToDateTime(choice as FHIR.dateTime),
               FHIRHelpers.ToDateTime(choice as FHIR.dateTime)]
    when choice is FHIR.Age then
      Interval[Patient.birthDate + (choice as FHIR.Age),
               Patient.birthDate + (choice as FHIR.Age) + 1 year]
    when choice is FHIR.Period then
      FHIRHelpers.ToInterval(choice as FHIR.Period)
    else null as Interval<DateTime>
  end

// Observation.value[x] — Quantity, CodeableConcept, string, boolean, integer, Range, Ratio
define fluent function asQuantity(obs FHIR.Observation):
  obs.value as FHIR.Quantity

define fluent function asCode(obs FHIR.Observation):
  obs.value as FHIR.CodeableConcept

// MedicationRequest.medication[x] — CodeableConcept or Reference
// Using CodeableConcept (preferred for value set membership):
define "Statin Orders by Code":
  [MedicationRequest] MR
    where MR.medication is FHIR.CodeableConcept
      and (MR.medication as FHIR.CodeableConcept) in "Statin Medications"

// Using Reference (resolve to Medication resource):
define "Statin Orders by Reference":
  [MedicationRequest] MR
    where MR.medication is FHIR.Reference
      and exists (
        [Medication: "Statin Medications"] Med
          where Med.id = (MR.medication as FHIR.Reference).id
      )
```

---

## Extension access

### Standard extension access via FHIRCommon
```cql
// Single-valued extension
define fluent function GetExtension(domainResource FHIR.DomainResource, url String):
  FHIRCommon.GetExtension(domainResource, url)

// US Core Race extension
define "Patient Race Categories":
  (FHIRCommon.GetExtension(Patient,
    'http://hl7.org/fhir/us/core/StructureDefinition/us-core-race')) race
  return (
    race.extension E
      where E.url = 'ombCategory'
      return E.value as FHIR.Coding
  )

// QI-Core notDone extension
define "Colonoscopy Not Done":
  [Procedure: "Colonoscopy"] P
    where P.status = 'not-done'
      and (
        FHIRCommon.GetExtension(P,
          'http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-notDoneValueSet')
        is not null
      )
```

### Nested extension access
```cql
// FHIR.Extension is a backbone; navigate .extension list for nested extensions
define fluent function getExtensionValue(
  ext FHIR.Extension, subUrl String
):
  singleton from (
    ext.extension E
      where E.url = subUrl
  ).value
```

---

## Relationship traversal (with / without)

```cql
// Encounters WITH a specific condition during the encounter
define "Encounters with Active Diabetes":
  [Encounter: "Office Visit"] E
    with [Condition: "Diabetes"] C
      such that C.clinicalStatus ~ FHIRCommon."active"
        and C.encounter.references(E)

// Encounters WITHOUT any palliative care order
define "Encounters Without Palliative Care":
  [Encounter: "Office Visit"] E
    without [ServiceRequest: "Palliative Care"] SR
      such that SR.intent = 'order'
        and SR.occurrence.toInterval() overlaps E.period

// Linking via references()
define "MedicationRequests by Ordering Encounter":
  [MedicationRequest: "Statin Medications"] MR
    with [Encounter: "Office Visit"] E
      such that MR.encounter.references(E)
        and E.period during "Measurement Period"
```

---

## FHIR primitive `.value` access

FHIR primitives in CQL are not native CQL types — they are FHIR wrapper types.
FHIRHelpers provides implicit conversion for most uses, but explicit `.value` access
is sometimes required:

```cql
// FHIR.string → CQL String
Condition.code.coding[0].display.value

// FHIR.boolean → CQL Boolean
Condition.active.value

// FHIR.integer → CQL Integer
Observation.valueInteger.value

// FHIR.dateTime — use FHIRHelpers for safe conversion
FHIRHelpers.ToDateTime(Encounter.period.start)

// FHIR.decimal — usually implicit via FHIRHelpers
(Observation.value as FHIR.Quantity).value.value
```

---

## Status code handling

Always filter by status — unsuppressed `entered-in-error` resources corrupt results:

| Resource | Valid statuses for measurement |
|---------|-------------------------------|
| Encounter | `finished` (or `in-progress` for current-care logic) |
| Condition | `active`, `recurrence`, `relapse` (not `inactive`, `remission`, `resolved`) |
| Observation | `final`, `amended`, `corrected` (not `preliminary`, `registered`) |
| MedicationRequest | `active`, `completed` (not `cancelled`, `stopped`, `entered-in-error`) |
| Procedure | `completed` (not `aborted`, `not-done` unless negation logic) |
| DiagnosticReport | `final`, `amended`, `corrected`, `appended` |
| ServiceRequest | `active`, `completed` (not `revoked`) |

---

## QI-Core 7.0.2 profile-specific patterns

### QI-Core Negation Rationale

```cql
// Not performed (with reason — QI-Core notDone pattern)
define "Colonoscopy Not Done Due to Patient Refusal":
  [Procedure: "Colonoscopy"] P
    where P.status = 'not-done'
      and singleton from (
        P.statusReason.coding C
          where C in "Patient Refusal"
      ) is not null
```

### QI-Core DeviceRequest

```cql
define "Cardiac Device Ordered":
  [DeviceRequest: "Implantable Defibrillator Device Order"] D
    where D.status in { 'active', 'completed' }
      and D.intent = 'order'
      and D.authored.toInterval() during "Measurement Period"
```

### Profile URL references
| QI-Core Profile | FHIR Resource |
|----------------|---------------|
| `QICore-Encounter` | Encounter |
| `QICore-Condition-encounter-diagnosis` | Condition (encounter dx) |
| `QICore-Condition-problems-health-concerns` | Condition (problem list) |
| `QICore-Observation-lab` | Observation (lab) |
| `QICore-ObservationClinicalResult` | Observation (clinical) |
| `QICore-MedicationRequest` | MedicationRequest |
| `QICore-MedicationDispense` | MedicationDispense |
| `QICore-Procedure` | Procedure |
| `QICore-DiagnosticReport-lab` | DiagnosticReport (lab) |
| `QICore-DiagnosticReport-note` | DiagnosticReport (report) |
| `QICore-ServiceRequest` | ServiceRequest |
| `QICore-Coverage` | Coverage |
| `QICore-Patient` | Patient |

---

## Common performance anti-patterns

```cql
// SLOW: Function call inside a large query's where clause
// (FHIRHelpers.ToDateTime is called per-row)
[Observation] O where FHIRHelpers.ToDateTime(O.effective) during "Measurement Period"

// FAST: Use fluent function memoized at retrieve level
[Observation: "HbA1c Lab"] O
  where O.effective.toInterval() during "Measurement Period"

// WRONG: retrieve ALL observations then filter by value set in CQL
[Observation] O where O.code in "HbA1c Laboratory Test"

// RIGHT: push the value set into the retrieve (enables server-side filtering)
[Observation: "HbA1c Laboratory Test"] O
```
