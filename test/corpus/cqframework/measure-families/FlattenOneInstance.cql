library FlattenOneInstance version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

define "Integer lists":
  flatten({ {1, 2}, {3} })

define "Mixed lists":
  flatten({ {1, 2}, {'a'} })
