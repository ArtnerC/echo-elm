library Helpers version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
parameter "MP" Interval<DateTime>
context Patient
define "Encs": [Encounter]
define "Finished": "Encs" E where E.status = 'finished'
define "Starts": "Encs" E return E.period.start
define "FirstEnc": First("Encs")
define "Periods": "Encs" E return FHIRHelpers.ToInterval(E.period)
define "Pair": Tuple { enc: First("Encs"), n: Count("Encs") }
define function "EncsIn"(p Interval<DateTime>): "Encs" E where E.period during p
define function "StartOf"(e FHIR.Encounter): e.period.start
define function "Untyped"(e FHIR.Encounter): e.period
define fluent function "prevalence"(c FHIR.Condition): FHIRHelpers.ToInterval(c.onset as FHIR.Period)
define fluent function "earliest"(encs List<FHIR.Encounter>): First(encs E sort by start of period)
