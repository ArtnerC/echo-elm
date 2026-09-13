// A null takes the type its context establishes, wrapped in As.
//
// The type used to be read from the other operand's syntax, which only
// recognised literals — so a structured branch (Interval<DateTime>), a call
// (Now()), or any binary operator left the null bare. And inside an Interval
// bound the null itself was stamped with the bound's type, where CQF leaves the
// null Any and lets the As do the narrowing. (issues/06 4.1)
library NullCasts version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "MP" Interval<DateTime>

context Patient

define "IfThenNull": if true then null else 1
define "IfThenNullCall": if true then null else Now()
define "CaseThenNull": case when true then null else 5 end
define "CaseElseNullInterval": case when true then "MP" else null end
define "EqualsNull": 1 = null
define "AddNull": 1 + null
define "CoalesceLeadingNull": Coalesce(null, 1)
define "CoalesceTrailingNull": Coalesce(Now(), null)
define "IntervalLowNull": Interval[null, @2024-01-01T00:00:00]
