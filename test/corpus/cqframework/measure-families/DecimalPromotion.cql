library DecimalPromotion version '1.0'

using FHIR version '4.0.1'

context Patient

// An Integer in a Decimal position is wrapped in ToDecimal.
define function "Weight"(b Boolean):
  if b then 1.5 else 0

define function "Takes decimal"(d Decimal):
  d + 1.0

define "Call with integer":
  "Takes decimal"(1)

define "Max mixed":
  Max({ 1.5, 2 })

// Control: Decimal on both sides.
define "Sum of weights":
  "Weight"(true) + "Weight"(false)
