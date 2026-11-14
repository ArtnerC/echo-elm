// Sort a FHIR resource query by a FHIR-typed element. CQF records the
// element's FHIR type on the sort item under --result-types.
library SortByFhirField version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define "By FHIR dateTime": [Claim] C sort by created
define "By FHIR code": [Claim] C sort by status
define "By System value": [Claim] C sort by created.value
define "By expression": [Claim] C sort by (FHIRHelpers.ToDateTime(created)) desc
