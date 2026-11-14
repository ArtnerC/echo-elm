library ImplicitConversions version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

codesystem "ExampleSystem": 'urn:example:codesystem'

valueset "Example A": 'urn:example:valueset:a'

context Patient

// A FHIR primitive compared with a System value is converted: Equal(ToString(C.id), 'example').
define "Claim ids equal":
  [Claim] C where C.id = 'example'

// A Coding alias compared with a Code or tested against a valueset is converted with ToCode.
// The sources are parenthesized so #7 (QualifiedQuerySource) does not interfere.
define "Coding alias equal":
  [Claim] C where exists ((C.type.coding) X where X ~ Code 'x' from "ExampleSystem")

define "Coding alias in":
  [Claim] C where exists ((C.type.coding) X where X in "Example A")

// Tuple elements keep the FHIR type; no conversion is applied.
define "Claim tuple":
  [Claim] C return { id: C.id, created: C.created, total: C.total.value }
