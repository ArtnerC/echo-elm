// Regression fixture for issues/03-cqf-parity-gaps.md B7 / B10 / B11 / B13.
//
// Covers the query clauses that the HEDIS comparison flagged: sort (by column,
// by expression, by direction, multi-key), aggregate (starting literal, all,
// distinct), let, and return all/distinct. Also pins the two identifier-scope
// rules: the aggregate accumulator is a QueryLetRef, and identifiers inside a
// sort-by expression stay unresolved as IdentifierRef.
library QueryClauses version '1.0'

define "Items": {
  Tuple { id: 1, score: 10, p: Interval[@2024-01-01, @2024-02-01] },
  Tuple { id: 2, score: 5, p: Interval[@2024-03-01, @2024-04-01] }
}

// ── sort ──────────────────────────────────────────────────────────────────────

define "SortByColumn":
  "Items" I
    sort by score

define "SortByColumnDesc":
  "Items" I
    sort by score desc

define "SortByDirectionOnly":
  "Items" I
    return I.score
    sort asc

define "SortByMultipleKeys":
  "Items" I
    sort by id asc, score desc

// Identifiers in a sort-by expression are scoped to the query result element,
// so CQF defers resolution to the engine via IdentifierRef.
define "SortByExpression":
  "Items" I
    sort by start of p

define "SortByThis":
  "Items" I
    return I.p
    sort by start of $this

// ── aggregate ─────────────────────────────────────────────────────────────────

// `starting` takes a simpleLiteral, a quantity, or a parenthesised expression.
define "AggregateStartingLiteral":
  "Items" I
    aggregate Total starting 0: Total + I.score

define "AggregateStartingString":
  "Items" I
    aggregate Acc starting 'x': Acc

define "AggregateStartingExpression":
  "Items" I
    aggregate Total starting (1 + 1): Total + I.score

define "AggregateAll":
  "Items" I
    aggregate all Acc starting 1: Acc * I.score

define "AggregateDistinct":
  "Items" I
    aggregate distinct Acc starting 1: Acc * I.score

// ── let / return ──────────────────────────────────────────────────────────────

define "LetClause":
  "Items" I
    let s: I.score
    where s > 1
    return s

define "ReturnDistinct":
  "Items" I
    return distinct I.id

define "ReturnAll":
  "Items" I
    return all I.id
