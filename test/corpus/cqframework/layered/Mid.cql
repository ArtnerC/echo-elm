library Mid version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include Common version '1.0' called Common
parameter "MP" Interval<DateTime>
context Patient
define "Onsets": Common."Active" C return C.onset.toInterval()
define "ActiveInMP": Common."Active" C where Common."Normalize"(C.onset) overlaps "MP"
define function "Within"(c FHIR.Condition, p Interval<DateTime>): c.onset.toInterval() during p
