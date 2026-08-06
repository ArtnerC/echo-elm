// Regression fixture for the implicit FHIRHelpers coercion rule.
//
// CQF inserts the FHIRHelpers conversion where a FHIR value is *consumed* as an
// operand of a typed operator — not at every property access. A property that is
// a define body, a query return expression, a list element, or the source of a
// further navigation keeps its raw FHIR value.
library FHIRCoercionContext version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'

parameter "Measurement Period" Interval<DateTime>

context Patient

// ── unconsumed positions: no conversion ───────────────────────────────────────

define "BareGender": Patient.gender
define "BareBirthDate": Patient.birthDate
define "GenderInList": { Patient.gender }
define "PeriodAsReturnExpression": ["Encounter"] E return E.period
// E.period is navigated into, so the structure — not a converted Interval — is
// the source; only the resulting dateTime is converted.
define "NavigatedPeriodStart":
  exists ["Encounter"] E where E.period.start during "Measurement Period"

// ── consumed positions: conversion applied ────────────────────────────────────

define "GenderCompared": Patient.gender = 'female'
define "BirthDateCompared": Patient.birthDate before @2000-01-01
define "PeriodAsInterval":
  exists ["Encounter"] E where E.period during "Measurement Period"
