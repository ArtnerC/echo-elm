// The implicit context accessor is emitted at the source position of the
// `context` declaration, not prepended.
//
// CQF walks declarations in source order and creates the accessor when it
// reaches the declaration, so a define written above `context Patient` comes out
// above the Patient accessor. (issues/05 3.7)
library ContextAccessorPosition version '1.0'

using FHIR version '4.0.1'

define "Declared Before Context": 1 + 1

context Patient

define "Declared After Context": 2 + 2
