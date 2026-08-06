// Regression fixture for cross-library terminology resolution.
//
// Every HEDIS measure library pulls its value sets and codes from a shared
// terminology library, so a library-qualified reference must resolve to the
// typed node (ValueSetRef / CodeRef / CodeSystemRef with libraryName) rather
// than degrading to a Property access on an ExpressionRef. Resolving this
// requires the include to be loaded, which is why the translate CLI searches
// the input file's own directory.
library CrossLibraryTerminology version '1.0'

include SharedTerminology version '1.0' called T

define "QualifiedCodeSystem": T."SNOMED"
define "QualifiedCode": T."Diabetes"
define "QualifiedValueSet": T."Diabetes VS"
define "QualifiedExpression": T."SharedExpression"

define "QualifiedCodeInQualifiedValueSet": T."Diabetes" in T."Diabetes VS"
define "StringInQualifiedValueSet": 'some-code' in T."Diabetes VS"
