// Helpers whose return type is Any by construction, for AnyBranchCast.
library AnyBranchHelpers version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

define function "Raise"(msg String):
  Message(null, true, 'AnyBranchHelpers.Error', 'Error', msg)

define function "Declared any"(msg String) returns Any:
  Message(null, true, 'AnyBranchHelpers.Declared', 'Error', msg)

define function "Typed"(amount Decimal): Truncate(amount)
