// A branch that calls a helper returning Any by construction (a Message over
// null) is cast to the other branch's type, as a Message written in place is.
// (issues/08 appendix: an `if` whose else calls an included error helper)
library AnyBranchCast version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include AnyBranchHelpers version '1.0' called Common

context Patient

define "Else calls helper":
  if Patient.birthDate is not null then Patient.birthDate.value
  else Common."Raise"('no birth date')

define "Then calls helper":
  if Patient.birthDate is null then Common."Raise"('no birth date')
  else Patient.birthDate.value

define "Interval branch":
  if Patient.birthDate is null then Interval[@2020-01-01, @2021-01-01]
  else Common."Raise"('unexpected')

define "Declared Any":
  if Patient.active is not null then 'active'
  else Common."Declared any"('no flag')

define "Typed helper is not cast":
  if Patient.active is not null then 1.5
  else Common."Typed"(2.5)

define "Case with helper":
  case
    when Patient.birthDate is null then Common."Raise"('no birth date')
    else Patient.birthDate.value
  end

define "Birth date value": Patient.birthDate.value
