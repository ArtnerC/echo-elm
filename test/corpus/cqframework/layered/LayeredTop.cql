// Layered includes: LayeredTop -> Mid -> Common, all resolvable.
//
//   - Types propagate through a second include level. The included library's
//     own includes used to be left unresolved when its types were computed, so
//     Mid's definitions came out List<Any> here. (issues/07 family 1)
//   - A choice-typed argument is narrowed to the declared choice parameter:
//     As(C.onset, Choice<Period, dateTime>).
//   - `&` is null-tolerant: each operand is wrapped in Coalesce(x, '').
//   - `+` between String-valued expressions is Concatenate, decided by type.
library LayeredTop version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
include Mid version '1.0' called Mid
include Common version '1.0' called Common
parameter "MP" Interval<DateTime> default Interval[@2024-01-01T00:00:00.0, @2025-01-01T00:00:00.0)
context Patient
define "TopOnsets": Mid."Onsets"
define "TopActive": Mid."ActiveInMP" A where Mid."Within"(A, "MP")
define "TopDirect": Common."Active" C return Common."Normalize"(C.onset)
define "TopFluent": Common."Active" C return C.onset.toInterval()
define "StartOfFirst": start of First(Mid."Onsets")
define "Concat": 'a' + 'b'
define "ConcatAmp": 'a' & 'b'
define "ConcatRef": "Label" + 'x'
define "Label": 'lbl'
