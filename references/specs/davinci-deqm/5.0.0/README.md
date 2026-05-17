# Da Vinci DEQM 5.0.0 notes

## Sources

- Home: <https://hl7.org/fhir/us/davinci-deqm/>
- Gaps in Care: <https://hl7.org/fhir/us/davinci-deqm/gaps-in-care-reporting.html>

Da Vinci Data Exchange for Quality Measures (DEQM) 5.0.0 is a US Realm STU5 guide based on FHIR R4. It defines reporting and data-exchange patterns around quality measurement.

## Supported scenario families

- Reporting scenarios: communicating measure calculation results.
- Exchange scenarios: exchanging data of interest for a measure or measure set.
- Gaps in care scenarios: communicating actual or perceived care gaps.

## Reporting outputs

DEQM builds on MeasureReport and related bundles:

- summary reporting: aggregate population counts and scores,
- individual reporting: patient/member-level MeasureReports,
- subject-list reporting: subject membership lists,
- data exchange: data-of-interest Bundles,
- gaps in care: Bundle + Composition + DetectedIssue + Group + individual MeasureReports.

## Gaps in care

A care gap is a discrepancy between measure-specified standards of care and services provided or documented. Gaps can be open, closed, or prospective. DEQM uses the individual MeasureReport profile so gaps reporting can reuse measure-calculation machinery.

Key `$care-gaps` parameters include:

- required `periodStart` and `periodEnd` for the gaps-through period,
- optional `subject`, `practitioner`, and `organization`,
- required `status` with open/closed/prospective gap values,
- one or more measure selectors by `measureId`, `measureIdentifier`, or `measureUrl`,
- optional `program`.

The operation returns Parameters containing zero or more Bundles conforming to the DEQM Gaps In Care Bundle profile.

## Engine implications

A dQM engine that claims DEQM compatibility should:

- generate DEQM-conformant MeasureReports and Bundles,
- preserve evaluated-resource links when supporting data is included,
- record criteria contribution with CQF criteria reference extensions when required,
- distinguish retrospective and prospective gap calculations by gaps-through period,
- support data-of-interest extraction independently from scoring,
- support attribution outside the measure logic when a reporting scenario requires it.

