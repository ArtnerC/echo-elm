# Issue: Post-Main Follow-Up Inputs After PR #4

**Priority:** High
**Labels:** parity, elm, bundle, diagnostics, follow-up
**Related:** PR #4, issues #5–#8, `06-bundle-recheck-stale-reference-and-remaining-gaps.md`

---

## Summary

PR #3 and PR #4 are merged to `main`. PR #4 explicitly worked through every remaining
translator item from issue 06 using synthetic CQF 5.0.0 reproductions under the
measure-bundle option profile. It added persistent `measure-shapes/` fixtures for the
fixed shapes and implemented a bundle-reference version warning.

Do not reopen these completed issue-06 items without new evidence from a fresh measurement:

- `context` and `If` versus `Or` differences were stale bundle-reference ELM artifacts,
  not echo-elm translator defects.
- Typed-null handling, `In` versus `IncludedIn`, type qualification, conversion-site
  selection, timing/operator selection, and nullary date/time operators were addressed
  in PR #4 and pinned by synthetic fixtures.
- `parity --bundle` now warns when a bundle reference ELM has no `translatorVersion` or
  declares a version other than the pinned CQF target.

The remaining work is intentionally limited to the four GitHub issues below. None can be
closed from repository-only evidence.

| GitHub issue | Owner of missing input | Blocking input |
|---|---|---|
| [#5](https://github.com/ArtnerC/echo-elm/issues/5) | Bundle publisher | Recompiled embedded reference ELM with an explicit translator version |
| [#6](https://github.com/ArtnerC/echo-elm/issues/6) | Licensed HEDIS-content holder | Fresh main-branch measurements and structural diffs |
| [#7](https://github.com/ArtnerC/echo-elm/issues/7) | Licensed HEDIS-content holder | A licence-clean description or structural-diff path for the unreproduced Property → ExpressionRef shape |
| [#8](https://github.com/ArtnerC/echo-elm/issues/8) | echo-elm maintainers after diagnostics census | Raw CQF diagnostic census and opt-in diagnostic parity fixtures |

---

## Part 1 — Bundle publisher remediation (#5)

**Finding preserved from issue 06:** published HEDIS 2025.2.1 bundles embed reference
ELM that does not equal a fresh compile of the same CQL with the bundle's declared
options and pinned CQF 5.0.0 target. On the largest measured bundle, the `context` field
matched fresh CQF versus echo-elm in `965 / 965` statements, while fresh CQF versus the
bundle reference matched only `300 / 965`.

The bundle reference did not carry a `translatorVersion` in `CqlToElmInfo`; fresh CQF
5.0.0 output does. It also retained the older `If`-wrapped condition shape where fresh
CQF and echo-elm emitted `Or`.

**What echo-elm has done:** PR #4 added the warning in the `parity --bundle` path. Echo
cannot repair a third-party bundle's embedded ELM.

**Needed from the publisher:**

1. Recompile the bundle CQL with the translator version intended for certification.
2. Preserve `translatorVersion` in each reference library's `CqlToElmInfo` annotation.
3. Republish the aggregate bundle with the new reference ELM.
4. Supply the resulting bundle version or a non-licensed metadata confirmation so the
   `parity --bundle` warning can be verified to disappear.

---

## Part 2 — Fresh measurement against merged main (#6)

The previous HEDIS re-measurement predates PR #4. Its raw bundle-reference match count
must not be used as a completion criterion because it mixes echo output with stale,
externally-produced reference ELM.

**Run from `main`, with licensed bundle access:**

```powershell
.\tools\echo-elm.exe parity --bundle <bundle.json> --cqf-version 5.0.0
```

Also conduct the three-way comparison used in issue 06:

1. fresh CQF 5.0.0 compile of bundle CQL;
2. echo-elm output from current `main`;
3. bundle-embedded reference ELM, reported separately and excluded as a translator
   oracle when its version warning is present.

**Return only licence-clean artifacts:**

- The aggregate node-substitution table, in issue-06 Part 4 format.
- The first ten `StructuralDiff` entries with definition names and structural paths.
- For each unfamiliar difference, one sentence describing the CQL construct without
  measure text, code systems, valuesets, or patient content.
- Whether `parity --bundle` emitted a stale-reference warning and the annotation metadata
  that caused it (`translatorVersion` presence/value and option string only).

This evidence both validates PR #4 on real content and tells us whether any new,
previously unrepresented construct remains.

### September 14, 2026 local re-measurement

The licensed local 2025.2.1 details bundles were re-run after PR #4 from the
current `main` code state. This was a **fresh-CQF comparison**, not the
`parity --bundle` embedded-reference comparison:

1. Every `Library.content[text/cql]` attachment was extracted into a temporary,
  local-only corpus.
2. CQFramework `cql-to-elm-cli` **5.0.0** compiled each extracted library with
  `--locators --result-types --disable-list-traversal
  --disable-list-demotion --disable-list-promotion`.
3. echo-elm translated the identical extracted source with the corresponding
  `measure-bundle` profile, resolving includes from the same temporary corpus.

The raw CQL, ELM, extracted corpora, and detailed reports remain outside this
repository under `C:\Dev\_scratch\echo-elm-parity-rerun\fresh-cqf\`.
Do not commit them or copy measure text into issues or fixtures.

| Bundle | Libraries | Match | Differ | Errors |
|---|---:|---:|---:|---:|
| AHU Details | 45 | 11 | 34 | 0 |
| BCSE Details | 43 | 11 | 32 | 0 |
| DDE Details | 36 | 8 | 28 | 0 |
| ENP Details | 12 | 2 | 10 | 0 |
| GSD Details | 38 | 9 | 29 | 0 |
| KED Details | 39 | 9 | 30 | 0 |
| LDM Details | 14 | 3 | 11 | 0 |
| PCR Details | 47 | 11 | 36 | 0 |
| RDM Details | 14 | 3 | 11 | 0 |
| SNSE Details | 37 | 8 | 29 | 0 |
| SPC Details | 44 | 11 | 33 | 0 |
| W30 Details | 38 | 9 | 29 | 0 |
| **Total** | **407** | **95** | **312** | **0** |

#### Fresh-CQF findings

This eliminates stale bundled reference ELM as an explanation for the remaining
differences. The completed reports show three recurring, actionable families:

1. **Cross-library type propagation:** CQF retains concrete FHIR element types,
  `ListTypeSpecifier`s, and `libraryName` qualifiers through included-library
  references where echo-elm often falls back to `Any` or omits the qualifier.
  The AHU and BCSE capped structural reports alone contain 433 missing
  `resultTypeName` entries, 378 missing `resultTypeSpecifier` entries, and 139
  missing `libraryName` entries.
2. **Locator and parameter metadata:** CQF emits locators on parameter default
  expressions, with a smaller set of associated annotations that echo-elm does
  not yet retain.
3. **Non-metadata lowering differences:** representative reports contain
  `Concatenate` versus `Add` and conversion-wrapper differences (for example,
  an extra `ToDateTime`). These remain semantic-shape investigations; they must
  not be normalized away as result-type noise.

The zero-error result across 407 fresh-CQF compilations is important: this is
not a parser or include-resolution failure. The next echo-elm work should start
with a license-clean synthetic reproduction of cross-library result-type and
`libraryName` propagation, then separately reproduce the node-lowering shapes.
The retained `report.json` files contain the first ten structural paths per
library without requiring licensed body text in this repository.

---

### Resolution of the fresh-CQF families

All three families reproduced synthetically and are fixed. Each is pinned by a fixture
under `test/corpus/cqframework/layered/`, a directory that carries a resolvable
FHIRHelpers because the layered-include behavior only shows up when every include in the
chain resolves, as in a real bundle.

| Family | Root cause | Fixture |
|---|---|---|
| 1. Cross-library types and `libraryName` | An included library's types were computed with no library source ("one level is enough"), so anything two include levels deep came out `List<Any>` | `LayeredTop`, `LayeredMain`, `Mid`, `Common`, `Helpers` |
| 1. Choice-typed arguments | An argument to a choice parameter was not cast to the declared choice | `LayeredTop` |
| 2. Parameter-default locators | Cleared unconditionally; only the Date-bound `Interval<DateTime>` default CQF rewrites is emitted bare | `ParameterDefaults` |
| 3. `Concatenate` vs `Add` | `+` chose `Concatenate` only for string *literals*; now by type. `&` also lacked its `Coalesce(x, '')` wrapping — a wrong-results bug | `LayeredTop` |
| 3. Extra / missing `ToDateTime` | `AgeInYearsAt(<Date>)` converted the birth date regardless; a `Date` meeting a `DateTime` in membership, lists, branches and interval bounds was not converted | `DateTimeMixing` |

Found on the way: `X is null` converted a FHIR value before testing it, and a sort-by
identifier naming a FHIR field skipped conversion (`sort by start of period`).

Also found by rechecking earlier gaps: implicit list traversal was not implemented, and a
list of codes tested against a value set emitted `InValueSet` instead of `AnyInValueSet`
(the latter applies under the measure profile). Both are fixed (`ListTraversal`,
`AnyInValueSet`).

Re-measuring the licensed bundles against the new `main` is tracked in
[#6](https://github.com/ArtnerC/echo-elm/issues/6).

---

## Part 3 — Property → ExpressionRef needs a trigger (#7)

The issue-06 survey found 22 locations where fresh CQF 5.0.0 emitted a `Property` and
echo-elm emitted an `ExpressionRef` at the corresponding structural path. PR #4's
synthetic work exercised property navigation on defines, parameters, query aliases,
context references, cross-library references, conversion sites, and FHIRHelpers calls.
Those paths now match their CQF fixtures.

The remaining substitution did not reproduce synthetically. Do not infer a translator
fix from its aggregate count alone.

**Needed to make this actionable:** one of the following, without licensed CQL text:

- a one-sentence construct description, such as "property navigation on a tuple returned
  by an included library definition"; or
- a `StructuralDiff` excerpt containing only the fixture/library identifier, definition
  name, and structural path.

That is enough to add a small fixture under
`test\corpus\cqframework\measure-shapes\`, establish whether the difference remains on
`main`, and implement a bounded fix if one is needed.

---

## Part 4 — CQF diagnostic annotation parity (#8)

The parity harness intentionally removes CQF `CqlToElmError` annotations before comparing
ELM. echo-elm does not yet emit these annotations. This makes the existing corpus parity
number unsuitable as evidence for diagnostic parity, by design.

The high-volume observed diagnostic is the CQF warning for an overloaded function call
when `SignatureLevel` is `None`. A terminology-membership resolution warning was also
observed. The final diagnostic surface must agree on:

- message text;
- severity and error type;
- source locator fields;
- occurrence and ordering.

**Required next work:**

1. Run the pinned CQF CLI without diagnostic stripping across the corpus and census each
   distinct annotation shape.
2. Add an opt-in diagnostics comparison mode to `corpus.yaml`, so individual diagnostic
   kinds can be verified before the global normalizer is removed.
3. Implement and fixture the overload-signature warning first, including function
   overload discovery and call-site locator.
4. Add each remaining diagnostic kind from the census, then remove the global exclusion
   only when all opted-in diagnostic fixtures pass.

This is independent of the measure-shape fixes in PR #4 and should remain a separate
workstream.

### Progress

The raw census found three diagnostic kinds across the corpus: the overload warning at
`SignatureLevel` None, "List-valued expression was demoted to a singleton.", and an
operand-hiding warning that only occurs inside FHIRHelpers itself. echo-elm now emits the
first two, placed after `CqlToElmInfo` as CQF places them. Fixtures opt in with
`diagnostics: true` in `corpus.yaml`, and eight are opted in.

The global exclusion stays. For the remaining fixtures, CQF warns twice at sites it
resolves twice (aggregate lifts, and `during` / `included in` on a converted operand), and
reproducing that means mirroring CQF's resolution attempts. Details are on
[#8](https://github.com/ArtnerC/echo-elm/issues/8).

---

## Status

| Item | Status |
|---|---|
| PR #3 measure-shape fixes | Merged to `main` |
| PR #4 issue-06 fixes and stale-reference warning | Merged to `main` |
| Bundle publisher remediation | External dependency — #5 open |
| Fresh main-branch HEDIS re-measurement | Complete: 407 fresh-CQF comparisons, 95 match / 312 differ / 0 errors — #6 open for remediation |
| Property → ExpressionRef trigger | Waiting for licence-clean evidence — #7 open |
| Fresh-CQF families 1–3 | **Fixed**, with synthetic fixtures under `layered/` — see Part 2 |
| CQF diagnostic annotation parity | Census done; two of three kinds emitted; eight fixtures opted in — #8 open for the double-resolution counts |

## Actions

- [x] Confirm PR #3 and PR #4 merged to `main`.
- [x] Remove completed issue-06 gaps from the follow-up scope.
- [x] Record the four GitHub issues that remain open and their required inputs.
- [ ] Obtain republished bundle reference ELM metadata from the publisher (#5).
- [x] Re-measure licensed bundles from `main` and return only structural evidence (#6).
- [x] Add a licence-clean synthetic fixture for cross-library result type and `libraryName`
  propagation, then compare it to CQF 5.0.0 (#6).
- [x] Add separate synthetic fixtures for `Concatenate` versus `Add` and conversion-wrapper
  lowering after the cross-library metadata fixture is resolved (#6).
- [ ] Provide a licence-clean trigger description or `StructuralDiff` path for
      Property → ExpressionRef (#7).
- [x] Census raw CQF diagnostics and add opt-in diagnostic parity fixtures (#8).
- [ ] Re-measure the licensed bundles against the new `main` (#6).
