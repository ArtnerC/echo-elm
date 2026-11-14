// A function declared external has no body; CQF marks the FunctionDef
// external and omits the expression.
library ExternalFunction version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define function "Lookup"(key System.String) returns System.String : external
define function "Resolve"(ref FHIR.Reference) returns FHIR.Resource : external
define "Looked up": "Lookup"('k')
