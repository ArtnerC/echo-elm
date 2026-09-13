// Where an implicit FHIRHelpers conversion is and is not inserted.
//
// CQF converts a FHIR value where it is consumed by a typed operator, and
// nowhere else. echo-elm had drifted both ways (issues/06):
//   - a `with` alias over a define never recorded its model type, so properties
//     navigated off it were not converted while the source alias's were;
//   - a consumed cast to a FHIR type (`O.effective as FHIR.dateTime`) and a
//     consumed alias over a convertible FHIR type (`T in "VS"`) were not
//     converted at all;
//   - selector bounds that are both properties, and the argument of an explicit
//     FHIRHelpers call, were converted when CQF leaves them alone.
library FHIRConversionSites version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

valueset "VS": 'http://example.org/fhir/ValueSet/vs'

parameter "MP" Interval<DateTime>

context Patient

define "Encs": [Encounter]

define "IntervalBounds": "Encs" E return Interval[E.period.start, E.period.end]
define "WithPeriod": "Encs" E with "Encs" E2 such that E2.period starts during E.period
define "WithCast": "Encs" E with [Condition] C such that (C.onset as FHIR.dateTime) during E.period
define "ObsValue": [Observation] O where (O.value as FHIR.Quantity) > 5 'mg'
define "ObsEffective": [Observation] O where (O.effective as FHIR.dateTime) during "MP"
define "ProcPerformed": [Procedure] P where (P.performed as FHIR.Period) during "MP"
define "AliasTypeIn": "Encs" E where exists ((E.type) T where T in "VS")
define "ExplicitToInterval": "Encs" E where start of FHIRHelpers.ToInterval(E.period) in "MP"
