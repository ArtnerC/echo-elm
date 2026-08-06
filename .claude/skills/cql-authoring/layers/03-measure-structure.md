# Layer 03 — Measure Structure, Population Criteria, and Component Annotations

This layer defines how to structure CQL for quality measures, covering all measure types,
population expressions, and the annotation format that links CQL expressions to FHIR Measure
population criteria.

---

## Measure types and their required populations

### Proportion measure
Most common measure type. Computes a rate: `Numerator / Denominator`.

| Population | Code | Required | Description |
|-----------|------|----------|-------------|
| Initial Population | `initial-population` | **Yes** | Basic eligibility (age, encounter, enrollment) |
| Denominator | `denominator` | **Yes** | Subset of IPP meeting denominator criteria |
| Denominator Exclusion | `denominator-exclusion` | No | Removed from denominator before scoring |
| Denominator Exception | `denominator-exception` | No | Removed from denominator; numerator failures only |
| Numerator | `numerator` | **Yes** | Subset of denominator meeting quality criteria |
| Numerator Exclusion | `numerator-exclusion` | No | Removed from numerator before scoring |

### Ratio measure
Two-rate measure. Both numerator and denominator have independent initial populations.

| Population | Code | Required |
|-----------|------|----------|
| Initial Population (IPP 1) | `initial-population` | **Yes** |
| Denominator | `denominator` | **Yes** |
| Denominator Exclusion | `denominator-exclusion` | No |
| Initial Population (IPP 2) | `initial-population` | **Yes** |
| Numerator | `numerator` | **Yes** |
| Numerator Exclusion | `numerator-exclusion` | No |

### Continuous variable measure
Computes an aggregate (mean, median, sum) over a measure observation function.

| Population | Code | Required | Description |
|-----------|------|----------|-------------|
| Initial Population | `initial-population` | **Yes** | Eligible population |
| Measure Population | `measure-population` | **Yes** | Subset of IPP eligible for observation |
| Measure Population Exclusion | `measure-population-exclusion` | No | Removed from population |
| Measure Observation | `measure-observation` | **Yes** | **Function** returning the observed value |

### Cohort measure
Binary: member / non-member. No score.

| Population | Code | Required |
|-----------|------|----------|
| Initial Population | `initial-population` | **Yes** |

### Composite measure
Aggregates component measures. Scoring types: individual, linear combination, opportunity score.
Each component is a reference to a component measure library.

---

## Standard component annotation format

Use this structured annotation header on **every** expression that is a measure component.
Tooling (including echo-elm) scans these annotations to:
- Populate the FHIR Measure resource automatically
- Validate that Measure.group.population criteria names match CQL expression names
- Generate documentation

```
/*
 * @measure-component: <population-code>
 * @group: <group-id>                    (omit for single-group measures)
 * @description: <clinical intent>       (required)
 * @improvement-notation: increase | decrease
 * @basis: Patient | Encounter | Episode  (only needed when non-default)
 */
```

### Population code values (from measure-population code system)

| Population | `@measure-component` value |
|-----------|---------------------------|
| Initial Population | `initial-population` |
| Denominator | `denominator` |
| Denominator Exclusion | `denominator-exclusion` |
| Denominator Exception | `denominator-exception` |
| Numerator | `numerator` |
| Numerator Exclusion | `numerator-exclusion` |
| Measure Population | `measure-population` |
| Measure Population Exclusion | `measure-population-exclusion` |
| Measure Observation | `measure-observation` |
| Stratifier | `stratifier` |
| Supplemental Data Element | `supplemental-data` |
| Risk Adjustment Variable | `risk-adjustment` |

---

## Complete proportion measure example (Diabetes HbA1c Control)

```cql
/**
 * Measure: Diabetes: Hemoglobin A1c (HbA1c) Poor Control (> 9%)
 * CMS ID: CMS122
 * NQF: 0059
 * Steward: NCQA
 * Measurement Year: 2024
 * Improvement Notation: Decrease (lower rate is better)
 *
 * This measure calculates the percentage of patients 18–75 years of age
 * with diabetes who had hemoglobin A1c > 9.0% during the measurement period.
 */
library CMS.CMS122 version '12.0.000'

using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers
include hl7.fhir.uv.cql.FHIRCommon version '4.0.1' called FHIRCommon
include CMS.CMS122-Terminology version '1.0.0' called Terminology
include CMS.ClinicalHelpers version '2.0.0' called Helpers
include CMS.PatientCharacteristics version '1.0.0' called PC

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

context Patient

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  INITIAL POPULATION                                                      ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: initial-population
 * @description: Patients 18–75 years with at least one qualifying outpatient
 *   encounter AND an active diabetes diagnosis during the measurement period.
 *   Diabetes diagnosis must precede the measurement period end.
 */
define "Initial Population":
  "Patient Age 18 to 75 at Start"
    and exists "Qualifying Encounters"
    and "Has Active Diabetes Diagnosis"

define "Patient Age 18 to 75 at Start":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[18, 75]

define "Qualifying Encounters":
  [Encounter: Terminology."Office Visit"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"

define "Has Active Diabetes Diagnosis":
  exists (
    [Condition: Terminology."Diabetes"] C
      where C.clinicalStatus ~ FHIRCommon."active"
        and C.verificationStatus ~ FHIRCommon."confirmed"
        and C.onset.toInterval() before end of "Measurement Period"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  DENOMINATOR                                                             ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: denominator
 * @description: All patients in the Initial Population. The denominator
 *   equals the IPP for this measure.
 */
define "Denominator":
  "Initial Population"

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  DENOMINATOR EXCLUSION                                                   ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: denominator-exclusion
 * @description: Patients enrolled in hospice or receiving palliative care
 *   during the measurement period are excluded. Patients with frailty AND
 *   dementia are also excluded per CMS logic.
 */
define "Denominator Exclusion":
  "Has Hospice During Measurement Period"
    or "Has Palliative Care Order or Encounter"
    or ( "Is Frail" and "Has Dementia" )

define "Has Hospice During Measurement Period":
  exists (
    [Encounter: Terminology."Hospice Care Ambulatory"] E
      where E.status in { 'finished', 'in-progress' }
        and E.period overlaps "Measurement Period"
  )
  or exists (
    [ServiceRequest: Terminology."Hospice Care Ambulatory"] SR
      where SR.status in { 'active', 'completed' }
        and SR.intent = 'order'
        and SR.occurrence.toInterval() overlaps "Measurement Period"
  )

define "Has Palliative Care Order or Encounter":
  exists (
    [Encounter: Terminology."Palliative Care Encounter"] PC
      where PC.status = 'finished'
        and PC.period overlaps "Measurement Period"
  )
  or exists (
    [ServiceRequest: Terminology."Palliative Care Intervention"] PC
      where PC.status in { 'active', 'completed' }
        and PC.occurrence.toInterval() overlaps "Measurement Period"
  )

define "Is Frail":
  exists (
    [Observation: Terminology."Frailty Indicator"] F
      where F.status in { 'final', 'amended' }
        and F.effective.toInterval() during "Measurement Period"
  )

define "Has Dementia":
  exists (
    [Condition: Terminology."Dementia and Mental Degenerations"] D
      where D.clinicalStatus ~ FHIRCommon."active"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  NUMERATOR                                                               ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: numerator
 * @description: Patients in the Denominator whose most recent HbA1c result
 *   during the measurement period was > 9.0% (poor glycemic control).
 *   Improvement notation is DECREASE — this is an inverted measure.
 * @improvement-notation: decrease
 */
define "Numerator":
  "Most Recent HbA1c Result" > 9.0 '%'

define "Most Recent HbA1c Result":
  Last(
    [Observation: Terminology."HbA1c Laboratory Test"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and O.effective.toInterval() during "Measurement Period"
      sort by O.effective.effectiveDateTime()
  ).value as FHIR.Quantity

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  STRATIFIERS                                                             ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: stratifier
 * @strat-name: Stratifier Age 18-44
 * @description: Age group 18–44 years at start of measurement period.
 */
define "Stratifier Age 18-44":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[18, 44]

/*
 * @measure-component: stratifier
 * @strat-name: Stratifier Age 45-64
 */
define "Stratifier Age 45-64":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[45, 64]

/*
 * @measure-component: stratifier
 * @strat-name: Stratifier Age 65-75
 */
define "Stratifier Age 65-75":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[65, 75]

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  SUPPLEMENTAL DATA ELEMENTS                                              ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Ethnicity
 */
define "SDE Ethnicity":
  PC."SDE Ethnicity"

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Race
 */
define "SDE Race":
  PC."SDE Race"

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Sex
 */
define "SDE Sex":
  PC."SDE Sex"

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Payer
 */
define "SDE Payer":
  PC."SDE Payer"
```

---

## Continuous variable measure example (Length of Stay)

```cql
library CMS.LengthOfStay version '1.0.0'

using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

context Patient

/*
 * @measure-component: initial-population
 * @description: Patients with a qualifying inpatient encounter during the
 *   measurement period.
 */
define "Initial Population":
  "Qualifying Inpatient Encounters"

define "Qualifying Inpatient Encounters":
  [Encounter: "Inpatient Encounter"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"

/*
 * @measure-component: measure-population
 * @description: All encounters in the Initial Population eligible for LOS
 *   calculation. Excludes encounters where patient was transferred in.
 */
define "Measure Population":
  "Qualifying Inpatient Encounters" E
    without [Encounter] Transfer
      such that Transfer.hospitalization.admitSource ~ "Transferred from Hospital"

/*
 * @measure-component: measure-population-exclusion
 * @description: Encounters where patient was discharged to another acute
 *   facility are excluded from LOS scoring.
 */
define "Measure Population Exclusion":
  "Qualifying Inpatient Encounters" E
    where E.hospitalization.dischargeDisposition ~ "Discharged to Acute Facility"

/*
 * @measure-component: measure-observation
 * @basis: Encounter
 * @aggregate: median
 * @description: Calculates length of stay in days for each qualifying encounter.
 *   Result is used to compute median LOS across the measure population.
 */
define function "Measure Observation"(encounter FHIR.Encounter):
  duration in days of encounter.period
```

---

## Multi-group measure (two populations in one Measure resource)

Some measures have separate scoring groups (e.g., one for proportion, one for CV):

```cql
/*
 * @measure-component: initial-population
 * @group: group-1
 * @description: IPP for Group 1 (proportion scoring)
 */
define "Initial Population 1":
  ...

/*
 * @measure-component: denominator
 * @group: group-1
 */
define "Denominator 1":
  "Initial Population 1"

/*
 * @measure-component: numerator
 * @group: group-1
 */
define "Numerator 1":
  ...

/*
 * @measure-component: initial-population
 * @group: group-2
 * @description: IPP for Group 2 (continuous variable scoring)
 */
define "Initial Population 2":
  ...

/*
 * @measure-component: measure-population
 * @group: group-2
 */
define "Measure Population 2":
  "Initial Population 2"

/*
 * @measure-component: measure-observation
 * @group: group-2
 * @aggregate: mean
 */
define function "Measure Observation 2"(encounter FHIR.Encounter):
  ...
```

---

## Population logic dependency rules

Enforce these as part of every code review:

```
Denominator SUBSET-OF Initial Population
Numerator SUBSET-OF Denominator
Denominator Exclusion SUBSET-OF Denominator (before exclusion)
Numerator Exclusion SUBSET-OF Numerator (before exclusion)
Denominator Exception NOT Subset-Of Numerator (patients who fail the quality action)

Measure Population SUBSET-OF Initial Population (CV)
Measure Population Exclusion SUBSET-OF Measure Population (CV)
Measure Observation is a FUNCTION, not an expression (CV)
```

### Implementation pattern for Denominator dependency
```cql
// CORRECT: Denominator calls IPP expressions directly
define "Denominator":
  "Initial Population"   // or a subset filter applied to IPP conditions

// WRONG: Denominator re-implements IPP logic independently
define "Denominator":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[18, 75]
    and exists "Qualifying Encounters"
    // ^ This is IPP logic restated — creates drift risk
```

---

## Encounter-based vs patient-based measures

### Patient-based (default)
- `context Patient`
- Returns `true`/`false` for each patient
- Population expressions return Boolean

### Encounter-based
- `context Patient`
- Populations return **lists of encounters**
- Each encounter is independently scored
- Use `"Qualifying Encounters"` as the anchor

```cql
/*
 * @measure-component: initial-population
 * @basis: Encounter
 * @description: Each qualifying emergency department encounter for patients ≥ 18.
 */
define "Initial Population":
  "Qualifying Emergency Encounters"

define "Qualifying Emergency Encounters":
  [Encounter: "Emergency Department Visit"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"
      and AgeInYearsAt(date from start of E.period) >= 18

/*
 * @measure-component: numerator
 * @basis: Encounter
 * @description: ED encounters admitted within 4 hours of arrival.
 */
define "Numerator":
  "Qualifying Emergency Encounters" E
    where duration in hours of
          Interval[start of E.period, E.period.end] <= 4
```

---

## Composite measure structure

```cql
---

## Self-contained single-file measure (no shared library dependencies)

When a shared helper library ecosystem (FHIRCommon, ClinicalHelpers, PatientCharacteristics)
is not available, write a fully self-contained measure in a single CQL file.
Inline all helpers and status codes that would otherwise come from shared libraries.

### When to use this pattern
- New domain / clinical specialty without an existing shared library
- Standalone test fixtures or parity corpus entries
- Initial prototyping before extracting helpers to shared libraries
- Translator/engine parity tests where dependency resolution must be self-contained

### Self-contained measure template

```cql
/**
 * Measure: <Measure Title>
 * Library: <LibraryName>
 * Version: 1.0.0
 * Type: Proportion
 * Improvement Notation: increase | decrease
 * Description: <One-sentence clinical purpose>
 */
library <LibraryName> version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

// ── Code Systems ─────────────────────────────────────────────────────────────

codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "LOINC": 'http://loinc.org'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "CPT": 'http://www.ama-assn.org/go/cpt'

// Inline condition status codes (substitutes for FHIRCommon imports)
codesystem "ConditionClinicalStatus":
  'http://terminology.hl7.org/CodeSystem/condition-clinical'
codesystem "ConditionVerificationStatus":
  'http://terminology.hl7.org/CodeSystem/condition-ver-status'

// ── Value Sets ───────────────────────────────────────────────────────────────

valueset "<Clinical Concept>": '<VSAC canonical URL>'

// ── Inline Status Codes ───────────────────────────────────────────────────────

code "active": 'active' from "ConditionClinicalStatus" display 'Active'
code "recurrence": 'recurrence' from "ConditionClinicalStatus" display 'Recurrence'
code "relapse": 'relapse' from "ConditionClinicalStatus" display 'Relapse'
code "confirmed": 'confirmed' from "ConditionVerificationStatus" display 'Confirmed'

// ── Parameters ───────────────────────────────────────────────────────────────

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

// ── Context ───────────────────────────────────────────────────────────────────

context Patient

// ── Inline Helper Functions ───────────────────────────────────────────────────

/*
 * @description: Convert FHIR.Period or FHIR.dateTime to CQL Interval<DateTime>.
 * Inlined from ClinicalHelpers to make this library self-contained.
 */
define fluent function toInterval(choice Choice<FHIR.dateTime, FHIR.Period>):
  case
    when choice is FHIR.dateTime then
      Interval[FHIRHelpers.ToDateTime(choice as FHIR.dateTime),
               FHIRHelpers.ToDateTime(choice as FHIR.dateTime)]
    when choice is FHIR.Period then
      FHIRHelpers.ToInterval(choice as FHIR.Period)
    else null as Interval<DateTime>
  end

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  INITIAL POPULATION                                                      ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: initial-population
 * @description: <IPP clinical criteria>
 */
define "Initial Population":
  AgeInYearsAt(date from start of "Measurement Period") >= 18
    and exists "Qualifying Encounters"
    and "Has Active Diagnosis"

define "Qualifying Encounters":
  [Encounter: "<Encounter Value Set>"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"

define "Has Active Diagnosis":
  exists (
    [Condition: "<Diagnosis Value Set>"] C
      where C.clinicalStatus ~ "active"
        and C.verificationStatus ~ "confirmed"
        and C.recordedDate before end of "Measurement Period"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  DENOMINATOR                                                             ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: denominator
 * @description: All patients in the Initial Population.
 */
define "Denominator":
  "Initial Population"

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  DENOMINATOR EXCLUSION                                                   ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: denominator-exclusion
 * @description: <Exclusion clinical criteria>
 */
define "Denominator Exclusion":
  "Has Exclusionary Condition"

define "Has Exclusionary Condition":
  exists (
    [Condition: "<Exclusion Value Set>"] C
      where C.clinicalStatus ~ "active"
        and C.recordedDate before end of "Measurement Period"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  NUMERATOR                                                               ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: numerator
 * @description: <Numerator clinical criteria>
 * @improvement-notation: increase
 */
define "Numerator":
  "Has Qualifying Intervention"

define "Has Qualifying Intervention":
  exists (
    [MedicationRequest: "<Treatment Value Set>"] MR
      where MR.status in { 'active', 'completed' }
        and MR.intent = 'order'
        and MR.authoredOn.toInterval() during "Measurement Period"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  SUPPLEMENTAL DATA ELEMENTS                                              ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Sex
 */
define "SDE Sex":
  Patient.gender

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Age
 */
define "SDE Age":
  AgeInYearsAt(date from start of "Measurement Period")
```

### Notes on the self-contained pattern

- **`toInterval()` fluent function**: Inline the definition rather than importing from a helper library.
- **Condition status codes**: Define `"active"` and `"confirmed"` codes from the FHIR terminology code systems rather than using `FHIRCommon."active"`.
- **`authoredOn.toInterval()`**: `MedicationRequest.authoredOn` is a `FHIR.dateTime`; calling `.toInterval()` on it works when the fluent function accepts `Choice<FHIR.dateTime, FHIR.Period>`.
- **SDE Race/Ethnicity**: For a self-contained measure, SDE Race and Ethnicity may be omitted or simplified to avoid US Core extension traversal complexity. Include them when the runtime supports extension access.

---

/**
 * Composite measure: Comprehensive Diabetes Care
 * Scoring: Opportunity (patients meeting numerator across all components)
 * Component measures: CMS122 (HbA1c), CMS123 (Foot Exam), CMS124 (Eye Exam)
 */
library CMS.ComprehensiveDiabetesCare version '1.0.0'

using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers
include CMS.CMS122 version '12.0.000' called HbA1c
include CMS.CMS123 version '8.0.000' called FootExam
include CMS.CMS124 version '8.0.000' called EyeExam

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

context Patient

/*
 * @measure-component: initial-population
 * @description: Union of all component measure IPPs. Patient must be in at
 *   least one component measure's initial population.
 */
define "Initial Population":
  HbA1c."Initial Population"
    or FootExam."Initial Population"
    or EyeExam."Initial Population"
```
