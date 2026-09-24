// Parameter defaults keep their locator. Only a default CQF rewrites — an
// Interval<DateTime> written with Date bounds, promoted with ToDateTime — is
// emitted without one. (issues/07 family 2)
library ParameterDefaults version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
parameter "MP" Interval<DateTime> default Interval[@2024-01-01T00:00:00.0, @2025-01-01T00:00:00.0)
parameter "MPDate" Interval<DateTime> default Interval[@2024-01-01, @2024-12-31]
parameter "Threshold" Integer default 5
parameter "Neg" Decimal default -1.5
parameter "Name" String default 'x' + 'y'
parameter "Codes" List<String> default { 'a', 'b' }
parameter "Tup" Tuple { a Integer } default Tuple { a: 1 }
parameter "NoDefault" Interval<DateTime>
parameter "Computed" DateTime default DateTime(2024, 1, 1)
context Patient
define "Uses": "Threshold" + 1
