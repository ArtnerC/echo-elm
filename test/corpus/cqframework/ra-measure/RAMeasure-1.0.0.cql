/**
 * Measure: Rheumatoid Arthritis — Disease Activity Monitoring with DMARD Therapy
 * Library: RAMeasure
 * Version: 1.0.0
 * Type: Proportion
 * Basis: Patient
 * Steward: echo-elm project (illustrative measure)
 * Improvement Notation: Increase (higher score = better monitoring adherence)
 *
 * Denominator: Adult patients (≥ 18 years) with an established, active RA diagnosis
 *   AND at least two qualifying encounters (office visit or rheumatology visit) during
 *   the measurement period.
 *
 * Denominator Exclusion: Active pregnancy, active tuberculosis, or severe hepatic
 *   impairment — all DMARD contraindications. Hospice care is also excluded.
 *
 * Numerator: Patients who received a DMARD or biologic therapy order AND had at least
 *   one documented disease activity assessment (DAS28, CDAI, SDAI, or equivalent)
 *   during the measurement period.
 *
 * Supplemental Data: Sex, Age (at start of measurement period).
 *
 * Clinical Rationale:
 *   ACR guidelines recommend treating-to-target with regular disease activity monitoring
 *   for all patients with active RA receiving DMARD or biologic therapy. Monitoring
 *   frequency should be every 1–3 months until remission or low disease activity is
 *   sustained, then every 3–6 months. This measure assesses whether patients with RA
 *   are receiving both DMARD therapy and documented disease activity monitoring.
 */
library RAMeasure version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers
include RAMeasureTerminology version '1.0.0' called Terminology

// ── Parameters ────────────────────────────────────────────────────────────────

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01, @2024-12-31]

// ── Context ───────────────────────────────────────────────────────────────────

context Patient

// ── Shared Building Blocks ────────────────────────────────────────────────────

/*
 * @description: Qualifying encounters — office visits or rheumatology visits
 *   that are finished and whose period falls during the measurement period.
 *   Used in both the IPP eligibility test and the encounter count test.
 */
define "Qualifying Encounters":
  ( [Encounter: Terminology."Office Visit"]
    union [Encounter: Terminology."Rheumatology Visit"] ) E
    where E.status = 'finished'
      and E.period during "Measurement Period"

/*
 * @description: Active, confirmed RA conditions recorded on or before the end
 *   of the measurement period. Includes recurrence and relapse per clinical
 *   status semantics — these represent ongoing disease activity, not resolved
 *   episodes. Excludes inactive, remission, and resolved status per measure spec.
 */
define "Active RA Diagnoses":
  [Condition: Terminology."Rheumatoid Arthritis"] C
    where ( C.clinicalStatus ~ Terminology."active"
        or C.clinicalStatus ~ Terminology."recurrence"
        or C.clinicalStatus ~ Terminology."relapse" )
      and C.verificationStatus ~ Terminology."confirmed"
      and C.recordedDate before end of "Measurement Period"

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  INITIAL POPULATION                                                      ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: initial-population
 * @description: Adult patients (≥ 18 years old at the start of the measurement
 *   period) with an established, active rheumatoid arthritis diagnosis AND at
 *   least two qualifying encounters (office visit or rheumatology visit) during
 *   the measurement period. The two-encounter threshold aligns with ACR guidance
 *   on minimum monitoring frequency and screens out patients with only a single
 *   incidental RA-related contact.
 */
define "Initial Population":
  AgeInYearsAt(start of "Measurement Period") >= 18
    and exists "Active RA Diagnoses"
    and "Has Two or More Qualifying Encounters"

define "Has Two or More Qualifying Encounters":
  Count("Qualifying Encounters") >= 2

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  DENOMINATOR                                                             ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: denominator
 * @description: All patients in the Initial Population. The denominator equals
 *   the IPP for this measure — no additional criteria are applied.
 */
define "Denominator":
  "Initial Population"

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  DENOMINATOR EXCLUSION                                                   ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: denominator-exclusion
 * @description: Patients with active contraindications to DMARD therapy:
 *   pregnancy, active tuberculosis, or severe hepatic impairment. Each
 *   condition must be clinically active and recorded on or before the end of
 *   the measurement period. Patients enrolled in hospice care are also excluded
 *   per standard CMS quality measure exclusion logic.
 */
define "Denominator Exclusion":
  "Has DMARD Contraindication"
    or "Has Hospice During Measurement Period"

/*
 * @description: True if the patient has any active DMARD contraindication
 *   — pregnancy, active TB, or severe hepatic impairment — documented on
 *   or before the end of the measurement period.
 */
define "Has DMARD Contraindication":
  exists (
    ( [Condition: Terminology."Pregnancy"]
        union [Condition: Terminology."Active Tuberculosis"]
        union [Condition: Terminology."Severe Hepatic Impairment"] ) C
      where C.clinicalStatus ~ Terminology."active"
        and C.recordedDate before end of "Measurement Period"
  )

/*
 * @description: True if the patient had an active hospice encounter during
 *   the measurement period. Standard CMS exclusion pattern.
 */
define "Has Hospice During Measurement Period":
  exists (
    [Encounter: Terminology."Hospice Care Ambulatory"] E
      where E.status in { 'finished', 'in-progress' }
        and E.period overlaps "Measurement Period"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  NUMERATOR                                                               ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: numerator
 * @description: Patients with BOTH an active DMARD or biologic therapy order
 *   AND at least one documented disease activity assessment during the
 *   measurement period. Both elements must be present:
 *   — DMARD/biologic: evidences treat-to-target therapy is underway.
 *   — Disease activity assessment: evidences monitoring is occurring.
 *   Improvement notation is INCREASE — more patients meeting both criteria
 *   is the desired direction.
 * @improvement-notation: increase
 */
define "Numerator":
  "Has Active DMARD or Biologic Therapy"
    and "Has Disease Activity Assessment"

/*
 * @description: True if the patient has an active or completed DMARD or
 *   biologic therapy order (intent = 'order') with authoredOn during the
 *   measurement period. Includes both synthetic DMARDs (methotrexate,
 *   hydroxychloroquine, sulfasalazine, leflunomide) and biologic agents
 *   (TNF inhibitors, IL-6 inhibitors, JAK inhibitors, etc.).
 */
define "Has Active DMARD or Biologic Therapy":
  exists (
    ( [MedicationRequest: Terminology."DMARD Therapy"]
        union [MedicationRequest: Terminology."Biologics for Rheumatoid Arthritis"] ) MR
      where MR.status in { 'active', 'completed' }
        and MR.intent = 'order'
        and MR.authoredOn during "Measurement Period"
  )

/*
 * @description: True if the patient has at least one disease activity
 *   assessment observation (DAS28-ESR, DAS28-CRP, CDAI, SDAI, or
 *   equivalent LOINC-coded score) with a final/amended/corrected status
 *   during the measurement period.
 */
define "Has Disease Activity Assessment":
  exists (
    [Observation: Terminology."Disease Activity Assessment"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and O.issued during "Measurement Period"
  )

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  STRATIFIERS                                                             ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: stratifier
 * @strat-name: Stratifier Age 18-44
 * @description: Patients aged 18–44 years at the start of the measurement
 *   period. RA onset in this group is often more aggressive; monitoring
 *   adherence in younger adults is a clinical priority.
 */
define "Stratifier Age 18-44":
  AgeInYearsAt(start of "Measurement Period") in Interval[18, 44]

/*
 * @measure-component: stratifier
 * @strat-name: Stratifier Age 45-64
 * @description: Patients aged 45–64 years at the start of the measurement
 *   period.
 */
define "Stratifier Age 45-64":
  AgeInYearsAt(start of "Measurement Period") in Interval[45, 64]

/*
 * @measure-component: stratifier
 * @strat-name: Stratifier Age 65 and Over
 * @description: Patients aged 65 years or older at the start of the
 *   measurement period. This group has heightened DMARD safety considerations
 *   (renal function, infection risk, drug interactions).
 */
define "Stratifier Age 65 and Over":
  AgeInYearsAt(start of "Measurement Period") >= 65

// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  SUPPLEMENTAL DATA ELEMENTS                                              ║
// ╚══════════════════════════════════════════════════════════════════════════╝

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Sex
 * @description: Patient administrative sex for stratified reporting.
 *   RA disproportionately affects women; sex-based reporting supports
 *   equity analysis.
 */
define "SDE Sex":
  Patient.gender

/*
 * @measure-component: supplemental-data
 * @sde-name: SDE Age
 * @description: Patient age in whole years at the start of the measurement
 *   period. Used for risk adjustment and equity analytics.
 */
define "SDE Age":
  AgeInYearsAt(start of "Measurement Period")
