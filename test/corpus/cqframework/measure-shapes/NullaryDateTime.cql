// Now, Today and TimeOfDay are ELM operators with no operands.
//
// They were emitted as a FunctionRef, which names a library function that does
// not exist — no engine can evaluate it. (issues/06)
library NullaryDateTime version '1.0'

context Unfiltered

define "N": Now()
define "T": Today()
define "TOD": TimeOfDay()
define "NowCompared": Now() > @2024-01-01T00:00:00
define "TodayInCase": case when true then Today() else null end
