# Layer 02 — Library Architecture (Multi-File Organization)

Enterprise CQL measure authoring uses a layered library structure. Every measure project
should follow this pattern for maintainability, testability, and reuse.

---

## Standard multi-file structure

```
measure-project/
  input/cql/
    ├── terminology/
    │   ├── MeasureName-Terminology.cql         # All value sets + code system declarations
    │   └── SharedTerminology.cql               # Org-wide shared terminology
    ├── libraries/
    │   ├── FHIRHelpers.cql                     # Embedded copy of FHIRHelpers (if not from IG)
    │   ├── FHIRCommon.cql                      # Embedded copy of FHIRCommon
    │   ├── GlobalFunctions.cql                 # Org-wide helper functions
    │   ├── ClinicalHelpers.cql                 # Clinical domain helpers (negation, timing)
    │   ├── PatientCharacteristics.cql          # SDE, demographics, race/ethnicity, SDOH
    │   └── SupplementalDataElements.cql        # Shared SDE logic
    ├── measures/
    │   ├── MeasureName.cql                     # Primary measure library (all populations)
    │   └── MeasureName-Supplemental.cql        # Risk adjustment + stratifiers (if complex)
    └── tests/
        ├── MeasureName-Tests.cql               # CQL-based test assertions
        └── cases/                              # FHIR test case bundles
            ├── TestCase-IPP-True.json
            ├── TestCase-DENEX-Hospice.json
            └── TestCase-NUMER-Pass.json
```

---

## Library type hierarchy

### 1. Terminology library
Contains only terminology declarations — no logic. Imported by all other libraries.

```cql
library OrganizationName.MeasureName-Terminology version '1.0.0'

using FHIR version '4.0.1'

// ─── Value Sets ──────────────────────────────────────────────────────────────

valueset "Diabetes": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.103.12.1020'
valueset "HbA1c Laboratory Test": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.198.12.1013'
valueset "Inpatient Encounter": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.666.5.307'
valueset "Palliative Care Encounter": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.600.1.1579'
valueset "Hospice Care Ambulatory": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1584'

// ─── Code Systems ────────────────────────────────────────────────────────────

codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "LOINC": 'http://loinc.org'

// ─── Direct-Reference Codes ──────────────────────────────────────────────────

code "Systolic blood pressure": '8480-6' from "LOINC" display 'Systolic blood pressure'
```

### 2. Shared helper functions library
Clinical utility functions used across multiple measures.

```cql
library OrganizationName.ClinicalHelpers version '2.0.0'

using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers
include hl7.fhir.uv.cql.FHIRCommon version '4.0.1' called FHIRCommon

context Patient

// ─── Timing Utilities ────────────────────────────────────────────────────────

/*
 * @description: Convert a FHIR choice effective[x] to a CQL Interval<DateTime>.
 * Handles dateTime, instant, Period, and Timing (returns null for Timing).
 */
define fluent function toInterval(
  choice Choice<FHIR.dateTime, FHIR.Period, FHIR.instant, FHIR.Timing>
):
  case
    when choice is FHIR.dateTime then
      Interval[FHIRHelpers.ToDateTime(choice as FHIR.dateTime),
               FHIRHelpers.ToDateTime(choice as FHIR.dateTime)]
    when choice is FHIR.Period then
      FHIRHelpers.ToInterval(choice as FHIR.Period)
    when choice is FHIR.instant then
      Interval[FHIRHelpers.ToDateTime(choice as FHIR.instant),
               FHIRHelpers.ToDateTime(choice as FHIR.instant)]
    else null as Interval<DateTime>
  end

/*
 * @description: Normalize Observation.effective to a DateTime for sorting.
 */
define fluent function effectiveDateTime(observation FHIR.Observation):
  case
    when observation.effective is FHIR.dateTime then
      FHIRHelpers.ToDateTime(observation.effective as FHIR.dateTime)
    when observation.effective is FHIR.Period then
      start of FHIRHelpers.ToInterval(observation.effective as FHIR.Period)
    when observation.effective is FHIR.instant then
      FHIRHelpers.ToDateTime(observation.effective as FHIR.instant)
    else null as DateTime
  end

// ─── Negation Patterns ───────────────────────────────────────────────────────

/*
 * @description: True if the patient had an active hospice encounter or order
 *               during the given period. Standard exclusion pattern across
 *               many CMS quality measures.
 */
define fluent function hasHospiceDuring(period Interval<DateTime>):
  exists (
    [Encounter: "Hospice Care Ambulatory"] E
      where E.status in { 'finished', 'in-progress' }
        and E.period overlaps period
  )

// ─── QI-Core Negation ────────────────────────────────────────────────────────

/*
 * @description: True if the resource was a negation (not done / not ordered)
 *               using the QI-Core notDone extension pattern.
 */
define fluent function isNotDone(observation FHIR.Observation):
  (FHIRCommon.GetExtension(observation, 'http://hl7.org/fhir/us/qicore/StructureDefinition/qicore-notDoneValueSet')) is not null
```

### 3. Patient characteristics library
Demographics, supplemental data elements (SDEs), and risk adjustment variables.

```cql
library OrganizationName.PatientCharacteristics version '1.0.0'

using FHIR version '4.0.1'

include hl7.fhir.uv.cql.FHIRHelpers version '4.0.1' called FHIRHelpers

codesystem "OmbRaceCategoryCodes": 'urn:oid:2.16.840.1.113883.6.238'
codesystem "NullFlavors": 'http://terminology.hl7.org/CodeSystem/v3-NullFlavor'

context Patient

// ─── Race (US Core Race Extension) ───────────────────────────────────────────

define fluent function race(patient FHIR.Patient):
  (singleton from (
    patient.extension E
      where E.url = 'http://hl7.org/fhir/us/core/StructureDefinition/us-core-race'
  )) race
    let
      ombCategory: race.extension E where E.url = 'ombCategory' return E.value as FHIR.Coding,
      detailed:    race.extension E where E.url = 'detailed'    return E.value as FHIR.Coding,
      text:        singleton from (race.extension E where E.url = 'text' return E.value as FHIR.string)
    return { ombCategory: ombCategory, detailed: detailed, text: text.value }

// ─── Ethnicity (US Core Ethnicity Extension) ─────────────────────────────────

define fluent function ethnicity(patient FHIR.Patient):
  (singleton from (
    patient.extension E
      where E.url = 'http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity'
  )) eth
    let
      ombCategory: eth.extension E where E.url = 'ombCategory' return E.value as FHIR.Coding,
      text:        singleton from (eth.extension E where E.url = 'text' return E.value as FHIR.string)
    return { ombCategory: ombCategory, text: text.value }

// ─── Standard SDEs ───────────────────────────────────────────────────────────

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Ethnicity
 */
define "SDE Ethnicity":
  Patient.ethnicity()

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Payer
 */
define "SDE Payer":
  [Coverage: type in "Payer Type"] C
    return { type: C.type, period: C.period }

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Race
 */
define "SDE Race":
  Patient.race()

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Sex
 */
define "SDE Sex":
  Patient.gender
```

### 4. Primary measure library (all population expressions)
The only library referenced by the FHIR Measure resource.

See Layer 03 for full population annotation conventions.

---

## Include dependency graph pattern

```
Measure.cql
  ├── include Terminology.cql
  ├── include FHIRHelpers (from HL7 IG)
  ├── include FHIRCommon (from HL7 IG)
  ├── include ClinicalHelpers.cql
  │     ├── include FHIRHelpers
  │     └── include FHIRCommon
  ├── include PatientCharacteristics.cql
  │     └── include FHIRHelpers
  └── include SupplementalDataElements.cql
        └── include PatientCharacteristics.cql
```

Rules:
- **No circular dependencies.** A library must not include anything that includes it.
- **Depth limit ≤ 4.** Deeply nested include graphs indicate architecture problems.
- **Terminology library has no includes** (only model `using` declaration).
- **Test libraries include the measure library** (not the other way around).

---

## Naming conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Library | `Org.MeasureName` | `CMS.DiabetesHbA1c` |
| Library version | `major.minor.patch` | `3.0.0` |
| Value set | Title Case, full clinical meaning | `"Inpatient Encounter"` |
| Code system | Pascal case, abbreviation | `"SNOMEDCT"`, `"LOINC"` |
| Code | Title Case, clinical description | `"Systolic blood pressure"` |
| Expression | Title Case, full sentence meaning | `"Initial Population"` |
| Helper function | camelCase (if fluent) | `toInterval`, `systolic` |
| Helper function | PascalCase (if standalone) | `AgeInYearsAt` |
| Query alias | Single uppercase letter | `E`, `O`, `C`, `MR`, `P` |
| Parameter | Title Case | `"Measurement Period"` |

### Prohibited naming patterns
- Abbreviations in expression names (`IPP`, `DENOM`, `NUMER`) — always use full names
- Generic names (`"Logic"`, `"Helper"`, `"Common"`) without context
- Names that collide with CQL keywords (`And`, `Or`, `Not`, `In`, `From`, `Where`)

---

## Versioning strategy

### Semantic version bump rules
```
PATCH (x.x.Z): Typographic / formatting corrections that do not change semantics
MINOR (x.Y.0): New expressions added; existing expressions not changed
MAJOR (X.0.0): Any change to an existing expression that changes results
```

### Version compatibility across a measure set
- All shared libraries used by a single measure build SHOULD share the same `PATCH` level.
- Breaking library changes MUST bump `MAJOR` and MUST be announced in the change log.
- CMS eCQM programs typically lock library versions at submission time; never silently re-version.

---

## FHIR Library resource mapping

Each `.cql` file corresponds to one FHIR `Library` resource:

```json
{
  "resourceType": "Library",
  "url": "https://example.org/fhir/Library/CMS-DiabetesHbA1c",
  "version": "3.0.0",
  "name": "CMS_DiabetesHbA1c",
  "title": "CMS Diabetes HbA1c Control Measure Library",
  "status": "active",
  "type": {
    "coding": [{ "system": "http://terminology.hl7.org/CodeSystem/library-type",
                 "code": "logic-library" }]
  },
  "content": [
    {
      "contentType": "text/cql",
      "data": "<base64-encoded CQL>"
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

## Multi-measure shared library strategy

For an organization maintaining many measures:

```
org-shared/
  input/cql/
    MedLogicHelpers.cql        # Medication period functions
    EncounterHelpers.cql       # Encounter classification helpers
    DiagnosisHelpers.cql       # Problem list and diagnosis status
    LaboratoryHelpers.cql      # Lab result ranking and filtering
    VitalSignHelpers.cql       # Blood pressure, BMI, weight helpers
    NegationHelpers.cql        # Negation / not-done patterns
    TimingHelpers.cql          # Interval arithmetic helpers
    SDECommon.cql              # Standard SDE set (race, ethnicity, payer, sex)
```

Each shared library is published as an independent FHIR Library artifact with its own
canonical URL and semantic version, enabling independent release cycles.
