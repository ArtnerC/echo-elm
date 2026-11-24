// Statement order for forward references. CQF adds a function to the library
// before translating its body, but an expression definition after, so a chain
// of functions is emitted caller first while a chain of defines is emitted
// callee first. (issues/08 appendix: definitions in a different order)
//
// Operand names are distinct on purpose: a function compiled lazily inside
// another's body sees the caller's operands, and a repeated name draws CQF's
// identifier-hiding warning (#8).
library FunctionOrder version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

define "Top": "Step one"(1)

define function "Step one"(a Integer): "Step two"(a)

define function "Step two"(b Integer): "Step three"(b)

define function "Step three"(c Integer): c + 1

define "Base list": { 'a', 'b' }

define "Reported": "Mapped" union { 'z' }

define "Mapped": "Map outer"("Base list")

define function "Map outer"(items List<String>): "Map inner"(items)

define function "Map inner"(values List<String>): values

define "Define chain": "Define middle"

define "Define middle": "Define bottom"

define "Define bottom": 3
