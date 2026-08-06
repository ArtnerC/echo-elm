// Regression fixture for block-comment @tag annotations.
//
// CQF has no allowlist of spec-defined tag names: it emits any tag whose name
// is a valid CQL identifier and silently drops the rest. So @author and
// @customThing produce tag data while @measure-component and @dotted.name do not.
library AnnotationTags version '1.0'

/*
@author: Chopin
@description: Every tag name here is a valid identifier
@comment: C
@seealso: S
@deprecated: X
@notes: N
@jurisdiction: J
@region: R
@terminologyKey: T
@customThing: CT
@with_underscore: U
@with2digits: D2
@UPPER: UP
@_leading: L
*/
define "ValidTagNames": true

/*
@measure-component: dropped
@dotted.name: dropped
@9leading: dropped
@description: kept
*/
define "InvalidTagNames": true

/*
@description: Determines the cumulative duration of a list of intervals
@comment: #This function collapses the input intervals prior to determining the cumulative duration
to ensure overlapping intervals do not contribute multiple times to the result#
 */
define function "CumulativeDuration"(Intervals List<Interval<DateTime>>):
  Sum((collapse Intervals) X return all duration in days of X)
