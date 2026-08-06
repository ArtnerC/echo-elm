// Regression fixture for issues/03-cqf-parity-gaps.md B8 / C4.
//
// `date from`, `time from` and `timezoneoffset from` are their own ELM unary
// operators (DateFrom / TimeFrom / TimezoneOffsetFrom). Only the calendar and
// clock components use DateTimeComponentFrom, and that node is a UnaryExpression
// so its "operand" is a single object rather than an array.
library DateTimeComponents version '1.0'

define "DateFromDateTime": date from @2025-01-15T10:30:00
define "TimeFromDateTime": time from @2025-01-15T10:30:00
define "TimezoneOffsetFromDateTime": timezoneoffset from @2025-01-15T10:30:00

define "YearComponent": year from @2025-01-15T10:30:00
define "MonthComponent": month from @2025-01-15T10:30:00
define "DayComponent": day from @2025-01-15T10:30:00
define "HourComponent": hour from @2025-01-15T10:30:00
define "MinuteComponent": minute from @2025-01-15T10:30:00
define "SecondComponent": second from @2025-01-15T10:30:00
define "MillisecondComponent": millisecond from @2025-01-15T10:30:00

define "DateFromDate": date from @2025-01-15
