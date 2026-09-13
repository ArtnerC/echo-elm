// Fluent invocation through an EXPRESSION receiver must still carry libraryName.
//
// `"N"."Doubled"()` parses as Doubled(N) — the same shape as a library-qualified
// call, but the receiver is a define, not an include alias, so the qualifier
// cannot come from the receiver. It has to be found from the fluent functions
// the included libraries declare. The include is deliberately unaliased, so the
// library name comes from the declaration. (issues/05 3.6)
library FluentCrossLibrary version '1.0'

include FluentHelpers version '1.0'

define "N": 21

define "Result": "N"."Doubled"()
