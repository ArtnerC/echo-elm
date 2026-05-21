library StringFunctionsTest

define StringLen: Length('Hello World')
define StringUpper: Upper('hello')
define StringLower: Lower('HELLO')
define StringSubstring: Substring('Hello World', 6)
define StringSubstringLen: Substring('Hello World', 6, 5)
define StringIndexOf: IndexOf('Hello World', 'World')
define StringStartsWith: StartsWith('Hello World', 'Hello')
define StringEndsWith: EndsWith('Hello World', 'World')
define StringContains: Contains('Hello World', 'lo W')
define StringMatches: Matches('Hello123', '[A-Za-z]+[0-9]+')
define StringConcat: 'Hello' + ' ' + 'World'
define CombineList: Combine({'a', 'b', 'c'})
define CombineWithSep: Combine({'a', 'b', 'c'}, ', ')
define SplitStr: Split('a,b,c', ',')
define ReplaceStr: ReplaceMatches('Hello World', 'World', 'CQL')
define PositionOf: PositionOf('l', 'Hello World')
define LastPositionOf: LastPositionOf('l', 'Hello World')
