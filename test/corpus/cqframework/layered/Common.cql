library Common version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define function "Normalize"(choice Choice<FHIR.dateTime, FHIR.Period>):
  case
    when choice is FHIR.dateTime then Interval[FHIRHelpers.ToDateTime(choice as FHIR.dateTime), FHIRHelpers.ToDateTime(choice as FHIR.dateTime)]
    when choice is FHIR.Period then FHIRHelpers.ToInterval(choice as FHIR.Period)
    else null as Interval<DateTime>
  end
define fluent function "toInterval"(choice Choice<FHIR.dateTime, FHIR.Period>): "Normalize"(choice)
define "Active": [Condition] C where C.clinicalStatus is not null
