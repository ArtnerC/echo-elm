// A fluent function called with no explicit receiver inside `sort by`. CQF
// binds the sort's implicit receiver: FunctionRef{libraryName, operand: [IdentifierRef $this]}.
library SortImplicitReceiver version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include SortFluentLib version '1.0' called Obs
context Patient
define "Implicit receiver": [Observation] O sort by start of "Effective interval"()
define "Explicit this receiver": [Observation] O sort by start of $this."Effective interval"()
