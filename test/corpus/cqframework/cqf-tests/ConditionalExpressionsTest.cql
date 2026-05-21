library ConditionalExpressionsTest

define IfThenElse: if true then 1 else 0
define IfNull: if null then 'was null' else 'not null'
define NestedIf: if 5 > 10 then 'big' else if 5 > 3 then 'medium' else 'small'

define CaseSimple:
  case
    when 1 + 1 = 2 then 'correct'
    when 1 + 1 = 3 then 'wrong'
    else 'unknown'
  end

define CaseComparand:
  case 3
    when 1 then 'one'
    when 2 then 'two'
    when 3 then 'three'
    else 'other'
  end

define IsNullExpr: null is null
define IsNotNullExpr: 5 is not null
define IsNullFunc: IsNull(null)
define IsNotNullFunc: not IsNull(5)

define CoalesceExpr: Coalesce(null, null, 42)
define IfNullExpr: if null is null then 0 else 1
