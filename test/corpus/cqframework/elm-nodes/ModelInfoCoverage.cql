// Regression fixture for the generated FHIR ModelInfo tables.
//
// The property-type table used to be hand-maintained and partial, so a property
// it did not list silently skipped its implicit FHIRHelpers conversion. These
// properties span resources the hand-maintained table never covered, plus the
// binding, nested-complex and Date/DateTime cases that the generated tables made
// resolvable.
library ModelInfoCoverage version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'

context Patient

// ── properties absent from the previous hand-maintained table ─────────────────

define "ImmunizationLot": exists ([Immunization] I where I.lotNumber = 'LOT123')
define "ClaimUse": exists ([Claim] C where C.use = 'claim')
define "TaskDescription": exists ([Task] T where T.description = 'follow up')
define "GoalDescriptionText": exists ([Goal] G where G.description.text = 'target')

// ── binding (enum) types resolve through their System value type ──────────────

define "GenderCompared": Patient.gender = 'female'
define "EncounterStatusCompared": exists ([Encounter] E where E.status = 'finished')

// ── nested complex-type properties ────────────────────────────────────────────

define "NameFamily": exists ((Patient.name) N where N.family = 'Smith')
define "AddressCity": exists ((Patient.address) A where A.city = 'Boston')

// ── Date/DateTime implicit conversion ─────────────────────────────────────────

// Specimen.receivedTime is an instant, so the Date literal is converted.
define "InstantComparedToDate":
  exists ([Specimen] S where S.receivedTime before @2020-01-01)
// Patient.birthDate is a date, so both sides stay Dates.
define "DateComparedToDate": Patient.birthDate before @2020-01-01

// ── list-valued primitives are not converted in place ─────────────────────────

// Address.line is a List<FHIR.string>; the conversion belongs on the element, so
// the list itself must be left alone.
define "AddressLines": (Patient.address) A return A.line

// Consumed as a list, the elements are converted one by one through an implicit
// query rather than the list being promoted with ToList.
define "AddressLineIn":
  exists ((Patient.address) A where 'Main St' in A.line)
define "GivenNameIn":
  exists ((Patient.name) N where 'Ann' in N.given)
