// Overloads of one fluent function name split across two includes; the
// receiver's type selects the library. (echo-elm #13)
library FluentEncounters version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define fluent function "Service dates"(encs List<Encounter>):
  encs E return start of E.period
define fluent function "First per"(encs List<Encounter>, gap System.Quantity):
  First(encs E sort by start of period)
define fluent function "First per"(dates List<DateTime>, gap System.Quantity):
  First(dates D sort asc)
