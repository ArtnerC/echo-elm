// Function operands typed with nested (backbone) FHIR types, e.g.
// FHIR.Claim.Item. CQF keeps the qualified name {http://hl7.org/fhir}Claim.Item;
// the type must not be shortened to its last segment.
library BackboneOperand version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define function "Item sequence"(item FHIR.Claim.Item): item.sequence
define function "Item sequence"(item FHIR.ExplanationOfBenefit.Item): item.sequence
define function "Is first"(dx FHIR.Claim.Diagnosis): dx.sequence = 1
define function "All items"(items List<FHIR.Claim.Item>): items I return I.sequence
define "Claim item sequences": [Claim] C return C.item I return "Item sequence"(I)
define "EOB item sequences": [ExplanationOfBenefit] E return E.item I return "Item sequence"(I)
