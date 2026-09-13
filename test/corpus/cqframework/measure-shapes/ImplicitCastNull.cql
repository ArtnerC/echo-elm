// A bare `null` at a typed parameter is wrapped in an As naming that type.
//
// `null` carries no type of its own, so without the wrapper the argument is an
// untyped Null and the call's overload cannot be recovered from the ELM.
// (issues/05 3.3)
library ImplicitCastNull version '1.0'

context Unfiltered

define function "Window"(period Interval<Date>, dates List<Date>): period
define function "Named"(s String): s

define "Null Interval Argument": "Window"(null, { @2024-01-01 })
define "Null List Argument": "Window"(Interval[@2024-01-01, @2024-12-31], null)
define "Null Named Argument": "Named"(null)
