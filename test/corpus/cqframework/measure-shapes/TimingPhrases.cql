// Timing phrases: inclusive relationships, boundary selectors, quantity offsets.
//
// Three parts of a timing phrase change what is compared, and all three were
// being dropped (issues/06):
//   - `on or` / `or on` make the comparison inclusive. The grammar spells them as
//     single tokens containing a space, and a check for "onor" never matched, so
//     `on or before` became a strict Before.
//   - `starts` / `ends` select a boundary of the left operand, and a trailing
//     `start` / `end` one of the right. Ignored, `E.period ends during P`
//     compared the whole period instead of its end.
//   - A quantity offset — `3 days or less before` — shifts and bounds the
//     comparison. Ignored, it compared against the anchor itself.
library TimingPhrases version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "MP" Interval<DateTime>
parameter "D" DateTime
parameter "D2" DateTime

context Patient

define "Encs": [Encounter]

// Inclusive relationships.
define "OnOrBefore": "Encs" E where E.period.start on or before end of "MP"
define "BeforeOrOn": "Encs" E where E.period.start before or on end of "MP"
define "OnOrBeforeDay": "Encs" E where E.period.start on or before day of end of "MP"
define "StartsOnOrAfterDay": "Encs" E where E.period starts on or after day of start of "MP"

// Boundary selectors.
define "EndsOnOrBefore": "Encs" E where E.period ends on or before end of "MP"
define "StartsDuring": "Encs" E where E.period starts during "MP"
define "EndsDuringDay": "Encs" E where E.period ends during day of "MP"
define "StartOfDuring": "Encs" E where start of E.period during "MP"
define "StartOfDuringDay": "Encs" E where start of E.period during day of "MP"

// Quantity offsets.
define "ExactBefore": "D" 3 days before "D2"
define "OrMoreBefore": "D" 3 days or more before "D2"
define "MoreThanAfter": "D" more than 3 days after "D2"
define "LessThanBefore": "D" less than 3 days before "D2"
define "OrLessBefore": "D" 3 days or less before "D2"
define "OrLessOnOrBefore": "D" 3 days or less on or before "D2"
define "OrLessAfter": "D" 3 days or less after "D2"
define "EndsOrLessOnOrBeforeEnd": "Encs" E where E.period ends 1 day or less on or before end of "MP"
define "StartsOrLessOnOrAfterDay": "Encs" E where E.period starts 30 days or less on or after day of start of "MP"
