// Single-hop references into an included library: definitions, functions,
// fluent calls, tuples and nested queries keep their types and libraryName,
// and `X is null` tests a FHIR value unconverted — IsNull takes Any, so there
// is nothing to convert it to. (issues/07)
library LayeredMain version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include Helpers version '1.0' called H
parameter "MP" Interval<DateTime>
context Patient
define "RefList": H."Encs"
define "RefQuery": H."Finished" F where F.class is not null
define "RefStarts": H."Starts"
define "RefFirst": H."FirstEnc".period
define "RefPeriods": H."Periods" P where P overlaps "MP"
define "RefTuple": H."Pair".enc.status
define "CallList": H."EncsIn"("MP")
define "CallScalar": H."StartOf"(First(H."Encs"))
define "CallUntyped": H."Untyped"(First(H."Encs"))
define "Fluent": [Condition] C return C.prevalence()
define "FluentList": H."Encs".earliest()
define "Nested": (H."EncsIn"("MP")) E return E.period.start
define "Exists": exists H."Finished"
define "CountRef": Count(H."Encs")
