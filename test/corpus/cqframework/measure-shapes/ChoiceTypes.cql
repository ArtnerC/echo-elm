// Choice types: alternative order, result types, and argument casts.
//
// CQF keeps a choice's alternatives sorted by the Java toString() of each type —
// FHIR.Age < FHIR.dateTime < System.Integer < interval<…> < list<…> — even for a
// declared operand type, and records a result type on the specifier itself. A
// value passed to a choice-typed parameter is cast to the choice. And a
// choice-typed FHIR property (Condition.onset) has the choice as its type, where
// it used to have none because the ModelInfo tables skipped choice elements.
// (issues/06)
library ChoiceTypes version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

define function "Mixed"(c Choice<Integer, FHIR.Age, List<String>, String, Interval<Date>, FHIR.dateTime>): c
define function "Sys"(c Choice<String, Integer, Boolean, Decimal>): c

define "CallMixed": "Mixed"(1)
define "CallSys": "Sys"('a')

define function "Normalize Interval"(choice Choice<FHIR.dateTime, FHIR.Period, FHIR.Timing, FHIR.instant, FHIR.string, FHIR.Age, FHIR.Range>):
  case
    when choice is FHIR.dateTime then
      Interval[FHIRHelpers.ToDateTime(choice as FHIR.dateTime), FHIRHelpers.ToDateTime(choice as FHIR.dateTime)]
    when choice is FHIR.Period then
      FHIRHelpers.ToInterval(choice as FHIR.Period)
    else null as Interval<DateTime>
  end

define "Onsets": [Condition] C return "Normalize Interval"(C.onset)
