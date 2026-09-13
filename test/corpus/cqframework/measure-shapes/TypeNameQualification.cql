// One type, one rendering: {modelUri}LocalName.
//
// echo-elm used to emit three spellings of the same type in a single library —
// `{http://hl7.org/fhir}Encounter` from a retrieve, `FHIR.Encounter` from an
// alias-qualified declaration, and bare `Encounter` from an unqualified one.
// Retrieves were always right; declared type specifiers kept the source text.
// (issues/05 3.1)

library TypeNameQualification version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

valueset "Office Visit": 'http://example.org/fhir/ValueSet/office-visit'

context Patient

define "All Encounters": [Encounter]
define "Encounters In Value Set": [Encounter: "Office Visit"]

// Alias-qualified in the declaration; must still render as {uri}Name.
define function "CountOf"(encounters List<FHIR.Encounter>): Count(encounters)

define function "First Condition"(conditions List<FHIR.Condition>):
  singleton from (conditions C where C.id is not null)

define "Encounter Count": "CountOf"("All Encounters")
define "Has Any Encounter": exists "All Encounters"
define "Value Set Reference": "Office Visit"
