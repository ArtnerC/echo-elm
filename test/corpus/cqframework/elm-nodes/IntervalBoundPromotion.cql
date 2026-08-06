// Regression fixture for the implicit Date→DateTime conversion of interval bounds.
//
// Comparing a DateTime against a Date-bounded interval literal converts each bound
// rather than the interval as a whole.
//
// This fixture is version-divergent: CQF 3.29.0 lowers the promotion through
// Property(low/high/lowClosed/highClosed) accessors over the source interval plus
// lowClosedExpression/highClosedExpression, while 4.8.0 folds the converted bounds
// directly into the promoted Interval. The 4.8.0 form is the golden.
library IntervalBoundPromotion version '1.0'

define "DateTimeInDateInterval":
  @2020-06-01T10:00:00 in Interval[@2020-01-01, @2020-12-31]

define "DateTimeDuringDateInterval":
  @2020-06-01T10:00:00 during Interval[@2020-01-01, @2020-12-31]
