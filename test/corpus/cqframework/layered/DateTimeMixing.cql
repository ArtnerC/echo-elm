// A Date meeting a DateTime converts to DateTime: as a point tested against an
// Interval<DateTime>, in a list, across if/else branches, and as an interval's
// bounds. AgeInYearsAt a Date uses the Date overload and leaves the birth date
// unconverted — the stray ToDateTime of issues/07 family 3.
library DateTimeMixing version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
parameter "MP" Interval<DateTime>
parameter "D" Date
parameter "DT" DateTime
context Patient
define "Encs": [Encounter]
define "DateVsDateTime": "D" < "DT"
define "DateTimeVsDate": "DT" = "D"
define "DateInDTInterval": "D" in "MP"
define "AgeAt": AgeInYearsAt(start of "MP")
define "AgeAtDate": AgeInYearsAt("D")
define "DiffDates": years between "D" and Today()
define "ToDTCall": ToDateTime(Patient.birthDate)
define "DateIntervalOverlaps": Interval["D", "D" + 1 day] overlaps "MP"
define "CoalesceDates": Coalesce(Patient.birthDate, "D")
define "MaxDateTime": Max({ "DT", "D" })
define "DateAdd": "D" + 1 year
define "DateFromDT": date from "DT" < "D"
define "IfDate": if true then "D" else "DT"
