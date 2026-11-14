// A list of codes is tested with AnyInValueSet — true when any element is in the
// value set — over its elements converted with FHIRHelpers. Condition.category
// is List<CodeableConcept>; InValueSet over it tested the list itself. This
// applies with list traversal off too, i.e. under the measure profile.
library AnyInValueSet version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
valueset "VS": 'http://example.org/vs'
codesystem "SYS": 'http://example.org/cs'
code "C1": 'c1' from "SYS"
context Patient
define "CatIn": [Condition] C where C.category in "VS"
define "CodeIn": [Condition] C where C.code in "VS"
define "CodingIn": [Condition] C where C.code.coding in "VS"
define "EncTypeIn": [Encounter] E where E.type in "VS"
define "ObsCatIn": [Observation] O where O.category in "VS"
define "CodesList": { "C1" } in "VS"
define "Reasons": [Encounter] E where E.reasonCode in "VS"
