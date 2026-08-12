// `same or before` / `same or after` are their own ELM operators.
//
// The grammar spells all three concurrent-with forms as
// `same <precision>? (or before | or after | as)`, so only the relativeQualifier
// separates them. Collapsing them to SameAs changes the truth conditions rather
// than the shape: `D same or before X` holds for every D at or before X, while
// SameAs holds only on equality. (issues/05 3.5)
library TemporalOperatorSelection version '1.0'

context Unfiltered

define "MP": Interval[@2024-01-01, @2024-12-31]
define "D": @2024-06-15
define "Sub": Interval[@2024-03-01, @2024-04-01]

define "Point In Interval": "D" in "MP"
define "Interval In Interval": "Sub" included in "MP"
define "Interval Contains Point": "MP" contains "D"
define "Interval Includes Interval": "MP" includes "Sub"
define "Same Or Before": "D" same or before end of "MP"
define "Same Or After": "D" same or after start of "MP"
define "Same As": "D" same as start of "MP"
define "Strictly Before": "D" before end of "MP"
