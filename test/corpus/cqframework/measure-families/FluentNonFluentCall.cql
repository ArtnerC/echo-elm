// Negative: a fluent function invoked with ordinary (non-fluent) call
// syntax. CQF 5.0.0 rejects this. (echo-elm #8)
library FluentNonFluentCall version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include FluentClaims version '1.0' called Claims
context Patient
define "Qualified call": Claims."Claim ids"([Claim])
