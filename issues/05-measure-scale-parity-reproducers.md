# Issue: Measure-Scale Parity — Verified Generic Reproducers for the Remaining Gaps

**Priority:** High (closes the final open action in `04-serializer-shapes-and-corrected-parity-gaps.md`)
**Labels:** parity, elm, corpus, fixtures, type-inference, resolution
**Related:** `03-cqf-parity-gaps.md`, `04-serializer-shapes-and-corrected-parity-gaps.md`, branch `feat/cqf5-and-serializer-shapes`

---

## Summary

Issue 04 closed with one unchecked action: *"Re-measure G3–G9 under the corrected profile
and a single serializer shape before acting."* This issue is that re-measurement, plus the
artifact it produces: **a set of small, generic, licence-clean CQL fixtures that reproduce
every remaining gap**, each verified against the pinned CQF 5.0.0 CLI *and* against
echo-elm as it stands today.

In dependency order:

1. **The corpus does not measure what the parity number implies.** `291/291` and `299/299`
   are real, but the corpus is CQF's own conformance suite plus a handful of synthetic
   libraries. Run against published measure content, echo-elm matches **0 of 39** libraries.
2. **Index-aligned diffing manufactures phantom findings.** Several counts carried forward
   from issue 03 are artifacts of comparing `statements.def[i]` across two differently
   ordered lists. They vanish under name alignment. Any future measurement must align by
   name first.
3. **The real gaps are few and mechanical**, and each one reproduces on a fixture of
   ten lines or fewer. Those fixtures are given below, with verified expected output.
4. **Two gaps do not reproduce at all** and should be recorded as negative controls rather
   than carried as open work.
5. **A missing option profile is why these gaps were invisible.** Result types are off in
   every corpus profile except `debug`. Measure content is published with result types on.
6. Several remaining differences are **harness normalization gaps, not translator bugs**,
   and should be fixed in the harness before anyone spends time in the translator.

Nothing in this issue contains third-party measure content. Every fixture is synthetic
and written for this repository, in the same spirit as `ra-measure/RAMeasure-1.0.0.cql`.

---

## Part 1 — Why the gaps were invisible

`corpus.yaml` defines seven option profiles. Exactly one of them, `debug`, sets
`enableResultTypes: true`. Published measure libraries declare this option set in their
`CqlToElmInfo` annotation:

```text
EnableLocators,EnableResultTypes,DisableListTraversal,DisableListDemotion,DisableListPromotion
signatureLevel: None
```

Result types are on, and list traversal/demotion/promotion are off. No corpus profile
matches that combination, so the entire type-inference and type-rendering surface was
never exercised in the shape real content uses.

**Add a profile that mirrors it.** The CLI flag names below were read out of
`cql-to-elm-cli-5.0.0.jar` (`org.cqframework.cql.cql2elm.cli.Main`), not guessed:

```yaml
  measure-bundle:
    description: >
      The option set published measure bundles declare in CqlToElmInfo. Result types
      are on, which is what makes the type-rendering and inference gaps observable;
      list traversal, demotion and promotion are off.
    cliFlags:
      - --locators
      - --result-types
      - --disable-list-traversal
      - --disable-list-demotion
      - --disable-list-promotion
    translatorOptions:
      enableAnnotations: false
      enableLocators: true
      enableResultTypes: true
      disableListTraversal: true
      disableListDemotion: true
      disableListPromotion: true
      signatureLevel: None
```

Every fixture in Part 3 is intended to run under this profile. Several of them are inert
without it.

---

## Part 2 — Methodology warning: align by name, never by index

This matters more than any individual gap, because it invalidates numbers already in
circulation.

A first pass compared `library.statements.def[i]` on each side. Because 12 of 39 measure
libraries emit definitions in a different order, that pass was diffing unrelated
definitions against each other. It reported, confidently:

| Reported substitution | Count | Real? |
|---|---|---|
| `If` → `Or` | 244 | **No** |
| `If` → `And` | 115 | **No** |
| `FunctionRef` → `Property` | 105 | **No** |
| `As` → `Null` | 319 | Partly — real, but not at this magnitude |

Re-run with definitions bucketed by `name` (skipping overloaded names, which cannot be
aligned without signatures) and guarded by locator equality, the first three vanish
entirely. The trustworthy list is two orders of magnitude smaller:

| Substitution | Count |
|---|---|
| `Property` → `ExpressionRef` | 16 |
| `SameOrBefore` → `Before` | 10 |
| `Contains` → `Includes` | 5 |
| `In` → `IncludedIn` | 4 |
| `Message` → `FunctionRef` | 4 |
| `CalculateAgeAt` → `FunctionRef` | 4 |
| `CalculateAge` → `FunctionRef` | 4 |
| `SameOrAfter` → `SameAs` | 2 |
| `FunctionRef` → `ToQuantity` | 2 |
| `InValueSet` → `In` | 1 |
| `FunctionRef` → `ToDecimal` | 1 |

The lesson is recursive: the unfixed ordering divergence *manufactured* evidence for a
node-substitution bug. This is the same category error issue 04 caught in issue 03, one
level deeper. **Fix the harness alignment before measuring anything else.**

A second consequence: `As` wrappers cannot be found by node-type comparison at all,
because CQF's wrapping `As` node carries no locator — only its inner operand does.
Detecting them needs a dedicated pass: *if the reference node type is a known wrapper and
one of its operands is structurally equal to echo's node, echo is missing a wrapper.*

---

## Part 3 — Verified generic reproducers

Every fixture below was compiled with `tools/cqframework/5.0.0/run.bat` and with
`go run ./cmd/echo-elm cqf translate`, both under the `measure-bundle` flag set. The
"CQF" and "echo-elm" columns are transcribed from the emitted JSON, not predicted.

**Each sample is complete and self-contained in this document.** Copy the block, save it
under the stated filename, and it compiles as-is — the only external dependency is
`FHIRHelpers-4.0.1.cql` in the same directory for the fixtures that include it. Adding
them to the corpus at `test/corpus/cqframework/measure-shapes/` is proposed as follow-up
work, not a prerequisite for reproducing anything here.

To reproduce a single fixture:

```bash
tools/cqframework/5.0.0/run.bat --input <file>.cql --format JSON \
  --locators --result-types \
  --disable-list-traversal --disable-list-demotion --disable-list-promotion

go run ./cmd/echo-elm cqf translate -input <file>.cql -format JSON \
  -locators -result-types \
  -disable-list-traversal -disable-list-demotion -disable-list-promotion \
  -output ./echo
```

### 3.1 Type-name qualification — `TypeNameQualification.cql`

Largest bucket by count: **3,174** of 7,928 result-type divergences are qualification-only,
meaning both sides inferred the same type and rendered it differently.

```cql
library TypeNameQualification version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

valueset "Office Visit": 'http://example.org/fhir/ValueSet/office-visit'

context Patient

// Retrieves infer the model type; the declarations below name it explicitly
// through the model alias. CQF renders every one as {http://hl7.org/fhir}Encounter.
define "All Encounters":
  [Encounter]

define "Encounters In Value Set":
  [Encounter: "Office Visit"]

define function "CountOf"(encounters List<FHIR.Encounter>):
  Count(encounters)

define function "First Condition"(conditions List<FHIR.Condition>):
  singleton from (conditions C where C.id is not null)

define "Encounter Count":
  "CountOf"("All Encounters")

define "Has Any Encounter":
  exists "All Encounters"

define "Value Set Reference":
  "Office Visit"
```

Distinct type names emitted across the whole library:

| CQF | echo-elm |
|---|---|
| `{http://hl7.org/fhir}Condition` | `Condition` |
| `{http://hl7.org/fhir}Encounter` | `Encounter` |
| `{http://hl7.org/fhir}id` | `FHIR.Condition` |
| `{urn:hl7-org:elm-types:r1}Boolean` | `FHIR.Encounter` |
| `{urn:hl7-org:elm-types:r1}Integer` | `{http://hl7.org/fhir}Encounter` |
| `{urn:hl7-org:elm-types:r1}ValueSet` | `{urn:hl7-org:elm-types:r1}Boolean` |
| | `{urn:hl7-org:elm-types:r1}Integer` |
| | `{urn:hl7-org:elm-types:r1}ValueSet` |

echo-elm emits **three different renderings of the same type inside a single library**:
`{http://hl7.org/fhir}Encounter`, `FHIR.Encounter`, and bare `Encounter`. System types
are already correct. The rule CQF applies without exception is `{modelUri}LocalName`.

At measure scale the three echo forms split `{uri}Name` 1,489 / `Alias.Name` 1,048 /
bare `Name` 637.

**Fix:** funnel every type-name rendering through one function. Sites are `namedTS` in
`internal/translator/typeinfer.go` and `qualifyDataType` in `internal/translator/translate.go`.
This is mechanical and should be done first — it will also stop it masking Part 3.2.

### 3.2 Missing FHIRHelpers conversions — `FHIRHelperConversions.cql`

> **Fixed.** Root cause was narrower than "inserts zero": echo-elm read the alias' model
> type out of the query source's *syntax*, so it was only found when a `Retrieve` was
> visible there. `"Encounters" E` has no Retrieve to find, so the alias carried no FHIR
> type and every implicit conversion inside the query was suppressed. `[Encounter] E`
> worked all along, which is why `elm-nodes/FHIRCoercionContext.cql` passed while measure
> content — where queries are written over defines — did not.
>
> The alias' inferred element type already knew the answer, so the type name now falls back
> to it. All six conversions in this fixture match, and the only residual difference is
> CQF's `CqlToElmError` overload warnings, which the harness already strips.
>
> A second fix was needed for the same fixture under `measure-bundle`: a function with no
> declared `returns` clause had no recorded return type, so calls to it degraded to `Any`.
> Inferred return types are now recorded and used for local and cross-library calls alike.
>
> Fixture: `measure-shapes/FHIRHelperConversions.cql`.

echo-elm inserted **zero** implicit model conversions here. CQF inserts six.

```cql
library FHIRHelperConversions version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "Measurement Period" Interval<DateTime>

context Patient

define "Encounters":
  [Encounter]

// System-typed parameters force CQF to insert the FHIRHelpers conversion at
// each call site. Without the conversion the operand is a raw FHIR Property.
define function "Label"(s String): s
define function "Flagged"(b Boolean): b
define function "Coded"(c Code): c
define function "Within"(p Interval<DateTime>): p
define function "AtTime"(d DateTime): d

define "Status Strings":
  "Encounters" E return "Label"(E.status)

define "Class Codes":
  "Encounters" E return "Coded"(E.class)

define "Period Intervals":
  "Encounters" E return "Within"(E.period)

define "Period Starts":
  "Encounters" E return "AtTime"(E.period.start)

define "Status Comparison":
  "Encounters" E where E.status = 'finished'

define "Period During MP":
  "Encounters" E where E.period during "Measurement Period"
```

| Definition | CQF | echo-elm |
|---|---|---|
| `Status Strings` | `FHIRHelpers.ToString(Property status)` | bare `Property` |
| `Class Codes` | `FHIRHelpers.ToCode(Property class)` | bare `Property` |
| `Period Intervals` | `FHIRHelpers.ToInterval(Property period)` | bare `Property` |
| `Period Starts` | `FHIRHelpers.ToDateTime(Property start)` | bare `Property` |
| `Status Comparison` | `FHIRHelpers.ToString(Property status)` | bare `Property` |
| `Period During MP` | `FHIRHelpers.ToInterval(Property period)` | bare `Property` |

Conversions are only inserted where a System type is *demanded*. That is why this gap
never surfaced on the existing corpus, which does not call FHIR-typed expressions into
System-typed positions.

At measure scale this class is roughly 150 occurrences: `ToString` 49, `ToInterval` 15,
`ToDateTime` 14, `ToCode` 11, `ToDecimal` 10, `ToQuantity` 2, plus the tail.

This is a correctness gap, not a cosmetic one — a downstream engine given a raw FHIR
`Property` where it expects a System value will fail or silently coerce.

### 3.3 Missing implicit cast wrappers — `ImplicitCastNull.cql`

```cql
library ImplicitCastNull version '1.0'

context Unfiltered

define function "Window"(period Interval<Date>, dates List<Date>):
  period

define "Null Interval Argument":
  "Window"(null, { @2024-01-01 })

define "Null List Argument":
  "Window"(Interval[@2024-01-01, @2024-12-31], null)
```

| Argument | CQF | echo-elm |
|---|---|---|
| `Null Interval Argument` arg0 | `As` with `asTypeSpecifier` = `IntervalTypeSpecifier{pointType: {urn:hl7-org:elm-types:r1}Date}` | bare `Null` |
| `Null List Argument` arg1 | `As` with `asTypeSpecifier` = `ListTypeSpecifier{elementType: {urn:hl7-org:elm-types:r1}Date}` | bare `Null` |

A bare `null` at a typed parameter must be wrapped in `As` carrying the declared type.
At measure scale, ~150 occurrences across `Interval<Date>`, `Date`, `Boolean`,
`List<Interval<Date>>`, `List<Condition>`, `List<Observation>`, `Tuple` and others.

### 3.4 Value set membership conversion — `ValueSetMembership.cql`

> **Expected to be fixed by 3.2** — it is the same defect at the terminology boundary.
> Not separately verified; needs its own fixture before being claimed.

```cql
library ValueSetMembership version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

valueset "Diabetes": 'http://example.org/fhir/ValueSet/diabetes'

context Patient

define "Conditions":
  [Condition]

define "Diabetic Conditions":
  "Conditions" C where C.code in "Diabetes"
```

Both sides correctly select `InValueSet`. They differ in its `code` operand:

| | code operand |
|---|---|
| CQF | `FHIRHelpers.ToConcept(Property code)` |
| echo-elm | bare `Property code` |

This is the same defect as 3.2 surfacing at the terminology boundary, and it is the one
with the clearest clinical consequence: value set membership evaluated against an
unconverted FHIR `CodeableConcept` is not the same test.

### 3.5 Temporal operator selection — `TemporalOperatorSelection.cql`

```cql
library TemporalOperatorSelection version '1.0'

context Unfiltered

define "MP":
  Interval[@2024-01-01, @2024-12-31]

define "D":
  @2024-06-15

define "Sub":
  Interval[@2024-03-01, @2024-04-01]

define "Point In Interval":
  "D" in "MP"

define "Interval In Interval":
  "Sub" included in "MP"

define "Interval Contains Point":
  "MP" contains "D"

define "Interval Includes Interval":
  "MP" includes "Sub"

define "Same Or Before":
  "D" same or before end of "MP"

define "Same Or After":
  "D" same or after start of "MP"

define "Strictly Before":
  "D" before end of "MP"
```

| Definition | CQF | echo-elm | |
|---|---|---|---|
| `Point In Interval` | `In` | `In` | ok |
| `Interval In Interval` | `IncludedIn` | `IncludedIn` | ok |
| `Interval Contains Point` | `Contains` | `Contains` | ok |
| `Interval Includes Interval` | `Includes` | `Includes` | ok |
| `Same Or Before` | `SameOrBefore` | **`SameAs`** | **wrong** |
| `Same Or After` | `SameOrAfter` | **`SameAs`** | **wrong** |
| `Strictly Before` | `Before` | `Before` | ok |

`same or before` and `same or after` both collapse to `SameAs`. This is the
highest-severity item in the whole set despite the smallest count: it silently changes the
truth conditions of the expression. `"D" same or before X` is true for every `D` at or
before `X`; `SameAs` is true only for equality.

Note this contradicts the direction recorded in issue 03/04, which had it as
`SameOrBefore` → `Before`. The name-aligned re-measurement shows `SameAs`.

### 3.6 Fluent function library resolution — `FluentHelpers.cql` + `FluentCrossLibrary.cql`

> **Fixed for the unambiguous case.** A fluent call through an expression receiver now
> resolves its `libraryName` from the fluent functions the included libraries declare, and
> the call's result type resolves through a new `libFuncReturns` map — `defTypeSpecs`
> deliberately excludes functions, so there was nothing to look up before. Fixtures:
> `measure-shapes/FluentCrossLibrary.cql` + `FluentHelpers.cql`, System-typed on purpose so
> they fail for one reason or none.
>
> **Still open:** an overloaded fluent name declared by more than one include. Choosing
> between them needs the receiver's element type, which is overload resolution rather than
> qualification, and depends on 3.1. Rather than guess, `fluentLibraryFor` returns nothing
> when the name is ambiguous — a confidently wrong `libraryName` is worse than a missing one.

Issue 04 recorded G6 as *"confirmed and fixed"*. It was not. The existing fix in
`resolveLibraryQualifiedCalls` (`internal/translator/translate.go`) bails out unless the
receiver is an include alias:

```go
if _, isLib := t.libSyms[ir.Name]; !isLib { return }
```

Real content invokes **fluent functions through an expression reference**, which never
takes that path. 364 `FunctionRef` nodes are still missing `libraryName`.

```cql
// FluentHelpers.cql
library FluentHelpers version '1.0'

using FHIR version '4.0.1'

define fluent function "Finished"(encounters List<FHIR.Encounter>):
  encounters E where E.status.value = 'finished'

define fluent function "Finished"(conditions List<FHIR.Condition>):
  conditions C where C.clinicalStatus is not null
```

```cql
// FluentCrossLibrary.cql
library FluentCrossLibrary version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers
include FluentHelpers version '1.0'

context Patient

define "Encounters":
  [Encounter]

define "Conditions":
  [Condition]

// Fluent invocation through an EXPRESSION receiver, not an include alias.
// The overload is selected by the receiver's element type.
define "Finished Encounters":
  "Encounters"."Finished"()

define "Finished Conditions":
  "Conditions"."Finished"()
```

| Definition | CQF | echo-elm |
|---|---|---|
| `Finished Encounters` | `FunctionRef name='Finished' libraryName='FluentHelpers'` | `FunctionRef name='Finished' libraryName=<absent>` |
| `Finished Conditions` | `FunctionRef name='Finished' libraryName='FluentHelpers'` | `FunctionRef name='Finished' libraryName=<absent>` |

Two properties of this fixture matter and should not be simplified away:

- The include is **unaliased**, so the library name has to come from the declaration.
- The fluent function is **overloaded**, so resolution requires the receiver's element
  type — which ties this gap to 3.1 and 3.2. Fix type rendering first.

### 3.7 Implicit context accessor position — `ContextAccessorPosition.cql`

```cql
library ContextAccessorPosition version '1.0'

using FHIR version '4.0.1'

define "Declared Before Context":
  1 + 1

context Patient

define "Declared After Context":
  2 + 2
```

| | `statements.def` order |
|---|---|
| CQF | `['Declared Before Context', 'Patient', 'Declared After Context']` |
| echo-elm | `['Patient', 'Declared Before Context', 'Declared After Context']` |

CQF emits the implicit context accessor **in the source position of the `context`
declaration**. echo-elm prepends it. `internal/translator/translate.go` appends all
context accessors at lines 939–946, before explicit statements are appended at line 948.

This is a one-line-of-reasoning fix and it accounts for a large share of the 12/39
ordering divergences on its own.

### 3.8 Negative controls

Two fixtures that **already match** and should be added anyway, to stop the corresponding
claims being reopened and to catch regressions in the fixes above.

`StatementOrdering.cql` — dependency ordering is correct:

```cql
library StatementOrdering version '1.0'

context Unfiltered

define "Alpha":
  "Gamma" + 1

define "Beta":
  2

define "Gamma":
  3
```

Both sides emit `['Gamma', 'Alpha', 'Beta']`. Neither is source order; both resolve
dependencies first with a source-order tiebreak. **The ordering algorithm is not the
bug.** The 12/39 divergence is entirely downstream of 3.6 (wrong resolution creates
phantom dependency edges) and 3.7 (accessor position). Issue 04's G7 should be reworded
accordingly rather than treated as an independent defect.

The simple type-rendering path also already matches — a library with only retrieves and
System types produces identical type names on both sides. The gap in 3.1 requires
explicitly declared, alias-qualified type specifiers to surface. Without that, a fixture
here is inert.

---

## Part 4 — Harness gaps, not translator gaps

These account for roughly 3,461 raw differences on the bundle comparison path and should
be excluded from any translator work item.

| Field | Count | Cause |
|---|---|---|
| `signature` | 2,619 | Empty array emitted by echo; the JAXB writer used inside `Library.content[]` omits empty collections |
| `let` | 670 | as above |
| `codeFilter` | 31 | as above |
| `dateFilter` | 31 | as above |
| `include` | 31 | as above |
| `otherFilter` | 31 | as above |
| `element` | 9 | as above |
| `signatureLevel` | 39 | one per library; not stripped by `stripVolatileFields` |

`cqfEmptyArrayField()` is gated on `CQFMode`, which is correct for the CLI shape per issue
04's G1/G2 withdrawal. The bundle path is the problem: `parity --bundle` reads reference
ELM out of `Library.content[]`, which is the **JAXB shape**, and
`stripEmptyAnnotations()` in `internal/parity/runner.go:529` only reduces empty
`annotation` and `t`. Extend it, or the bundle path will keep reporting a translator bug
that does not exist.

`simpleDiff()` should also stop being used to characterise divergence. It is line-by-line
and massively overstates the difference; it is fine for eyeballing a single fixture and
useless for counting.

---

## Part 5 — Result-type attachment policy

Of the 7,928 result-type divergences, once 3.1 (qualification, 3,174) is removed:

| Bucket | Count | Share |
|---|---|---|
| `missing_in_echo` | 3,241 | 40.9% |
| `qualification_only` | 3,174 | 40.0% |
| `genuine_type_difference` | 919 | 11.6% |
| `extra_in_echo` | 594 | 7.5% |

`missing_in_echo` and `extra_in_echo` together are an **attachment policy** difference —
which node kinds get a result type at all — not an inference difference. That is a survey
of `elm.SetResultType` call sites, and it should be settled before anyone looks at the 919.

The 919 are the real residual of issue 04's G3. Representative case, 102 occurrences:

| | inferred |
|---|---|
| CQF | `List<{urn:hl7-org:elm-types:r1}Date>` |
| echo-elm | `List<{urn:hl7-org:elm-types:r1}Any>` |

An element type collapsing to `Any` is the signature of list element inference giving up.
Given `disableListPromotion`/`disableListDemotion` are on in the measure profile, check
those paths first.

---

## Part 6 — Revisiting issue 04

| Item | Recorded in 04 | Evidence now | Verdict |
|---|---|---|---|
| G1, G2 (empty arrays) | Withdrawn | ~3,461 on the bundle path | Withdrawal correct **for the CLI shape**. Recurs on the bundle path as a harness gap — see Part 4 |
| G3 (result types) | Not re-measured; "large" | 7,928, split 40/40/12/8 | **Re-scope** into three items: rendering (3.1), attachment policy (Part 5), inference (919) |
| G4 (annotations) | Not re-measured | Normalizer collapses annotations before comparison | **Structurally unmeasurable** by the current harness. Either instrument it or close it |
| G5 (context) | Does not reproduce; blocked | 558 occurrences of `Patient` vs `Unfiltered` | **Reopen.** The doc's guess — context established through an included library — is the right lead |
| G6 (`libraryName`) | Confirmed and fixed | 364 still missing | **Not fixed.** Fix covers include-alias receivers only; see 3.6 |
| G7 (ordering) | Fixed | 12/39 still diverge | **Reword.** The algorithm is correct (3.8); divergence is downstream of G6 and the accessor position (3.7) |
| G8 | Not re-measured | — | Fold into Part 5 |
| G9 (`Or`/`And`) | Not re-measured | Vanishes under name alignment | **Withdraw.** Artifact of index-aligned diffing. The genuine operator gap is 3.5, and it is a different pair |

---

## Part 7 — Suggested issue split

| # | Title | Scope | Depends on |
|---|---|---|---|
| 1 | Harness: align definitions by name; add wrapper-detection pass | `internal/parity/runner.go` | — |
| 2 | Harness: reduce empty containers and `signatureLevel` on the bundle path | `stripEmptyAnnotations`, `stripVolatileFields` | — |
| 3 | Corpus: add `measure-bundle` profile and `measure-shapes/` fixtures | `corpus.yaml`, new fixtures | — |
| 4 | Single type-name rendering path | `typeinfer.go`, `translate.go` | 3 |
| 5 | Result-type attachment policy survey | `elm.SetResultType` call sites | 1, 4 |
| 6 | Implicit model conversions (`FHIRHelpers.To*`) | conversion insertion | 4 |
| 7 | Implicit cast wrappers around untyped literals | `As` insertion | 4 |
| 8 | Fluent function resolution through expression receivers | `resolveLibraryQualifiedCalls` | 4 |
| 9 | Context accessor source position | `translate.go:939-946` | — |
| 10 | `same or before` / `same or after` operator selection | operator lowering | — |
| 11 | Context propagation from included libraries (G5) | context resolution | 8 |
| 12 | Residual type inference — `List<Any>` collapse | list element inference | 5 |

Recommended order: 1, 2, 3 (measure honestly), then 9 and 10 (small, independent, and 10
is a correctness bug), then 4, then 6, 7, 8, 11, and finally 5 and 12.

---

## Part 8 — Licensing constraint

Published HEDIS measure content is NCQA-licensed and must not be committed to this
repository, quoted in issues, or included in the corpus. The measurements in this issue
were taken against a local, gitignored copy; only counts, node types and structural
shapes are reported here.

Every fixture in Part 3 is synthetic and written for this repository. Where a fixture
mirrors a real construct, it mirrors the *shape* only — the fluent-overload-through-
expression-receiver pattern, a `define` preceding `context`, a `null` at a typed
parameter. This is the same approach already taken by `ra-measure/RAMeasure-1.0.0.cql`.

---

## Status

| Item | Outcome |
|---|---|
| Issue 04 final action — re-measure G3–G9 | **Done**; results in Parts 3, 5, 6 |
| Generic reproducers | **Done**; 9 fixtures, all compiled against CQF 5.0.0 and echo-elm |
| Gaps reproducing on a generic fixture | 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7 |
| Gaps confirmed **not** reproducing | statement ordering, simple-path type rendering (3.8) |
| Gaps still without a reproducer | G5 context propagation (558 occurrences, lead identified in Part 6) |
| Fixes applied | **None.** This issue is measurement and reproducers only |

---

## Progress against Part 7

| # | Item | State |
|---|---|---|
| 1 | Harness: align definitions by name | open |
| 2 | Harness: reduce empty containers + `signatureLevel` on the bundle path | **done** |
| 3 | Corpus: `measure-bundle` profile and `measure-shapes/` fixtures | **profile done**, 6 of 9 fixtures |
| 4 | Single type-name rendering path | open |
| 5 | Result-type attachment policy survey | open |
| 6 | Implicit model conversions (`FHIRHelpers.To*`) | **done** for query-source aliases (3.2); 3.4 expected to follow, unverified |
| 7 | Implicit cast wrappers around untyped literals | open |
| 8 | Fluent function resolution through expression receivers | **done** for the unambiguous case; overloaded fluent names still need receiver-type resolution |
| 9 | Context accessor source position | **done** |
| 10 | `same or before` / `same or after` operator selection | **done** |
| 11 | Context propagation from included libraries (G5) | open |
| 12 | Residual type inference — `List<Any>` collapse | open |

Corpus parity against CQF 5.0.0 after 6, 8, 9 and 10: **324/324**, including the new
`measure-bundle` profile. That number covers the corpus, which is CQF's conformance suite
plus synthetic libraries — it is not evidence about measure-shaped content, which is the
whole point of Part 1.

---

## Actions

- [x] Re-measure G3–G9 under the corrected profile and a single serializer shape *(closes the open action in `04-serializer-shapes-and-corrected-parity-gaps.md`)*
- [ ] Fix definition alignment in the parity harness (name-bucketed, overload-skipping) before any further measurement
- [ ] Add the wrapper-detection pass so missing `As` nodes are countable
- [x] Extend `stripEmptyAnnotations`/`stripVolatileFields` to cover the bundle path (Part 4) — `normalizeShape` + `Config.BundleShapedRef`, scoped by test so the CLI path keeps checking those fields
- [x] Add the `measure-bundle` option profile to `corpus.yaml` — `StringFunctionsTest` excluded from it; CQF itself cannot resolve `IndexOf(String, String)` without demotion
- [~] Lift the Part 3 samples into `test/corpus/cqframework/measure-shapes/` — 5 of 9 landed (3.5, 3.6, 3.7, and the 3.8 negative control), i.e. the ones whose gap is now fixed. The rest land with their fixes
- [x] Amend `04-serializer-shapes-and-corrected-parity-gaps.md` per the Part 6 table — G6 downgraded to partially fixed, G7 reworded as a consequence, G9 withdrawn, G5 reopened
- [ ] Correct the parity claim in the PR description and README: state the corpus it covers, and that measure-shaped content is not yet covered
- [ ] Split Part 7 into tracked issues
