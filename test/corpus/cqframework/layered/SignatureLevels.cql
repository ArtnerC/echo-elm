// Signature levels on membership and conversion shapes.
//
// Differing records a signature exactly where resolution converted or cast an
// operand: `E.period.start during MP` gets one because E.period.start was
// converted, while a FHIRHelpers call whose argument already has the parameter
// type gets none. Overloads is deliberately not run here — CQF's Overloads
// decision depends on which operator instantiations the library has already
// accumulated (see the tracking GitHub issue), and this fixture would pin
// guesses about it. For the same reason `null in { 1, null }` is left out: CQF
// casts that null when FHIRHelpers is resolvable and leaves it bare when not.
library SignatureLevels version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
parameter "MP" Interval<DateTime>
parameter "D" DateTime
parameter "Ns" List<Integer>
valueset "VS": 'http://example.org/vs'
context Patient
define "Encs": [Encounter]
define "IntInList": 1 in { 1, 2 }
define "IntInRef": 2 in "Ns"
define "IntInInterval": 3 in Interval[1, 5]
define "DTInParam": "D" in "MP"
define "ListContains": { 1, 2 } contains 1
define "IntervalContains": "MP" contains "D"
define "PropDuring": "Encs" E where E.period.start during "MP"
define "PeriodIncludedIn": "Encs" E where E.period included in "MP"
define "StatusIn": "Encs" E where E.status in { 'finished' }
define "CodeInVS": "Encs" E where E.type in "VS"
define "DateInDay": "D" in day of "MP"
define "ExistsEncs": exists "Encs"
define "CountEncs": Count("Encs")
define "Union": { 1 } union { 2 }
define "IntervalOverlap": "MP" overlaps Interval[@2024-01-01T00:00:00, @2024-02-01T00:00:00]
define "StartEnc": "Encs" E return start of FHIRHelpers.ToInterval(E.period)
