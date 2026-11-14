// A library that includes both FluentClaims and FluentEncounters and calls
// the shared fluent names on receivers of each type. (echo-elm #13)
library FluentOverloadMain version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include FluentClaims version '1.0' called Claims
include FluentEncounters version '1.0' called Encounters
context Patient
define "Claim dates": ([Claim])."Service dates"()
define "Encounter dates": ([Encounter])."Service dates"()
define "First claim": ([Claim])."First per"(1 day)
define "First encounter": ([Encounter])."First per"(1 day)
define "First date": "Encounter dates"."First per"(1 day)
define "Filtered claims": ([Claim] C where C.status = 'active')."Service dates"()

define "Control unique fluent": ([Claim])."Claim ids"()

