// Library-qualified function calls must carry libraryName.
//
// `H."Doubled"(21)` parses identically to a method invocation — the grammar
// cannot tell a library qualifier from a receiver — so the qualifier has to be
// recovered from the include aliases. Without it the call emits a FunctionRef
// naming a function in the current library, which is a different reference:
// here "Doubled" exists locally too, so the unqualified form silently resolves
// to the wrong function rather than failing.
library CrossLibraryFunction version '1.0'

include SharedFunctions version '1.0' called H

define "QualifiedCall": H."Doubled"(21)
define "QualifiedNoArgs": H."Constant"()
define "LocalCall": "Doubled"(21)
define function "Doubled"(n Integer): n * 100
