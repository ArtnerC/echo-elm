// .value on FHIR primitive elements: CQF types the Property with the
// primitive's System type; the type must flow into enclosing nodes.
library PrimitiveValue version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define function "Status text"(s FHIR.ClaimStatus): s.value
define function "Created"(d FHIR.dateTime): d.value
define function "Flag"(b FHIR.boolean): b.value
define function "Amount"(q FHIR.Quantity): q.value.value
define "Created period": [Claim] C return Interval[C.billablePeriod.start.value, C.billablePeriod."end".value]
