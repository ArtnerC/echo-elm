// Overloads of one fluent function name split across two includes; the
// receiver's type selects the library. (echo-elm #13)
library FluentClaims version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
context Patient
define fluent function "Service dates"(claims List<Claim>):
  claims C return C.created
define fluent function "First per"(claims List<Claim>, gap System.Quantity):
  First(claims C sort by created)
define fluent function "Claim ids"(claims List<Claim>):
  claims C return C.id

