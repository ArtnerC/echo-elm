library IntervalOpsTest

define IntInterval: Interval[1, 10]
define IntIntervalOpen: Interval(0, 11)
define IntIntervalHalfOpen: Interval[1, 11)

define IntervalUnion: Interval[1, 5] union Interval[3, 8]
define IntervalIntersect: Interval[1, 8] intersect Interval[3, 10]
define IntervalExcept: Interval[1, 10] except Interval[3, 7]

define PointInInterval: 5 in Interval[1, 10]
define IntervalContainsPoint: Interval[1, 10] contains 5
define IntervalOverlaps: Interval[1, 5] overlaps Interval[3, 8]
define IntervalOverlapsBefore: Interval[1, 5] overlaps before Interval[3, 8]
define IntervalOverlapsAfter: Interval[3, 8] overlaps after Interval[1, 5]
define IntervalIncludes: Interval[1, 10] includes Interval[3, 7]
define IntervalIncludedIn: Interval[3, 7] included in Interval[1, 10]
define ProperIncludes: Interval[1, 10] properly includes Interval[3, 7]
define ProperIncludedIn: Interval[3, 7] properly included in Interval[1, 10]

define IntervalStart: start of Interval[1, 10]
define IntervalEnd: end of Interval[1, 10]
define IntervalWidth: width of Interval[1, 10]

define DateInterval: Interval[@2020-01-01, @2020-12-31]
define DateIntervalBefore: @2020-06-15 before end of Interval[@2020-01-01, @2020-12-31]
define DateIntervalAfter: @2020-06-15 after start of Interval[@2020-01-01, @2020-12-31]

define MeetsBefore: Interval[1, 5] meets before Interval[6, 10]
define MeetsAfter: Interval[6, 10] meets after Interval[1, 5]
define Meets: Interval[1, 5] meets Interval[6, 10]
