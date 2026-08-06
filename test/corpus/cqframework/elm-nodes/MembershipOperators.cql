// Regression fixture for issues/03-cqf-parity-gaps.md C2 / C5 / C6.
//
// Three operator-lowering rules the HEDIS comparison flagged:
//   - `in` promotes its right operand with ToList only when that operand does
//     not already satisfy an In overload for the left operand's type. A
//     reference to a list- or interval-valued define must not be promoted.
//   - `during` / `included in` select In (ProperIn) for a point left operand
//     and IncludedIn (ProperIncludedIn) for an interval one.
//   - `!=` and `!~` have no ELM operator of their own; they lower to
//     Not(Equal(...)) and Not(Equivalent(...)).
library MembershipOperators version '1.0'

define "Period": Interval[@2024-01-01, @2024-12-31]
define "Day": @2024-06-01
define "Numbers": { 1, 2, 3 }

// ── in: promotion only when the right operand does not already match ──────────

define "PointInIntervalRef": "Day" in "Period"
define "PointInListRef": 2 in "Numbers"
define "PointInListLiteral": 2 in { 1, 2, 3 }
define "PointInIntervalLiteral": @2024-06-01 in Interval[@2024-01-01, @2024-12-31]
// The left operand is itself an interval, so this resolves to In(T, List<T>)
// with T = Interval<DateTime> and the right operand IS promoted with ToList.
define "IntervalInIntervalRef": Interval[@2024-02-01, @2024-03-01] in "Period"

// ── included in / during / properly included in ───────────────────────────────

define "PointIncludedIn": "Day" included in "Period"
define "PointDuring": "Day" during "Period"
define "PointProperlyIncludedIn": "Day" properly included in "Period"
define "IntervalIncludedIn": Interval[@2024-02-01, @2024-03-01] included in "Period"
define "IntervalProperlyIncludedIn": Interval[@2024-02-01, @2024-03-01] properly included in "Period"
define "IntervalContains": "Period" contains "Day"

// ── negated equality ──────────────────────────────────────────────────────────

define "NotEqualInteger": 1 != 2
define "NotEquivalentInteger": 1 !~ 2
define "NotEqualString": 'a' != 'b'
define "EqualInteger": 1 = 2
define "EquivalentInteger": 1 ~ 2
