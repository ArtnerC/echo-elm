library ParameterTags version '1.0.0'

using FHIR version '4.0.1'

/*
@description: Synthetic parameter with documentation tags
@uri: http://example.org/parameters/period
*/
parameter "Report period" Interval<DateTime>
  default Interval[@2025-01-01T00:00:00.0, @2026-01-01T00:00:00.0)

/*
@min: 1
@max: 10
*/
parameter "Lookback count" Integer default 3

/*
@description: Synthetic statement with a documentation tag
*/
define "Tagged statement":
  "Lookback count" + 1
