// `within`, trailing boundary keywords, point membership, and point promotion.
//
//   - `within q of Y` is the window [Y - q, Y + q]; its quantity was dropped
//     entirely, so X was compared against Y itself.
//   - A trailing `start` / `end` keyword selects a boundary of the right operand
//     and is attributed to that keyword.
//   - `includes` against a point is Contains, the mirror of `included in`
//     becoming In.
//   - Comparing an interval with a point promotes the point — on whichever side
//     it is — to `if X is null then null else Interval[X, X]`.
// (issues/06 follow-up)
library TimingWithinAndPromotion version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "MP" Interval<DateTime>
parameter "D" DateTime
parameter "D2" DateTime

context Patient

define "Encs": [Encounter]

define "Within": "D" within 3 days of "D2"
define "ProperlyWithin": "D" properly within 3 days of "D2"
define "WithinYears": "D" within 1 year of "D2"
define "StartsWithin": "Encs" E where E.period starts within 3 days of start of "MP"
define "EndsWithinOfEnd": "Encs" E where E.period ends within 3 days of end "MP"
define "ProperlyStartsWithin": "Encs" E where E.period starts properly within 3 days of start of "MP"
define "WithinStartKeyword": "D" within 3 days of start "MP"

define "BeforeStartKeyword": "D" before start "MP"
define "EndsBeforeStartKeyword": "Encs" E where E.period ends before start "MP"
define "SameDayAsStartKeyword": "D" same day as start "MP"
define "IncludesStartKeyword": "MP" includes start "MP"

define "IntervalBeforePoint": "Encs" E where E.period before "D"
define "IntervalOnOrBefore": "Encs" E where E.period on or before end of "MP"
define "PointBeforeInterval": "D" before "MP"
define "PointOnOrBeforeInterval": "D" on or before "MP"
define "IntervalOnOrAfterPoint": "MP" on or after "D"
define "IntervalBeforeDayOfPoint": "MP" before day of "D"
