# cql-tests provenance

Generated from the HL7 CQL conformance suite by `internal/tools/cqltestsgen`.

| Field | Value |
|---|---|
| **Repository** | https://github.com/cqframework/cql-tests |
| **Commit** | `9e921932877bcf7a92e14a19c36a2b9cff077654 2026-09-22` |
| **License** | Apache-2.0 (see NOTICE) |
| **Modifications** | Each suite file's test expressions are wrapped as `define`s in one library; expressions are unchanged |

16 libraries, 1825 defines.

Left out: 5 tests the suite marks `invalid="syntax"` or `invalid="semantic"` (expected translation failures), and 12 the pinned CQF release rejects:

| Test | CQF 5.0.0 |
|---|---|
| `CqlDateTimeOperatorsTest/DateTimeComponentFrom.DateTimeComponentFromTimezoneOffset` | the timezone keyword is only valid at compatibility level 1.3 or lower |
| `CqlDateTimeOperatorsTest/Uncertainty tests.TimeDurationBetweenHourDiffPrecision` | syntax error at 'Z' in a Time literal |
| `CqlListOperatorsTest/Slice.SliceAll` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceEmpty` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceEnd` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceEndNull` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceNegative` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceNull` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SlicePast` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceStart` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceStartAndNegative` | no public Slice overload |
| `CqlListOperatorsTest/Slice.SliceStartNull` | no public Slice overload |
