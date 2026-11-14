// A query source or with/without source written as an unparenthesized
// dotted name whose first part is an alias, operand or let binding, not a
// library. CQF resolves it as a Property; it must not become an
// ExpressionRef with libraryName set to the alias. (echo-elm #7)
library QualifiedQuerySource version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define "With over alias field":
  [Claim] C
    where exists (C.item I with C.careTeam T such that T.sequence = I.sequence)
define "Nested source over alias field":
  [Claim] C
    return Count(C.item I return I.sequence)
define "Unparenthesized alias source":
  [Claim] C
    where exists (from C.item I where I.sequence > 1)
define function "Operand source"(ref FHIR.Reference):
  [Encounter] E with ref.identifier Id such that Id.value = E.id
define function "With over operand field"(claim FHIR.Claim):
  claim.item I with claim.careTeam T such that T.sequence = I.sequence
define "Let source":
  [Claim] C
    let Items: C.item
    return exists (from Items I where I.sequence > 1)
