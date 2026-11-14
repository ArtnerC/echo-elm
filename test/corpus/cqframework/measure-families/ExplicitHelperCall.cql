// Explicit FHIRHelpers call and .value navigation on a FHIR element, in
// sort and non-sort positions.
library ExplicitHelperCall version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define "Return explicit": [Claim] C return FHIRHelpers.ToDateTime(C.created)
define "Where explicit": [Claim] C where FHIRHelpers.ToDateTime(C.created) is not null
define "Return value": [Claim] C return C.created.value
define "Sort explicit": [Claim] C sort by (FHIRHelpers.ToDateTime(created))
define "Sort value": [Claim] C sort by created.value
define "Sort implicit": [Claim] C sort by created
