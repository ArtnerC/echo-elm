library ValueSetMembership version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

valueset "Example A": 'urn:example:valueset:a'
valueset "Example B": 'urn:example:valueset:b'

context Patient

// Membership against a ValueSet-typed operand is InValueSet{code, valuesetExpression}.
define function "Coding in"(c Coding, vs System.ValueSet):
  c in vs

define function "String in"(s String, vs System.ValueSet):
  s in vs

define function "Concept in"(cc CodeableConcept, vs System.ValueSet):
  cc in vs

// A valueset used where a list of codes is expected is wrapped in ExpandValueSet.
define "Union of valuesets":
  "Example A" union "Example B"

define "List of valuesets":
  { "Example A" union "Example B" }

// Control: membership against a declared valueset matches today.
define "Concept property in":
  [Claim] C where C.type in "Example A"
