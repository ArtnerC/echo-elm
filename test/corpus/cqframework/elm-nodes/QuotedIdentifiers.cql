// Regression fixture for the quoted-identifier query source found while
// verifying issues/03-cqf-parity-gaps.md.
//
// A quoted identifier used as a query source must be unquoted like every other
// reference position; it previously kept its double quotes, producing an
// ExpressionRef named "\"My Items\"" that no engine could resolve.
library QuotedIdentifiers version '1.0'

define "My Items": { 1, 2, 3 }
define "Plain": 42

define "QuotedDirectRef": "My Items"
define "UnquotedDirectRef": Plain

define "QuotedQuerySource":
  "My Items" I
    return I

define "QuotedQuerySourceWhere":
  "My Items" I
    where I > 1
    return I

define "QuotedRelationshipSource":
  "My Items" I
    with "My Items" J such that I = J
    return I

define function "My Function"(x Integer) returns Integer: x + 1
define "QuotedFunctionCall": "My Function"(1)
