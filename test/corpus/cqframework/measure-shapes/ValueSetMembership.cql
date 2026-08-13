// Value set membership converts its code operand.
//
// The same defect as FHIRHelperConversions at the terminology boundary, and the
// one with the clearest clinical consequence: membership evaluated against an
// unconverted FHIR CodeableConcept is not the same test as against a Concept.
// (issues/05 3.4)
library ValueSetMembership version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

valueset "Diabetes": 'http://example.org/fhir/ValueSet/diabetes'

context Patient

define "Conditions": [Condition]

define "Diabetic Conditions": "Conditions" C where C.code in "Diabetes"
