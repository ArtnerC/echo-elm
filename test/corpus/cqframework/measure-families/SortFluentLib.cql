// Declares a fluent function used as an implicit-receiver sort key by
// SortImplicitReceiver.cql.
library SortFluentLib version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define fluent function "Effective interval"(obs Observation):
  FHIRHelpers.ToInterval(obs.effective as FHIR.Period)
