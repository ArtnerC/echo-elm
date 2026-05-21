library ListOperationsTest

define MyList: {1, 2, 3, 4, 5}
define TypedList: List<Integer>{1, 2, 3, 4, 5}

define ListFirst: First({3, 1, 4, 1, 5})
define ListLast: Last({3, 1, 4, 1, 5})
define ListCount: Count({1, 2, 3, 4, 5})
define ListDistinct: distinct {1, 2, 2, 3, 3, 3}
define ListFlatten: flatten {{1, 2}, {3, 4}, {5}}
define ListExists: exists {1, 2, 3}
define ListExistsEmpty: exists {}
define ListNotEmpty: not (exists {})

define ListTail: Tail({1, 2, 3, 4, 5})
define ListSkip: Skip({1, 2, 3, 4, 5}, 2)
define ListTake: Take({1, 2, 3, 4, 5}, 3)
define ListIndexOf: IndexOf({10, 20, 30}, 20)
define ListContains: {1, 2, 3} contains 2

define SortedAsc: (List<Integer>{3, 1, 4, 1, 5}) L sort asc
define SortedDesc: (List<Integer>{3, 1, 4, 1, 5}) L sort desc

define ListUnion: {1, 2, 3} union {3, 4, 5}
define ListIntersect: {1, 2, 3, 4} intersect {2, 4, 6}
define ListExcept: {1, 2, 3, 4} except {2, 4}

define AllTrue: AllTrue({true, true, true})
define AnyTrue: AnyTrue({false, true, false})

define MinList: Min({5, 2, 8, 1, 9})
define MaxList: Max({5, 2, 8, 1, 9})
define SumList: Sum({1, 2, 3, 4, 5})
define AvgList: Avg({2.0, 4.0, 6.0})
define MedianList: Median({1, 2, 3, 4, 5})
