// Implicit list traversal: navigating `.p` on a list navigates into each
// element — `Patient.name.family` is a query over Patient.name, flattened when
// the property is itself a list (`Patient.name.given`). echo-elm took the
// property of the list itself, a different value. Traversal is on by default in
// CQF and in echo-elm's modern translate, and off under the measure profile.
library ListTraversal version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define "Addr": Patient.address
define "FirstLine": First(Patient.address.line)
define "FirstLineDirect": First(Patient.address[0].line)
define "Given": Patient.name.given
define "Family": Patient.name.family
define "GivenFirst": First(Patient.name.given)
define "TelecomValues": Patient.telecom.value
define "IdentSystems": Patient.identifier.system
define "CodingCodes": [Condition] C return C.code.coding.code
define "ExistsLine": exists Patient.address.line
define "CountNames": Count(Patient.name)
