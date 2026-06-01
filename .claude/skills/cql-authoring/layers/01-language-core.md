# Layer 01 — CQL Language Core

CQL 1.5.3 reference for clinical informaticists. Read before writing any CQL expression.

---

## Library declaration (required, always first)

```cql
library OrganizationName.MeasureName version '1.0.0'
```

Rules:
- First non-comment line in every CQL file.
- Version MUST be `<major>.<minor>.<patch>` (semantic versioning).
- Namespace (`OrganizationName.`) is optional but strongly recommended for enterprise authoring.
- Library name must be unique within the namespace and match the FHIR Library `name` element.

---

## Using declarations (data model)

```cql
// FHIR R4 — base model for most modern quality measures
using FHIR version '4.0.1'

// QI-Core — preferred for US Realm quality measures (profile-informed)
using QICore version '7.0.0'

// US Core — alternative to QI-Core for lighter-weight profiles
using USCore version '7.0.0'

// QDM — legacy model for eCQMs predating FHIR migration
using QDM version '5.6'
```

Rules:
- Version string MUST be included.
- Use QI-Core for new US Realm measures unless the program specifies otherwise.
- Never mix FHIR and QDM in the same library's retrieves.

---

## Include declarations (shared libraries)

```cql
include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers
include hl7.fhir.uv.cql.FHIRCommon version '4.0.1' called FHIRCommon
include Organization.SharedLogic version '2.0.0' called SharedLogic
include Organization.Terminology version '1.3.0' called Terminology
```

Rules:
- The `called` alias is required for libraries other than FHIRHelpers (which is auto-invoked).
- FHIRHelpers MUST be included when using FHIR model to enable implicit conversions.
- Never include FHIRHelpers without specifying the version; it must match the FHIR model version.

---

## Terminology declarations

### Value set
```cql
// Version-independent (preferred for active content)
valueset "Inpatient Encounter": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.666.5.307'

// Version-pinned (required for reproducibility in archived measures)
valueset "Diabetes": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020'
  version '20230101'

// Non-VSAC value set
valueset "Emergency Department Visit": 'http://example.org/fhir/ValueSet/emergency-encounter'
```

### Code system
```cql
codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "LOINC": 'http://loinc.org'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "CPT": 'http://www.ama-assn.org/go/cpt'
codesystem "RXNORM": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "HCPCS": 'https://www.cms.gov/Medicare/Coding/HCPCSReleaseCodeSets'
codesystem "MeasurePopulationType": 'http://terminology.hl7.org/CodeSystem/measure-population'
codesystem "ObservationCategoryCodes": 'http://terminology.hl7.org/CodeSystem/observation-category'
```

### Direct-reference code (inline code)
```cql
code "Systolic blood pressure": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic blood pressure": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "Well child visit": '410620009' from "SNOMEDCT" display 'Well child visit (procedure)'
```

### Concept (multi-code)
```cql
concept "Bilateral mastectomy": {
  "Unilateral mastectomy, right",
  "Unilateral mastectomy, left"
} display 'Bilateral mastectomy'
```

---

## Parameter declarations

```cql
// Measurement period — standard parameter for all quality measures
parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

// Optional parameters for filtering
parameter "Product Line" String default null

// Boolean feature flags
parameter "Enable Debug" Boolean default false
```

Rules:
- `"Measurement Period"` is the standard parameter name; do not rename it.
- Always provide a sensible default for the measurement period parameter.

---

## Context declaration

```cql
// Patient context — use for most quality measures (one record per patient)
context Patient

// Practitioner context — for provider-level reporting
context Practitioner

// Unfiltered — cross-patient aggregations (rarely needed in measure libraries)
context Unfiltered
```

---

## Expression definitions

```cql
// Simple Boolean expression
define "Has Active Diabetes Diagnosis":
  exists (
    [Condition: "Diabetes"] C
      where C.clinicalStatus ~ "active"
  )

// Returns a list of resources
define "Active Diabetes Conditions":
  [Condition: "Diabetes"] C
    where C.clinicalStatus ~ "active"
      and C.verificationStatus ~ "confirmed"

// Derived value
define "Most Recent HbA1c":
  Last(
    [Observation: "HbA1c Laboratory Test"] O
      where O.status in { 'final', 'amended', 'corrected' }
      sort by issued.value
  )

// Date/time-relative filter
define "HbA1c During Measurement Period":
  [Observation: "HbA1c Laboratory Test"] O
    where O.status in { 'final', 'amended', 'corrected' }
      and O.effective.toInterval() during "Measurement Period"
```

---

## Function definitions

### Simple function
```cql
define function "AgeInYearsAt"(asOf DateTime):
  AgeInYearsAt(asOf)

define function "ToDate"(Value Choice<FHIR.dateTime, FHIR.date, FHIR.instant>):
  case
    when Value is FHIR.dateTime then date from FHIRHelpers.ToDateTime(Value as FHIR.dateTime)
    when Value is FHIR.date then FHIRHelpers.ToDate(Value as FHIR.date)
    when Value is FHIR.instant then date from FHIRHelpers.ToDateTime(Value as FHIR.instant)
    else null
  end
```

### Fluent function (CQL 1.5+, preferred for chaining)
```cql
// Convert FHIR choice type to interval
define fluent function toInterval(choice Choice<FHIR.dateTime, FHIR.Period>):
  case
    when choice is FHIR.dateTime then
      Interval[FHIRHelpers.ToDateTime(choice as FHIR.dateTime),
               FHIRHelpers.ToDateTime(choice as FHIR.dateTime)]
    when choice is FHIR.Period then
      FHIRHelpers.ToInterval(choice as FHIR.Period)
    else null as Interval<DateTime>
  end

// Access slice from Observation.component
define fluent function systolic(observation FHIR.Observation):
  singleton from (
    observation.component C
      where C.code ~ "Systolic blood pressure"
  )

define fluent function diastolic(observation FHIR.Observation):
  singleton from (
    observation.component C
      where C.code ~ "Diastolic blood pressure"
  )

// Extension access
define fluent function birthsex(patient FHIR.Patient):
  (singleton from (
    patient.extension E
      where E.url = 'http://hl7.org/fhir/us/core/StructureDefinition/us-core-birthsex'
  )).value as FHIR.code
```

---

## Query syntax

### Basic retrieve-and-filter
```cql
define "Qualifying Encounters":
  [Encounter: "Inpatient Encounter"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"
```

### Multi-source query with relationship
```cql
define "Encounters With Qualifying Diagnosis":
  [Encounter: "Inpatient Encounter"] E
    with [Condition: "Diabetes"] C
      such that C.encounter.references(E)
        and C.clinicalStatus ~ "active"
    where E.status = 'finished'
      and E.period during "Measurement Period"
```

### Without clause (exclusion)
```cql
define "Encounters Without Palliative Care":
  [Encounter: "Inpatient Encounter"] E
    without [Procedure: "Palliative Care"] PC
      such that PC.performed.toInterval() overlaps E.period
    where E.status = 'finished'
```

### Let clause
```cql
define "Most Recent Qualifying Encounter":
  Last(
    [Encounter: "Inpatient Encounter"] E
      let period: E.period
      where E.status = 'finished'
        and period during "Measurement Period"
      sort by start of period
  )
```

### Return projection
```cql
define "Encounter Summary":
  [Encounter: "Inpatient Encounter"] E
    return {
      id: E.id,
      start: start of E.period,
      end: end of E.period,
      lengthOfStay: hours between start of E.period and end of E.period
    }
```

### Aggregate (CQL 1.5+)
```cql
define "Total LOS in Hours":
  Sum(
    [Encounter: "Inpatient Encounter"] E
      return hours between start of E.period and end of E.period
  )
```

---

## Operators reference

### Interval operators (critical for clinical logic)

```cql
// Membership
E.period during "Measurement Period"        // Period is contained within measurement period
E.period overlaps "Measurement Period"      // Any overlap
E.period starts during "Measurement Period" // Start falls within
E.period ends during "Measurement Period"   // End falls within
E.period includes Today()                   // Period contains a point

// Relative to a point
30 days or less before Today()
12 months or less before end of "Measurement Period"
starts 90 days or less after end of E.period

// Comparison
difference in days between start of A and start of B
duration in days of E.period
```

### Date/time operators

```cql
Today()                          // Current date
Now()                            // Current date-time
start of "Measurement Period"    // Lower bound of interval
end of "Measurement Period"      // Upper bound of interval
date from encounterDateTime       // Extract date component
AgeInYearsAt(start of "Measurement Period")
AgeInYearsAt(Today())
```

### List operators

```cql
exists(list)                     // Non-empty
Count(list)                      // Size
First(list)                      // First element (null if empty)
Last(list)                       // Last element (null if empty)
singleton from list              // Exactly one element, null or error
flatten list                     // Flatten nested lists
list1 union list2                // All elements
list1 intersect list2            // Common elements
list1 except list2               // Difference
list contains element            // Membership
element in list                  // Membership (reverse)
```

### Null safety

```cql
// Null-conditional access
X is null
X is not null
Coalesce(A, B, C)               // First non-null
if X is null then default else X
X ?? default                    // Null-coalescing (CQL 1.5+)
```

---

## Type system

### CQL system types
```
Boolean, Integer, Long, Decimal, String, Quantity, Ratio,
DateTime, Date, Time, Interval<T>, List<T>, Tuple {...},
Choice<T1, T2>, Code, Concept, Vocabulary, Any
```

### Type conversion
```cql
// Explicit cast
(O.effective as FHIR.Period)
(O.value as FHIR.Quantity)

// Type test
O.effective is FHIR.Period
O.effective is FHIR.dateTime

// Convert
ToString(42)
ToInteger('7')
ToDecimal('3.14')
ToQuantity('5.0 mg')

// FHIRHelpers conversions (implicit when FHIRHelpers included)
FHIRHelpers.ToDateTime(dt)
FHIRHelpers.ToInterval(period)
FHIRHelpers.ToConcept(coding)
FHIRHelpers.ToQuantity(quantity)
```

---

## Comments and documentation style

```cql
// Single-line comment for inline explanation

/*
 * Multi-line block comment for function or expression documentation.
 * Describe clinical intent, not just code behavior.
 * Include relevant measure ID or specification reference.
 */

/**
 * @description: Returns true if the patient has an active HbA1c result
 *               ≥ 9.0% within the measurement period. Per NQF 0059.
 * @returns: Boolean
 */
define "Poor Glycemic Control":
  ...
```

---

## Common clinical patterns

### Age-based eligibility
```cql
define "Patient Age 18 to 75":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[18, 75]
```

### Most recent result
```cql
define "Most Recent HbA1c Result":
  Last(
    [Observation: "HbA1c Laboratory Test"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and O.effective.toInterval() before end of "Measurement Period"
      sort by FHIRHelpers.ToDateTime(O.issued)
  )
```

### Medication active during period
```cql
define "Active Metformin":
  [MedicationRequest: "Metformin"] MR
    where MR.status = 'active'
      and MR.intent = 'order'
      and MR.medication.toInterval() overlaps "Measurement Period"
```

### Encounter length-of-stay
```cql
define "Long Inpatient Stays":
  [Encounter: "Inpatient Encounter"] E
    where E.status = 'finished'
      and duration in days of E.period >= 2
```

### Hospice / palliative care exclusion (common pattern)
```cql
define "Has Palliative Care During Measurement Period":
  exists (
    [Encounter: "Palliative Care Encounter"] PC
      where PC.status = 'finished'
        and PC.period overlaps "Measurement Period"
  )
  or exists (
    [Procedure: "Palliative Care Intervention"] PC
      where PC.status = 'completed'
        and PC.performed.toInterval() overlaps "Measurement Period"
  )
```

---

## CQL formatting standards

- Indent: 2 spaces (no tabs)
- Lines: ≤ 100 characters
- Operator keywords: lowercase (`and`, `or`, `not`, `in`, `where`, `return`, `sort by`)
- Reserved words as expression names: always quote with double quotes
- Expression names: Title Case with full clinical meaning (`"Initial Population"`, not `"IPP"`)
- Alias names: Single uppercase letter or short meaningful name (`E`, `O`, `C`, `MR`)
- Opening `(` or `[` at end of line for multi-line expressions
- Binary operators (`and`, `or`) start the new line, not end the previous
