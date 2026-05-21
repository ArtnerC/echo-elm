library ConversionFunctionsTest

define ToIntFromString: ToInteger('42')
define ToDecFromString: ToDecimal('3.14')
define ToStringFromInt: ToString(42)
define ToStringFromDecimal: ToString(3.14)
define ToBoolFromString: ToBoolean('true')
define ToStringFromBool: ToString(true)

define ToDateFromString: ToDate('2020-01-15')
define ToDateTimeFromString: ToDateTime('2020-01-15T10:30:00')
define ToTimeFromString: ToTime('10:30:00')

define ToStringFromDate: ToString(@2020-01-15)
define ToStringFromDateTime: ToString(@2020-01-15T10:30:00)

define ConvertToInteger: convert '42' to Integer
define ConvertToDecimal: convert '3.14' to Decimal
define ConvertToDate: convert '2020-01-15' to Date

define IsInteger: 42 is Integer
define IsDecimal: 3.14 is Decimal
define IsString: 'hello' is String
define IsBoolean: true is Boolean

define AsInteger: 42 as Integer
