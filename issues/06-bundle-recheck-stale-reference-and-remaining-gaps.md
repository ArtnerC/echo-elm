# Issue: HEDIS-Bundle Re-Measurement — Stale Reference ELM, Not an echo-elm Defect (Plus the Real Residual Gaps)

**Priority:** High (answers the "re-measurement" ask from `05-measure-scale-parity-reproducers.md` Part 9)
**Labels:** parity, elm, bundle, context, result-types, root-cause
**Related:** `04-serializer-shapes-and-corrected-parity-gaps.md`, `05-measure-scale-parity-reproducers.md`, branch `feat/cqf5-and-serializer-shapes`

---

## Summary

Issue 05 (Part 9) asked for a re-measurement against real HEDIS content once the fixes in
that issue landed. This is that re-measurement, run against four published HEDIS 2025.2.1
measure bundles (39 libraries total, no duplicates across bundles once shared libraries
are deduplicated). It reaches a different conclusion than the raw match/differ counts
suggest:

1. **The published bundles' embedded reference ELM (`Library.content[]`) does not agree
   with a fresh CQF 5.0.0 CLI compile of the same CQL source.** Re-running the exact
   source text through the pinned `cql-to-elm-cli-5.0.0.jar` — the same jar this repo's
   parity harness already vendors — produces ELM that **matches echo-elm's output far
   more often than it matches the bundle's own embedded reference.** The bundle's
   reference ELM was compiled with a different translator configuration (or a different
   translator build) than 5.0.0, and every downstream count in this issue changes once
   that's accounted for.
2. **The `context` divergence (the largest single bucket in every bundle rerun) is 100%
   explained by this.** Across all 965 statements checked in the largest bundle, fresh
   CQF 5.0.0 agrees with echo-elm on `context` in every single case (100%) and agrees with
   the bundle's embedded reference in only 31.1% of cases. This is not an echo-elm bug.
3. **The reported `If`→`Or` "divergence" is the same root cause, not a translator bug.**
   A fresh CQF 5.0.0 compile of the exact same source produces `Or`, matching echo-elm —
   the bundle's embedded reference is the one still emitting the older `If`-wrapped shape.
4. **Once the stale-reference noise is removed, there are real, still-open gaps**, and
   they are the same shapes issue 04/05 already tracked, just confirmed at bundle scale:
   an `As`-wrapping-`Null` normalization gap, a `FunctionRef`↔`Property` conversion-site
   gap, and list/interval `TypeSpecifier` qualification gaps. None of these are new; this
   section exists to give exact counts now that the reference-staleness noise is removed.
5. **This does not change PR #3's mergeability or the corpus-based 306/335 parity claim.**
   The corpus fixtures in `test/corpus/cqframework` were already compiled fresh by this
   repo's own harness, so they were never affected by the staleness in this issue. This
   issue is specific to `--bundle`-profile comparisons against externally-supplied,
   pre-compiled reference ELM.

Nothing in this issue contains third-party measure content. Every reproducer below is
synthetic, mirrors the pinned CQF options profile
(`EnableLocators,EnableResultTypes,DisableListTraversal,DisableListDemotion,DisableListPromotion`),
and was verified against the vendored `tools/cqframework/5.0.0/cql-to-elm-cli.jar`.

---

## Part 1 — What was actually run

Four published HEDIS 2025.2.1 measure bundles were re-checked against the current state
of `feat/cqf5-and-serializer-shapes`, using the `--bundle` profile (reads reference ELM
out of each bundle's own `Library.content[]`, per issue 05 Part 4):

| Bundle | Libraries | Match | Differ (JSON) | Errors |
|---|---|---|---|---|
| AAB_Reporting | 39 | 0 | 39 | 0 |
| ACP_Reporting | 36 | 0 | 36 | 0 |
| ADDE_Reporting | 36 | 0 | 36 | 0 |
| AMR_Reporting | 39 | 0 | 39 | 0 |

Read naively, that's 0/150 — a regression from the 306/335 corpus number. It isn't one.
The rest of this issue is the re-measurement that explains why.

For comparison, the CQF-conformance corpus run on the same commit (`test/corpus/cqframework`,
compiled fresh by this repo's own harness, not sourced from an external bundle):

| Status | Count |
|---|---|
| ✓ Match | 306 |
| ≠ Differ (JSON) | 26 |
| ✗ Errors | 0 |
| **Total** | **335** |

The corpus number is unaffected by anything in this issue — see Part 5.

---

## Part 2 — The bundle's reference ELM does not match a fresh CQF 5.0.0 compile

**Method:** for every `.cql` source file backing the AAB_Reporting bundle, the exact
source text was recompiled with the same `tools/cqframework/5.0.0/cql-to-elm-cli.jar` this
repo already vendors and pins for corpus parity, using the bundle's declared option
profile (`--locators --result-types --disable-list-traversal --disable-list-demotion
--disable-list-promotion`, matching the bundle's own `CqlToElmInfo.translatorOptions`
annotation exactly). The result — call it **fresh-CQF** — was then compared against both
the bundle's embedded reference ELM (**bundle-ref**) and echo-elm's own output
(**echo**), statement-by-statement, with locators/annotations/signature arrays stripped
(the harness-known-gap fields from issue 05 Part 4).

| Comparison | Common statements | Exact match | Match rate |
|---|---|---|---|
| fresh-CQF vs **bundle-ref** | 965 | 118 | 12.2% |
| fresh-CQF vs **echo** | 965 | 40 | 4.1% |
| bundle-ref vs echo | 965 | 15 | 1.6% |

At first glance fresh-CQF doesn't match either side well — full-statement equality is a
strict bar (result types, resultTypeSpecifiers, and locators all have to line up). But
narrowed to the single field that dominates every bucket table in issue 04/05 —
`context` — the picture is unambiguous:

| Comparison (context field only) | Common statements | Exact match |
|---|---|---|
| fresh-CQF vs **echo** | 965 | **965 (100.0%)** |
| fresh-CQF vs **bundle-ref** | 965 | 300 (31.1%) |

**echo-elm's context assignment agrees with a fresh, correctly-configured CQF 5.0.0
compile on every single statement in this bundle.** The bundle's own embedded reference
ELM does not. The bundle-ref annotation carries
`translatorOptions: EnableLocators,EnableResultTypes,DisableListTraversal,DisableListDemotion,DisableListPromotion`
but **no `translatorVersion` field** — unlike a fresh 5.0.0 compile, which always stamps
`"translatorVersion": "5.0.0"`. The reference ELM shipped in these bundles was compiled by
something that either predates the annotation's `translatorVersion` field or strips it,
and produces different context-inference and node-lowering behavior than the pinned
5.0.0 jar. Whatever produced it is not the artifact this repo is supposed to match.

### Verified generic reproducer — `ContextAfterDefine.cql`

The actual pattern in every affected bundle statement is a `define` that appears
**before** the library's `context Patient` declaration but is semantically scoped to it
once the context is in effect (e.g. a `Measure` metadata tuple declared above `context
Patient`, or a private helper function with no context declaration of its own inside a
library that never declares `context Patient` at all but is `include`d only from
patient-context statements). Minimal synthetic reproduction:

```cql
library ContextAfterDefine version '1.0'

using FHIR version '4.0.1'

define "Measure":
    Tuple { name: 'x' }

context Patient

define "Initial population":
    true
```

| Definition | fresh CQF 5.0.0 | echo-elm |
|---|---|---|
| `Measure` | `Unfiltered` | `Unfiltered` |
| `Initial population` | `Patient` | `Patient` |

Both agree: a `define` positioned before the library's `context` declaration is
`Unfiltered`, never retroactively promoted. echo-elm's behavior here (already covered by
`internal/translator/translate.go`'s `currentContextName` tracking, updated only when a
`context` statement is encountered in source order) is correct and matches the reference
implementation. **This confirms the divergence in the real bundles is not a
before/after-context positional bug in echo-elm** — it's specifically that the bundle's
own `Measure` statement (source line 33, before `context Patient` on line 41) should be
`Unfiltered` per both fresh-CQF and echo, and indeed *is* `Unfiltered` on both — the
mismatch is on statements the bundle marks `Patient` that neither fresh-CQF nor echo
agree with, which is the staleness described above, not a positional edge case.

---

## Part 3 — The `If`→`Or` "divergence" is the same root cause

Issue tracking from the prior session flagged an unresolved case where the bundle
reference showed `expression.condition.type: "If"` and echo-elm showed `"Or"`, on a
pattern shaped like `if a is null or b is null then null else ...`. Re-verified against
fresh CQF 5.0.0:

### Verified generic reproducer — `IfOrRepro.cql`

```cql
library IfOrRepro version '1.0'

using FHIR version '4.0.1'

define fluent function "Interval"(coverage FHIR.Coverage):
    if coverage is null or coverage.period is null then null
    else
        coverage.period.start
```

| Compiler | `expression.condition.type` |
|---|---|
| fresh CQF 5.0.0 CLI | `Or` |
| echo-elm | `Or` |
| bundle's embedded reference (same statement, real source) | `If` |

**Fresh CQF 5.0.0 and echo-elm agree; the bundle reference is the outlier.** This is not
CQF lowering short-circuit `Or` into an `If` node — a vanilla 5.0.0 compile keeps it as a
plain `Or` boolean expression, exactly like echo-elm does. Whatever compiled the bundle's
reference ELM is, again, not running the pinned 5.0.0 translator. **Close this as a
non-defect** — same root cause as Part 2, not a second bug.

---

## Part 4 — What's left once staleness is removed: the real residual gaps

Comparing **fresh-CQF 5.0.0 vs echo-elm only** (bundle-ref excluded entirely) across all
39 AAB_Reporting libraries, on statements present in both:

| Metric | Value |
|---|---|
| Common statements | 965 |
| Exact structural match (all fields, locator/annotation/signature stripped) | 40 (4.1%) |
| Structural difference | 925 (95.9%) |

That 4.1% headline sounds alarming, but per issue 04/05's precedent, full-statement
equality is the wrong granularity — result-type and type-specifier fields dominate any
node that touches a typed FHIR property, and one qualification-shape gap cascades into
every statement that calls the affected function. Node-type substitution buckets,
aggregated across all 39 libraries (fresh-CQF type → echo-elm type, where the two
compilers chose different ELM node types for the same source construct):

| Fresh-CQF type | echo-elm type | Count | Status |
|---|---|---|---|
| `As` | `Null` | 277 | Open — see 4.1 |
| `FunctionRef` | `Property` | 161 | Open — matches issue 04 G3/G4 (conversion-site collapsing), tracked |
| `IntervalTypeSpecifier` | `NamedTypeSpecifier` | 119 | Open — matches issue 05 Part 3.1 (qualification), tracked |
| `ListTypeSpecifier` | `NamedTypeSpecifier` | 94 | Open — same family as above |
| `As` | `FunctionRef` | 39 | Open — related to 4.1 |
| `NamedTypeSpecifier` | `ListTypeSpecifier` | 36 | Open — same family, opposite direction |
| `FunctionRef` | `As` | 23 | Open — related to 4.1 |
| `Property` | `ExpressionRef` | 22 | Open — matches issue 05's `Property`→wrapped-function-call pattern |
| `FunctionRef` | `AliasRef` | 20 | Open — new; not previously catalogued |
| `In` | `IncludedIn` | 17 | **Did not reproduce on a minimal fixture — see 4.2** |
| `SameOrBefore` | `Before` | 15 | Open — precision-comparison operator selection |
| `IntervalTypeSpecifier` | `ListTypeSpecifier` | 13 | Open — same family as qualification gaps |

None of these are new discoveries — they are the same shapes issue 04's G3/G4/G8 and
issue 05's Part 3.1/3.3 already opened tickets for. This table exists to give real,
staleness-free counts at bundle scale, so whoever picks up those tickets has an accurate
sense of blast radius: **the qualification family (`IntervalTypeSpecifier`/
`ListTypeSpecifier` ↔ `NamedTypeSpecifier`, ~262 occurrences) and the `As`/`FunctionRef`/
`Property` conversion-site family (~522 occurrences) account for the overwhelming
majority of remaining structural difference** once the reference-staleness explained in
Parts 2–3 is subtracted out.

### 4.1 `As` wrapping `Null` — echo drops the `resultTypeName` narrowing

```json
// fresh CQF 5.0.0
{
  "type": "As",
  "asType": "{urn:hl7-org:elm-types:r1}Integer",
  "operand": {
    "type": "Null",
    "resultTypeName": "{urn:hl7-org:elm-types:r1}Any"
  }
}
// echo-elm — same statement
{
  "type": "As",
  "asType": "{urn:hl7-org:elm-types:r1}Integer",
  "operand": {
    "type": "Null",
    "resultTypeName": "{urn:hl7-org:elm-types:r1}Integer"
  }
}
```

Both keep the outer `As` node (this bucket's label is a shorthand for "the operand's
`resultTypeName` differs", not a missing node — the classifier used for the aggregate
table stops descending once it finds a type match, so it's reported at the `Null` leaf).
CQF leaves the wrapped `Null`'s own `resultTypeName` at `Any` (the untyped null literal's
natural type) and relies on the outer `As` to narrow; echo-elm appears to push the
narrowed type down onto the inner `Null` node itself. Cosmetically harmless for
evaluation (the `As` node still governs the expression's effective type either way) but
worth a small fix for exact-match parity — narrow only at the `As` node, not at its
operand.

### 4.2 `In` → `IncludedIn` — did not reproduce on a minimal fixture; needs a real trigger

```cql
library InVsIncludedIn version '1.0'

context Unfiltered

define "PointInInterval":
    Today() in Interval[@2020-01-01, @2020-12-31]
```

Both fresh CQF 5.0.0 and echo-elm emit `In` for this minimal point-in-interval check —
this fixture does **not** reproduce the substitution. The 17 real occurrences all involve
an interval-endpoint expression (`Start of X in Interval[...]`) as the left operand rather
than a bare point value; the minimal fixture above uses a bare point and doesn't trigger
whatever operand-shape condition causes echo to select `IncludedIn`. **This needs a
second, more targeted fixture** built around an interval-endpoint-in-interval shape before
it can be handed to translator work — recorded here as a known open item rather than a
verified reproducer, per issue 05's own distinction between "verified" and "not yet
reproduced" gaps.

---

## Part 5 — Why the corpus number is unaffected

`test/corpus/cqframework`'s reference ELM is generated by this repo's own harness
invoking the vendored `tools/cqframework/5.0.0/cql-to-elm-cli.jar` at parity-run time —
see `internal/parity/runner.go`'s `runUpstream`. There is no externally-supplied,
pre-compiled reference artifact anywhere on that path; both sides of every corpus
comparison are always freshly compiled by the exact same jar this issue used to
root-cause the bundle divergence. **306/335 stands.** The `--bundle` profile is a
different code path specifically because it exists to test against real-world,
externally-produced reference ELM — which is exactly the path this issue shows is
carrying stale reference content.

---

## Part 6 — Resolution (branch `feat/issue06-bundle-residuals`)

No HEDIS content was available locally, so every item was reproduced synthetically
against the pinned CQF 5.0.0 CLI under the bundle's own option profile
(`--locators --result-types --disable-list-traversal --disable-list-demotion
--disable-list-promotion`), with FHIRHelpers source both absent and present. Every fix
is pinned by a corpus fixture under `test/corpus/cqframework/measure-shapes/`. Several
of this issue's diagnoses changed on contact with a reproducer.

### 6.1 Corrections to this issue

- **The corpus is 332/335 with 0 differ**, not 306/335 with 26 differ. The three
  unmatched are the `expectedStatus: failure` fixtures, which are skipped. That was true
  on `feat/cqf5-and-serializer-shapes` before this branch and is still true after it.
- **4.1 was misdiagnosed.** The dominant shape is echo-elm *omitting* the `As` wrapper —
  `1 = null`, `1 + null`, `case … else null` over an `Interval<DateTime>` branch,
  `if … then null else Now()`, `Coalesce(Now(), null)`. The Part 4 table (`As` → `Null`)
  was right and the 4.1 prose was wrong. The prose's shape — the inner `Null` stamped
  with the narrowed type — is real too, but only in `Interval` bounds. Both are fixed.
- **4.2 reproduces.** The trigger is `during` / `included in`, not `in`, with a
  `Start`/`End` left operand: `start of E.period during P`, `E.period starts during P`.
  Two root causes: `FHIRHelpers.ToInterval` had no known return type, so the point-ness
  of `Start(ToInterval(…))` could not be inferred; and `starts`/`ends` in a timing phrase
  were ignored outright.
- **"Already tracked elsewhere" did not hold for most of Part 4.** Several families were
  new defects, and four of them change evaluation results, not just ELM shape.

### 6.2 Fixed

| Gap | Effect | Fix |
|---|---|---|
| `on or before`, `before or on`, `on or after` emitted `Before` / `After` | **Wrong results** — the boundary was excluded | The grammar makes `on or` / `or on` single tokens *containing a space*; the inclusive relationship is now read |
| `starts` / `ends` and a trailing `start` / `end` ignored | **Wrong results** — the whole interval was compared instead of its boundary | Carried on `TimingExpr` and applied as `Start` / `End` |
| Quantity offsets ignored (`1 day or less on or before`) | **Wrong results** — the offset was dropped | Lowered exactly as CQF does: `SameAs`, `SameOrBefore`/`After`, `Before`/`After`, `In(open interval)`, `And(In(interval), Not(IsNull(anchor)))` |
| `Now()` / `Today()` / `TimeOfDay()` emitted as `FunctionRef` | **Unevaluable** — names a function that does not exist | `NullaryOperatorNode` |
| `during` / `included in` with a `Start`/`End` left operand → `IncludedIn` (4.2) | Operator selection | `Start`/`End` are points; a FHIR `Period`/`Range` never is |
| Precision timing comparisons typed `Integer` | Result type | Only `DurationBetween`, `DifferenceBetween` and `CalculateAgeAt` are integer-valued |
| Null left bare in binary operators, `case`, `if`, `Coalesce` (4.1) | Overload not recoverable from ELM | Typed from the *translated* sibling, not its syntax |
| Null in an `Interval` bound stamped with the bound's type (4.1) | Result type | The null keeps `Any`; the `As` narrows |
| `FHIRHelpers.X(y)` with FHIRHelpers unresolved → method call on an `ExpressionRef` | **Wrong call** | Include aliases are library qualifiers whether or not their source resolves |
| Argument of an explicit FHIRHelpers call converted again | Double conversion | Translated as unconsumed |
| `ToCode`/`ToConcept` typed `String`; `ToInterval` untyped | Result types; cascaded into `Interval<Any>` | Correct return types; `ToInterval` by argument (Period / Range) |
| `with` alias over a define never typed | Missing conversions on one side of a comparison | Same fallback query sources got in issues/05 3.2 |
| Consumed `x as FHIR.T` not converted | Missing conversions | Converted exactly as a property is |
| Consumed alias over a convertible FHIR type not converted | Membership tested against an unconverted `CodeableConcept` | Converted at consumption |
| `Interval[E.period.start, E.period.end]` bounds converted | Extra conversions | Left as `Interval<FHIR.dateTime>` when both bounds are properties |
| Choice alternatives in declaration order | Serialization | Sorted by CQF's `toString()` order |
| Choice specifiers, choice-typed parameters and choice-typed FHIR properties untyped | Result types | Choice type specs; the ModelInfo generator now captures choice elements |
| Argument to a choice-typed parameter not cast | Overload not recoverable | `As{asTypeSpecifier: Choice<…>}` |
| `--bundle` compared silently against a stale reference | Harness | Warns when a reference library's `translatorVersion` is missing or is not the pin |

### 6.3 Follow-up fixes

The two shapes this section originally left bounded are fixed, along with three more
that probing them turned up. Each is pinned by
`measure-shapes/TimingWithinAndPromotion.cql` or `measure-shapes/AggregateConversions.cql`.

| Gap | Effect | Fix |
|---|---|---|
| `within q of Y` | **Wrong results** — the quantity was dropped, comparing X against Y itself | The window `[Y - q, Y + q]`, open for `properly within`, with `starts`/`ends` and a trailing keyword |
| Aggregates over FHIR values | `Min`/`Max`/`Sum`/`Avg` over unconverted FHIR values | CQF's element-wise lift, `X return FHIRHelpers.To<T>(X)`, for the aggregates overloaded per System type; `Count`, `distinct`, `First` are generic and untouched |
| `Avg` over FHIR `Quantity` | Widened to `Decimal` | The FHIR lift takes precedence over decimal widening |
| An interval compared with a point | No promotion | `if X is null then null else Interval[X, X]`, on whichever side the point is (`SameAs` and `Meets` have no promotion; CQF rejects them) |
| `includes start Y` | `Includes` against a point | `Contains`, the mirror of `included in` becoming `In` |
| Trailing `start` / `end` keyword | No locator on the synthesized boundary | Attributed to the keyword |
| Empty `annotation: []` / `t: []` containers | Omitted on eight node kinds; the harness stripped them from both sides to compensate, and 152 of 345 runs depended on it | Emitted on every Element in CQF mode by one fill pass; the harness now compares them, and only the bundle path (whose JAXB writer omits them) reduces them |

`Property → ExpressionRef` (22 in Part 4) did not reproduce synthetically: tracked as
[#7](https://github.com/ArtnerC/echo-elm/issues/7). Emitting `CqlToElmError` diagnostics is
tracked as [#8](https://github.com/ArtnerC/echo-elm/issues/8).

### 6.4 On the bundle's reference ELM

`parity --bundle` now prints a warning whenever a reference library declares no
`translatorVersion`, or declares one other than the pin — a cql-to-elm compile always
records it, so its absence is the fastest signal that the reference was not produced by
the translator this repository targets.

The supplier-side action is outside this repository: the bundles' embedded ELM does not
match a CQF 5.0.0 compile of their own CQL, independent of echo-elm. Tracked as
[#5](https://github.com/ArtnerC/echo-elm/issues/5) so it can be raised with the bundle's publisher.

### 6.5 What to send back

Tracked as [#6](https://github.com/ArtnerC/echo-elm/issues/6).


Re-run the three-way comparison (fresh-CQF vs echo-elm) on this branch and send the
node-substitution table plus `StructuralDiff` output for the first ten differing
libraries. Every substitution in Part 4 should now be gone except the two shapes in 6.3;
anything else is new and should come with a construct description, not measure text.

---

## Status

| Item | Status |
|---|---|
| Bundle rerun (4 bundles, 150 library-comparisons) | Done — see Part 1 |
| Root-cause: bundle-ref vs fresh-CQF 5.0.0 divergence | Confirmed — see Part 2; `--bundle` now warns (6.4) |
| `context` bucket | Closed as non-defect — see Part 2 |
| `If`→`Or` divergence | Closed as non-defect — see Part 3 |
| Real residual gap counts, staleness excluded | Measured — see Part 4; resolved in Part 6 |
| `As`/`Null` (4.1) | **Fixed** — misdiagnosed, see 6.1 |
| `In`/`IncludedIn` (4.2) | **Reproduced and fixed** — see 6.1 |
| Timing phrases (`on or`, `starts`/`ends`, offsets) | **Fixed** — new, correctness-affecting; see 6.2 |
| Qualification family (`TypeSpecifier` substitutions) | **Fixed** — choice types, FHIRHelpers return types; see 6.2 |
| Conversion-site family | **Fixed**, including the aggregate list-lift; see 6.3 |
| Corpus claim | 332/335, 0 differ, before and after — see 6.1 |

## Actions

- [x] Recompile bundle source with a fresh, pinned CQF 5.0.0 CLI invocation and compare
      three-way (fresh-CQF, bundle-ref, echo) instead of two-way (bundle-ref, echo only)
- [x] Root-cause the `context` bucket — confirmed bundle-ref staleness, not an echo bug
- [x] Root-cause the `If`→`Or` divergence — same root cause, closed as non-defect
- [x] Re-aggregate node-type substitution buckets on the staleness-free (fresh-CQF vs
      echo) comparison to get accurate remaining-gap counts
- [x] Build a targeted reproducer for the `In`→`IncludedIn` substitution (4.2) —
      reproduced with `during` and a `Start`/`End` left operand, and fixed
- [x] Fix the `As`/`Null` gap (4.1) — fixed, and its diagnosis corrected
- [x] Make `--bundle` warn when a reference ELM's `CqlToElmInfo` lacks `translatorVersion`
- [x] Communicate to whoever supplies these bundles that their embedded reference ELM does
      not match a 5.0.0 compile of their own source — filed as #5 (needs the publisher)
- [x] Aggregate list-lift and interval-vs-point promotion — fixed, with `within`,
      trailing keywords and `includes` against a point; see 6.3
