library QueryExpressionsTest

define Tuples: { Tuple{a: 1, b: 2}, Tuple{a: 3, b: 4}, Tuple{a: 5, b: 6} }

define SimpleQuery:
  (Tuples) T
    where T.a > 2

define ProjectionQuery:
  (Tuples) T
    return T.a

define TupleProjection:
  (Tuples) T
    return Tuple { x: T.a, y: T.b }

define FilterAndProject:
  (Tuples) T
    where T.a >= 3
    return T.b

define SortedQuery:
  (Tuples) T
    sort by a ascending

define SortedQueryDesc:
  (Tuples) T
    sort by a descending

define WithQuery:
  (Tuples) T
    with (Tuples) S such that S.a = T.b

define WithoutQuery:
  (Tuples) T
    without (Tuples) S such that S.a = T.b

define AggregateQuery:
  Count((Tuples) T where T.a > 2)

define LetQuery:
  ({ 1, 2, 3 }) N
    let doubled: N * 2
    return doubled

define MultiSource:
  from
    ({1, 2}) A,
    ({10, 20}) B
  return A + B
