// Implicit FHIRHelpers conversions on a query whose source is a define.
//
// The alias' model type used to be read out of the source's *syntax*, so it was
// only found when a Retrieve was visible there. `"Encounters" E` has no Retrieve
// to find, so the alias carried no FHIR type and every implicit conversion
// inside the query was silently suppressed — including at the terminology and
// comparison boundaries, where an unconverted FHIR value is a different test.
// (issues/05 3.2)
library FHIRHelperConversions version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "Measurement Period" Interval<DateTime>

context Patient

define "Encounters": [Encounter]

// System-typed parameters force the conversion at each call site.
define function "Label"(s String): s
define function "Coded"(c Code): c
define function "Within"(p Interval<DateTime>): p
define function "AtTime"(d DateTime): d

define "Status Strings": "Encounters" E return "Label"(E.status)
define "Class Codes": "Encounters" E return "Coded"(E.class)
define "Period Intervals": "Encounters" E return "Within"(E.period)
define "Period Starts": "Encounters" E return "AtTime"(E.period.start)
define "Status Comparison": "Encounters" E where E.status = 'finished'
define "Period During MP": "Encounters" E where E.period during "Measurement Period"
