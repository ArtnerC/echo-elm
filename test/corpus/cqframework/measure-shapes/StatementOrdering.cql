// Negative control: dependency ordering is correct and must stay that way.
//
// Both translators emit ['Gamma', 'Alpha', 'Beta'] — neither is source order;
// both resolve dependencies first with a source-order tiebreak. Recorded so the
// ordering algorithm is not blamed for divergence that comes from resolution
// (issues/05 3.8).
library StatementOrdering version '1.0'

context Unfiltered

define "Alpha": "Gamma" + 1
define "Beta": 2
define "Gamma": 3
