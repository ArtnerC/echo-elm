/**
 * Measure: Rheumatoid Arthritis — Disease Activity Monitoring with DMARD Therapy
 * Library: RAMeasureTerminology
 * Version: 1.0.0
 * Steward: echo-elm project (illustrative measure)
 * Description: Terminology library for the RA Disease Activity Monitoring measure.
 *              Contains all value set declarations, code system declarations, and
 *              direct-reference codes. Contains no logic — imported by RAMeasure.
 *
 * Value set OIDs are from VSAC and reflect standard eCQM authoring conventions.
 * Verify and pin version strings when preparing for CMS eCQM submission.
 */
library RAMeasureTerminology version '1.0.0'

// ── Code Systems ──────────────────────────────────────────────────────────────

codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "LOINC": 'http://loinc.org'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "CPT": 'http://www.ama-assn.org/go/cpt'
codesystem "HCPCS": 'https://www.cms.gov/Medicare/Coding/HCPCSReleaseCodeSets'

codesystem "ConditionClinicalStatus":
  'http://terminology.hl7.org/CodeSystem/condition-clinical'
codesystem "ConditionVerificationStatus":
  'http://terminology.hl7.org/CodeSystem/condition-ver-status'

// ── Value Sets ────────────────────────────────────────────────────────────────

// Diagnosis
valueset "Rheumatoid Arthritis":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.113.12.1005'

// Qualifying encounters
valueset "Office Visit":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.101.12.1001'
valueset "Rheumatology Visit":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1143'

// Treatments
valueset "DMARD Therapy":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1273'
valueset "Biologics for Rheumatoid Arthritis":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1272'

// Disease activity assessments (LOINC-coded clinical observations)
valueset "Disease Activity Assessment":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1448'

// Laboratory monitoring (required for DMARD safety)
valueset "Complete Blood Count Panel":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.198.12.1050'
valueset "Hepatic Function Panel":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.198.12.1015'
valueset "C-Reactive Protein Test":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1225'
valueset "Erythrocyte Sedimentation Rate":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1226'

// Exclusion conditions — DMARD contraindications
valueset "Pregnancy":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.378'
valueset "Active Tuberculosis":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.111.12.1012'
valueset "Severe Hepatic Impairment":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.199.12.1070'

// General exclusion
valueset "Hospice Care Ambulatory":
  'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.526.3.1584'

// ── Direct-Reference Codes ────────────────────────────────────────────────────

// RA-specific SNOMED finding
code "Rheumatoid arthritis": '69896004' from "SNOMEDCT"
  display 'Rheumatoid arthritis (disorder)'

// Disease activity score LOINC codes (used for observation retrieval / annotation)
code "DAS28-ESR score": '54793-3' from "LOINC"
  display 'Disease activity score in 28 joints using ESR'
code "DAS28-CRP score": '55805-4' from "LOINC"
  display 'Disease activity score in 28 joints using CRP'
code "CDAI score": '85352-4' from "LOINC"
  display 'Clinical Disease Activity Index score'
code "SDAI score": '85354-0' from "LOINC"
  display 'Simplified Disease Activity Index score'

// Condition status codes (inline — no FHIRCommon dependency)
code "active": 'active' from "ConditionClinicalStatus"
  display 'Active'
code "recurrence": 'recurrence' from "ConditionClinicalStatus"
  display 'Recurrence'
code "relapse": 'relapse' from "ConditionClinicalStatus"
  display 'Relapse'
code "confirmed": 'confirmed' from "ConditionVerificationStatus"
  display 'Confirmed'
