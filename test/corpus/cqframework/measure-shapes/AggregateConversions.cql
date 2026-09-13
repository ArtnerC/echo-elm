// Aggregates over a list of FHIR values convert each element first.
//
// The aggregates overloaded per System type — Sum, Min, Max, Avg and their kin —
// need every element converted, and CQF lifts that conversion into
// `X return FHIRHelpers.To<T>(X)` over the list. The generic ones (Count,
// distinct, First) take the list as it is. Avg over FHIR Quantity additionally
// used to be widened to Decimal, where CQF converts with ToQuantity.
// (issues/06 follow-up)
library AggregateConversions version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

define "Obs": [Observation]
define "Encs": [Encounter]

define "MinStart": Min("Encs" E return E.period.start)
define "MaxEnd": Max("Encs" E return E.period.end)
define "MaxIssued": Max("Obs" O return O.issued)
define "MinStatus": Min("Encs" E return E.status)
define "SumValues": Sum("Obs" O return (O.value as FHIR.Quantity))
define "AvgValues": Avg("Obs" O return (O.value as FHIR.Quantity))

define "CountStatus": Count("Encs" E return E.status)
define "DistinctStatus": distinct ("Encs" E return E.status)
define "FirstStart": First("Encs" E return E.period.start)
