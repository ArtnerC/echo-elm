# Layer 08 — Enterprise Patterns, Governance, and Common Pitfalls

This layer covers organization-wide CQL governance, authoring pitfalls, performance,
program-specific rules (CMS, HEDIS, Medicaid), change management, and code review checklists.

---

## Enterprise library governance model

### Role structure

| Role | Responsibilities |
|------|-----------------|
| CQL Architect | Defines shared library interfaces, naming conventions, dependency graph rules |
| Clinical Informaticist | Translates clinical intent to CQL logic; validates population definitions |
| Terminology Steward | Owns value set governance; approves VSAC changes |
| Quality Engineer | Writes test cases; runs parity harness; owns CI gates |
| Measure Owner | Accountable for clinical accuracy; signs off before submission |
| Data Engineer | Validates data requirements against source systems |

### Governance artifacts

```
governance/
  naming-conventions.md        # Expression naming, library naming, alias rules
  dependency-policy.md         # Shared library versioning and upgrade rules
  value-set-management.md      # VSAC lifecycle, custom value set process
  change-request-template.md   # Required before any breaking library change
  review-checklist.md          # Pre-merge gate for all CQL changes
  submission-checklist.md      # Pre-submission gate for eCQM programs
```

---

## Naming conventions (enterprise rules)

### Expression names — always Title Case, full clinical meaning

```cql
// CORRECT
define "Initial Population":
define "Denominator Exclusion":
define "Has Active Diabetes Diagnosis":
define "Most Recent HbA1c Result":
define "Qualifying Outpatient Encounters":
define "Stratifier Age 18-44":

// WRONG — abbreviations
define "IPP":
define "DENEX":
define "DX_Active":
define "HbA1c_Latest":
```

### Function names — fluent functions use camelCase; standalone use PascalCase

```cql
// Fluent (called as method)
define fluent function toInterval(...)
define fluent function systolic(...)
define fluent function effectiveDateTime(...)

// Standalone (called as function)
define function AgeInYearsAt(...)
define function EncounterDuration(encounter FHIR.Encounter)
define function MostRecentResult(observations List<FHIR.Observation>)
```

### Parameter names — Title Case, used as query alias

```cql
parameter "Measurement Period" Interval<DateTime>
// Used: "Measurement Period" throughout the library
```

### Query alias rules

```cql
// CORRECT: Single uppercase abbreviation for simple queries
[Encounter: "Office Visit"] E
[Condition: "Diabetes"] C
[Observation: "HbA1c"] O
[MedicationRequest: "Statin"] MR
[Procedure: "Colonoscopy"] P
[DiagnosticReport: "Mammography"] D
[Coverage: "Payer"] Cov

// CORRECT: Two letters for disambiguation
[Encounter: "Inpatient"] IE
[Encounter: "Outpatient"] OE

// WRONG: full words or snake_case
[Encounter: "Office Visit"] encounter
[Condition: "Diabetes"] diabetes_condition
```

---

## Code review checklist

### Clinical correctness
```
□ IPP reflects the correct denominator eligibility population (age range, enrollment type)
□ Denominator is a subset of or equal to IPP — never broader
□ Numerator is a subset of or equal to Denominator
□ All exclusion criteria have clinical justification in comments
□ Exception criteria are documented (denominator exceptions only — numerator failures, never successes)
□ Improvement notation is documented (increase or decrease) and correct
□ Measurement period parameter is used consistently; no hardcoded year references
□ HbA1c result comparison uses > 9.0 (not >=) — consistent with measure specification
```

### Technical correctness
```
□ Status filters on all retrieved resources (no unfiltered retrieves)
□ Choice types handled with case-when or fluent functions (not raw access)
□ UCUM units on all Quantity literals
□ Code comparisons use ~ (not =) for FHIR.Coding / FHIR.CodeableConcept
□ Value set membership uses in (not ~ or =)
□ No circular includes
□ FHIRHelpers.ToDateTime / FHIRHelpers.ToInterval used for FHIR primitive conversions
□ Null-safe arithmetic (no division by zero risk in CV measures)
□ Population expressions are exported (not private)
□ All @measure-component annotations present and accurate
```

### Terminology
```
□ All value set canonical URLs are VSAC format (not raw OIDs)
□ No value set versions pinned without documented requirement
□ No custom code system URLs without a published CodeSystem resource
□ All direct-reference codes have display values
```

### Testing
```
□ Test cases cover all population strata (IPP yes/no, DENOM yes/no, NUMER yes/no)
□ Test cases cover each exclusion criterion independently
□ Boundary value tests (age boundaries, date boundaries, threshold values)
□ Null/missing data tests for each key data element
□ Expected MeasureReport population counts match clinical intent
□ Parity harness passes (echo-elm output matches CQF golden)
```

---

## Common CQL authoring pitfalls

### 1. Using `=` instead of `~` for code comparison
```cql
// WRONG: = on FHIR.CodeableConcept compares object identity
Condition.clinicalStatus = FHIRCommon."active"

// CORRECT: ~ compares system + code only
Condition.clinicalStatus ~ FHIRCommon."active"
```

### 2. Missing status filter
```cql
// WRONG: retrieves cancelled, entered-in-error records
[MedicationRequest: "Statin Medications"]

// CORRECT
[MedicationRequest: "Statin Medications"] MR
  where MR.status in { 'active', 'completed' }
    and MR.intent = 'order'
```

### 3. Date arithmetic on FHIR.dateTime (missing FHIRHelpers)
```cql
// WRONG: FHIR.dateTime is not a CQL DateTime
Observation.effective + 30 days

// CORRECT
FHIRHelpers.ToDateTime(Observation.effective as FHIR.dateTime) + 30 days
// OR use a fluent helper:
Observation.effective.toInterval() starts after (Now() - 30 days)
```

### 4. Not using `Last` / `First` with explicit sort
```cql
// WRONG: result set order is non-deterministic
singleton from (
  [Observation: "HbA1c"] O
    where O.status in { 'final', 'amended' }
      and O.effective.toInterval() during "Measurement Period"
)

// CORRECT: explicit ascending sort + Last
Last(
  [Observation: "HbA1c"] O
    where O.status in { 'final', 'amended' }
      and O.effective.toInterval() during "Measurement Period"
    sort by O.effective.effectiveDateTime() ascending
)
```

### 5. Denominator re-implementing IPP logic
```cql
// WRONG: duplicated logic — drift risk
define "Denominator":
  AgeInYearsAt(date from start of "Measurement Period") in Interval[18, 75]
    and exists "Qualifying Encounters"
    and "Has Active Diabetes Diagnosis"

// CORRECT: Denominator delegates to IPP
define "Denominator":
  "Initial Population"  // or: "Initial Population" and <additional criteria>
```

### 6. Interval boundary confusion (starts before vs during)
```cql
// WRONG for onset check: "during" requires the entire interval during the period
Condition.onset.toInterval() during "Measurement Period"

// CORRECT: onset can start before the period, just needs to exist
Condition.onset.toInterval() starts before end of "Measurement Period"
// OR more precisely: any overlap
Condition.onset.toInterval() overlaps "Measurement Period"
```

### 7. Forgetting that `null` propagates
```cql
// If Most Recent HbA1c Result returns null (no result), this is null (not false)
define "Numerator":
  "Most Recent HbA1c Result" > 9.0 '%'
// null > 9.0 = null → patient is NOT in Numerator (null is treated as false in measure eval)
// This is CORRECT behavior — document it in comments

// WRONG fix attempt: this hides the null
define "Numerator":
  Coalesce("Most Recent HbA1c Result", 0.0 '%') > 9.0 '%'
// Incorrectly treats missing HbA1c as controlled (< 9.0%) — clinical error!
```

### 8. Quantity literal without units
```cql
// WRONG
observation.value > 9.0     // unitless — runtime error if value has units

// CORRECT
(observation.value as FHIR.Quantity) > 9.0 '%'
```

### 9. `singleton from` on a non-singleton list
```cql
// WRONG: if multiple components match, runtime error
singleton from (
  bp.component C
    where C.code ~ "Systolic blood pressure"
).value

// CORRECT: guard against multiples or use First/Last
First(
  bp.component C
    where C.code ~ "Systolic blood pressure"
).value
```

---

## Performance optimization patterns

### Push retrieves to the server (code filter in retrieve)
```cql
// SLOW: retrieves all observations, filters in CQL engine
[Observation] O
  where O.code in "HbA1c Laboratory Test"

// FAST: code filter in retrieve (FHIR server evaluates)
[Observation: "HbA1c Laboratory Test"] O
```

### Avoid redundant evaluations with let
```cql
// SLOW: subquery evaluated multiple times
define "Qualifying Encounters With Comorbidities":
  [Encounter: "Office Visit"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"
      and exists (
        [Condition: "Diabetes"] C
          where C.encounter.references(E)
      )
      and exists (
        [Condition: "Hypertension"] C
          where C.encounter.references(E)
      )

// BETTER: pre-compute the encounter list once
define "Qualifying Encounters":
  [Encounter: "Office Visit"] E
    where E.status = 'finished'
      and E.period during "Measurement Period"

define "Qualifying Encounters With Comorbidities":
  "Qualifying Encounters" E
    where exists (
      [Condition: "Diabetes"] C
        where C.encounter.references(E)
    )
    and exists (
      [Condition: "Hypertension"] C
        where C.encounter.references(E)
    )
```

### Minimize traversal depth
```cql
// PREFER: flat reference check
MR.encounter.references(E)

// AVOID where possible: deep .resolve() chains
MR.encounter.resolve() as Encounter   // requires server-side resolve
```

---

## Program-specific rules

### CMS eCQM (MIPS, Medicaid, EP/EC)
- Measure URL: `https://madie.cms.gov/Measure/<CMS-ID>`
- Use MADIE for authoring; echo-elm for independent translation verification
- Annual program year value set versions MUST match the published VSAC set
- Supplemental data elements (race, ethnicity, payer, sex) are required
- Both CQL and ELM must be embedded in Library resources
- Translator software system extension required on Library

### NCQA HEDIS
- HEDIS measures use QDM (not FHIR) through 2024; FHIR HEDIS transition ongoing
- HEDIS exclusion criteria include specific age bands and continuous enrollment requirements
- HEDIS administrative exclusions differ from clinical exclusions — do not conflate
- HEDIS rate specifications define eligible populations per measurement year with specific enrollment periods

### Joint Commission (TJC) eCQMs
- TJC measures typically use inpatient encounter basis
- Encounter-based measures: population expressions return lists of encounters, not Boolean
- Timing requirements are critical: many TJC measures measure within N hours of encounter

### Commercial / Medicaid (state)
- HEDIS technical specifications apply for most commercial payers
- Medicaid programs may use custom specifications — always reference the program-specific spec
- Age-specific eligibility often differs (Medicaid may extend upper age beyond 75)

---

## Change management workflow

### Breaking change process
1. **Identify change type**: Is this a breaking change (expression result changes)?
2. **Open change request**: Document old behavior, new behavior, clinical rationale
3. **Clinical sign-off**: Measure owner approves
4. **Terminology impact**: Does the change affect value sets? → Terminology steward approval
5. **Version bump**: MAJOR version for breaking changes (`X.0.0`)
6. **Test case update**: Update expected results in all affected test cases
7. **Parity rebaseline**: Re-run parity harness; update goldens if needed
8. **Announce**: Notify downstream consumers (echo-qm, reporting tools)

### Non-breaking change process (minor/patch)
1. Implement change
2. Update tests (add, don't remove)
3. MINOR version for new expressions; PATCH for formatting/comment-only changes
4. Run full test suite + parity

---

## CQL expression documentation standard

Every public expression and function MUST have a documentation block:

```cql
/*
 * @measure-component: <component>           (if a measure component)
 * @description: <clinical intent>           (required for all public expressions)
 * @logic: <one-line summary of logic>       (optional; add when not obvious)
 * @guidance: <clinical guidance reference>  (optional; cite measure spec section)
 * @test: <test case bundle reference>       (optional; link to test case)
 */
define "Most Recent HbA1c Result":
  ...
```

Private (helper) expressions should have at minimum a `// Comment` describing purpose.

---

## Submission pre-flight checklist

Before CMS eCQM submission:

```
□ Measure CQL version matches Library resource version
□ All Library resources have CQL + ELM (XML + JSON) embedded (base64)
□ ELM produced with: EnableAnnotations, EnableLocators, DisableListDemotion,
    DisableListPromotion, compatibilityLevel 1.5, SignatureLevel Overloads
□ Translator version recorded in Library.extension (cqfm-softwaresystems)
□ All value set versions pinned to the submission year VSAC release
□ All relatedArtifact entries present for value sets in Library
□ All dataRequirement elements complete (type, profile, mustSupport, codeFilter)
□ Measure.effectivePeriod matches the program year
□ Test cases pass in CQF Ruler or Bonnie for all population strata
□ Parity harness green: echo-elm ELM matches CQF golden
□ MADIE validation passes (if using MADIE for submission)
□ Clinical review sign-off obtained and recorded
□ Change log updated with this version's delta
```
