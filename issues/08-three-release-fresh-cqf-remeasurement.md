# Issue: Three-Release Fresh-CQF Remeasurement, Deduplicated Families, and Synthetic Repros

**Priority:** High
**Labels:** parity, elm, measure-bundle, remeasurement, repro
**Related:** PR #14, issues #5, #7, #8, #9, #11, #12, #13, #15, `07-post-main-follow-up-inputs.md`
**Repros:** moved to `test/corpus/cqframework/measure-families/` — see [`08-repros/`](08-repros/README.md) (licence-free, 21 fixtures, CQF 5.0.0 goldens)
**Appendix:** [`08-first-differences.md`](08-first-differences.md) (#11 per-bundle first differences)

---

## Summary

The merged HEDIS measure bundles for 2025.2.1, 2025.3.0 and 2026.1.0 were remeasured with
PR #14 head (`41c1a46`) against a **fresh CQF 5.0.0 compile** of the same CQL (`--fresh-cqf`).
Embedded bundle ELM was not used as the oracle (see #5).

Every remaining difference was grouped into a deduplicated family. Each family has a
counted site census per release, a severity, a fix pointer, and, where a shape could be
isolated, a licence-free fixture in `08-repros/` that reproduces it against stored CQF 5.0.0
goldens.

Headline:

- The two 2025 releases are almost identical: 103 of 382 libraries match and 279 differ.
- **2026.1.0 is blocked by an input problem, not a translator problem.** 79 CQL libraries in
  the 2026.1.0 package start with a UTF-8 byte-order mark. CQF 5.0.0 and echo-elm both reject
  them, with a syntax error at `1:0`, and every library that includes one fails with it. See
  [Part 2](#part-2--2026.1.0-utf-8-bom-input-defect). With the BOM stripped, 109 of 407 match,
  284 differ and 14 are CQF errors.
- **The 14 remaining 2026.1.0 errors are a CQF 5.0.0 diagnostic, not an echo-elm failure.**
  CQF reports `Flatten(list<System.Any>)` as ambiguous once two other `Flatten` generic
  instantiations already exist in the library. One shared library hits it, plus 13 libraries
  that include it. echo-elm compiles the same content. See family W15 and #8/#9 in Part 6.
- In 2025.x measure content, eight families can change evaluation results: #7, the
  sort-implicit receiver, backbone element types, #13, `InValueSet`/`ExpandValueSet`,
  `CalculateAgeAt`, implicit conversions (missing and extra), and Integer→Decimal promotion.
  Fix backbone types and #13 first. Together they cause most of the type-cascade
  differences.

All counts below are structural: library and definition names, ELM node types, paths and
counts. No measure CQL, ELM, value set or code system identifiers are reproduced.

---

## Part 1 — Method and per-release results

| Item | Value |
|---|---|
| echo-elm | PR #14 head `41c1a46` (`echo-elm-pr14.exe`) |
| Reference | Fresh CQF `cql-to-elm-cli` 5.0.0, measure-bundle option profile |
| Input | One merged bundle per release (all measure bundles, de-duplicated by library name and version) |
| Command | `echo-elm parity --bundle merged-<rel>.bundle.json --fresh-cqf --tools-dir tools --jobs 12 --run-id pr14-<rel> --out runs/pr14-<rel>` (run from the repo root) |

| Release | Libraries | Match | Differ | Error | Notes |
|---|---:|---:|---:|---:|---|
| 2025.2.1 | 382 | 103 | 279 | 0 | |
| 2025.3.0 | 382 | 103 | 279 | 0 | |
| 2026.1.0 (as published) | 407 | 30 | 64 | 313 | 308 `Syntax error` at `1:0` (BOM); 5 CQF ambiguity errors |
| 2026.1.0 (BOM stripped) | 407 | 109 | 284 | 14 | All 14 are CQF `Flatten(list<System.Any>)` ambiguity errors (W15) |

---

## Part 2 — 2026.1.0 UTF-8 BOM input defect

**Finding:** 79 of 407 `Library` resources in the 2026.1.0 package carry `text/cql` content whose
decoded bytes start with `EF BB BF`. 324 have no BOM, three start with `CRLF`, and one
starts with a comment. Neither 2025 package has this.

| Translator | Behaviour on a BOM-prefixed library | Exit code |
|---|---|---|
| CQF 5.0.0 | `Translation failed due to errors: Error:[1:0, 1:0] Syntax error` | 0 |
| echo-elm `41c1a46` | `error:[n/a] token recognition error at: '\uFEFF'`, then `Translation failed due to errors:` | 0 |

Both tools reject the BOM and both exit 0, so echo-elm is CQF-compatible here. The exit code
is not an echo-elm bug. But in the 2026.1.0 run, 308 libraries were reported as
`upstream-error` and were never compared: the 79 BOM libraries plus every library that
includes one of them, directly or transitively.

Five further libraries fail in CQF with `Call to operator Flatten(list<System.Any>) is
ambiguous with: ...`. With the BOM stripped, the number of libraries with this error rises to
14: one shared library plus 13 libraries that include it, directly or transitively. It is a
genuine CQF diagnostic on 2026.1.0 content (family W15; #8 and #9 in Part 6).

Probe (licence-free): write `library BomLib version '1.0'` / `define "One": 1` as UTF-8 *with*
BOM and translate with each tool.

**Recommended echo-elm changes:**

1. `parity --bundle`: after base64-decoding `Library.content`, detect a leading UTF-8 BOM.
   Either strip it before writing the CQL handed to both translators, or report the
   library as a distinct `input-error` status instead of `upstream-error`. Today a
   publisher-side encoding problem looks like 308 translator failures.
2. Optional: an `--strip-bom` (or default-on) normalization in `cqf translate`. Keep the
   CQF-compatible error as the default if strict CLI parity matters more.
3. Publisher (#5 owner): re-export the 2026.1.0 CQL without BOMs.

The BOM-stripped remeasurement row in Part 1 was produced by stripping only those three bytes
from a scratch copy of the merged bundle. The licensed source package was not modified.

---

## Part 3 — Deduplicated difference families

Site counts are ELM nodes at which the fresh CQF and echo-elm trees first diverge, per
family, from the `Diff` lines of the parity report plus a direct ELM walk where the 40-line
`Diff` cap truncated output. "Libs" is the number of libraries with at least one site.

### 3a. Wrong results (fix first)

| # | Family | 2025.2.1 sites / libs | 2025.3.0 sites / libs | 2026.1.0 (BOM-stripped) | Repro fixture | Fix pointer |
|---|---|---|---|---|---|---|
| W1 | **#7** unparenthesized `Alias.field Y` query/`with` source emitted as `ExpressionRef{libraryName: Alias, name: field}` instead of `Property{scope/source, path}` | `Query` 7 (5) + `With` 13 (3) | same | `Query` 15 (6) + `With` 13 (3) | `QualifiedQuerySource` | Query-source parser: an identifier qualified by an in-scope alias or operand is a property path, not a library reference. |
| W2 | Sort implicit receiver: fluent call with no receiver in `sort by` gets no operand and no `libraryName` | 32 (10) | 32 (10) | 40 (14) | `SortImplicitReceiver` (+ `SortFluentLib`) | Sort-item resolution must bind `$this` as the implicit fluent receiver. |
| W3 | Backbone element types shortened (`FHIR.Claim.Item` → `{http://hl7.org/fhir}Item`). Overload keys collide, element types fall to `Any`, implicit `ToInteger` is dropped | 830 (17) | 830 (17) | 932 (47) | `BackboneOperand` | Model-info type naming: keep the full dotted backbone name in `QName`s and overload signatures. |
| W4 | **#13** fluent function declared in multiple includes: `FunctionRef.libraryName` missing, no result type | 9756 (213) | 9811 (213) | 9630 (220) | `FluentOverloadMain` (+ `FluentClaims`, `FluentEncounters`) | 99.7 % of sites are name collisions across includes. Resolve by receiver type across all includes. Many collisions are backbone-typed, so fix W3 first. |
| W5 | `c in <ValueSet operand>` emitted as `In` instead of `InValueSet{code, valuesetExpression}` | 9 (4) | 9 (4) | 9 (4) | `ValueSetMembership` | Operator resolution: a `System.ValueSet`-typed right operand selects `InValueSet` (expression form). |
| W6 | Value set union not wrapped in `ExpandValueSet` | 110 (2) | 110 (2) | 161 (2) | `ValueSetMembership` | Implicit `ValueSet`→`List<Code>` conversion when a value set is used as a list. |
| W7 | `CalculateAgeIn{Years,Months,…}At(...)` left as an unresolved `FunctionRef` (no `libraryName`) instead of `CalculateAgeAt{precision}` | 10 (3) | 10 (3) | 4 (1) | `AgeFunctions` | System-function table: map the `CalculateAgeIn<Precision>At` overloads with explicit birth date. |
| W8 | Implicit FHIRHelpers conversion **missing** in echo (`ToDecimal` 94, `ToString` 64, `ToDate` 24, `ToDateTime` 13, `ToInterval` 6, `ToQuantity` 5) | 209 (29) | 209 (29) | 126 (32) | `ImplicitConversions` ("Claim ids equal") | Conversion insertion for FHIR-primitive vs System comparisons (`id = 'x'`, etc.). |
| W9 | Implicit FHIRHelpers conversion **extra** in echo: `ToCode(AliasRef)` where the alias is already `Code`/`Coding` (112), `ToConcept(Property CodeableConcept)` (25), `ToInteger(positiveInt)` (12), and tuple-element conversions | 172 (15) | 172 (15) | 104 (15) | `ImplicitConversions` ("Coding alias equal", "Claim tuple") | Do not convert an operand whose type already satisfies the signature. Tuple elements keep FHIR types in CQF. |
| W10 | Integer→Decimal promotion (`ToDecimal`) missing in `if` branches, function arguments, comparisons and `Max`/`Min` list elements | 763 (28) | 799 (30) | 795 (36) | `DecimalPromotion` | Apply implicit Integer→Decimal promotion at overload selection and at list/`if` type unification. 561 sites sit under `FunctionRef < List < Flatten`. |
| W11 | `ToDecimal` applied at a different node (echo types `Add` of integers as Integer and wraps the sum) | 46 (12) | 46 (12) | 46 (12) | `DecimalPromotion` ("Sum of weights" control) | Same unification point as W10. |
| W12 | `: external` functions lose `external: true` and get a `Null` body | 0 in measure bundles (jar FHIRHelpers only) | 0 | 0 | `ExternalFunction`, `FHIRHelpers` | `FunctionDef` serializer. |
| W13 | Sort double conversion `ToDateTime(ToDateTime(x))` | 0 (synthetic only) | 0 | 0 | `SortByFhirField` | Sort-item resolution. |
| W14 | Inline `Code 'x' from "System"` keeps quotes in `CodeSystemRef.name` (`"\"System\""`) and has no locator or result type | 0 (synthetic only) | 0 | 0 | `ImplicitConversions` ("Coding alias equal") | Strip the quoted identifier in the `codeSelector` system reference. |
| W15 | **Mixed-element `flatten`**, where the inner lists have different element types. (a) When it compiles, CQF types the operand as a `Query` over `List<Any>` (source, `let`, `relationship`, untyped `return`); echo-elm emits a plain `List` typed `List<Integer>` (or the first element type). (b) When two other `Flatten` instantiations already exist in the same library, CQF reports `Flatten(list<System.Any>)` as **ambiguous** and fails; echo-elm compiles. | 1 site (1) | 1 site (1) | 3 sites (3) + **14 libs CQF error** | `FlattenOneInstance`, `FlattenMixedOnly` (a); `FlattenAmbiguity` (b, `expectedStatus: upstream-error`) | (a) List-literal type unification: with no common element type, unify to `List<Any>` and keep the CQF operand shape. (b) Decide on CQF bug-compatibility: either reproduce the instantiation-order ambiguity as an error (strict parity, #8) or document it as a deliberate divergence. CQF counts generic `Flatten` instantiations accumulated earlier in the library (#9 state dependence). |

### 3b. Type cascade (mostly fixed by W3, W4 and W10)

| # | Family | 2025.2.1 | 2025.3.0 | 2026.1.0 (BOM-stripped) | Repro | Notes |
|---|---|---|---|---|---|---|
| T1 | **#15** `.value` on FHIR primitives untyped or `Any` (echo typed / CQF typed) | 124 / 788 | 125 / 866 | 124 / 995 | `PrimitiveValue`, `FHIRHelpers` | Type `.value` with the System type from model info. |
| T2 | Typed null (`As{Null}` type differs) | 684 | 676 | 1009 | — | Mostly cascade from T1/W4. 23 sites not reproduced in isolation. |
| T3 | Missing `resultTypeName`/`resultTypeSpecifier` | 37008 | 37307 | 39769 | — | Cascade. Remeasure after W3/W4/W10. |
| T4 | Point `In` emitted as `IncludedIn` | 6 (5) | 6 (5) | 15 (6) | — | Cascade from an untyped alias (W4/T1). |
| T5 | **#15** `Message` in a `case` branch not wrapped in `As` | 3 | 3 | 3 | `FHIRHelpers` | |

### 3c. Metadata only

| # | Family | 2025.2.1 | 2025.3.0 | 2026.1.0 (BOM-stripped) | Repro | Notes |
|---|---|---|---|---|---|---|
| M1 | Documentation tags on `parameter` not emitted as `Annotation` | 306 (78) | 306 (78) | 453 (80) | `ParameterTags` | |
| M2 | Sort `ByColumn` / `ByExpression` missing result type | 34 / 34 | 38 / 34 | 56 / 38 | `SortByFhirField`, `FluentClaims` | |
| M3 | n-ary `union` grouped differently (left vs right association) | 12 (12) | 12 (12) | 3 (3) | — | Semantically equivalent. Low priority. |
| M4 | Definitions present on only one side (`def-only-cqf` / `def-only-echo`) | 200 / 170 | 200 / 170 | 202 / 169 | — | Overload pairing artefact: the harness pairs overloads positionally, so W3/W4 signature differences show up as unpaired definitions. |

### 3d. Closed or not reproduced

| Item | Result |
|---|---|
| **#12** `context` differences | 0 in all compared libraries. Close. |
| Multi-argument fluent positional shifts (e.g. `"Starts during"`, `"With continuous enrollment during"`) | All are W4 (#13) cascades. Not a separate family. |
| Typed-null residual | 23 sites in 2025.x not reproduced by any synthetic shape tried. Remeasure after W3/W4/T1. |

---

## Part 4 — Recommended fix order

1. **W3 backbone type names.** This is the root of overload collisions and `Any` element types,
   and of many #13 sites.
2. **W4 (#13) fluent resolution across includes.** It is the largest family and drives T3.
3. **T1 `.value` typing**, then **W8/W9/W10/W11 conversions and promotion.** These
   share the conversion-selection code path.
4. **W1 (#7), W2, W5/W6, W7, W15(a).** These are small, isolated fixes, each with a fixture.
   Decide the W15(b) policy (strict CQF error parity vs documented divergence) in the same
   step. It decides whether 14 libraries in 2026.1.0 can ever count as `match`.
5. Metadata (M1, M2, M3), W12–W14.
6. **Remeasure** all three releases (2026.1.0 BOM-stripped or republished), then move each fixed
   fixture from `08-repros/` into `test/corpus/cqframework/measure-shapes/`.

---

## Part 5 — Node-substitution table (#11 data)

Top substitutions at the first divergent node, 2025.2.1, sorted by count. "cqf X vs echo Y"
means CQF emitted node type X where echo-elm emitted Y.

| Substitution | Count |
|---|---:|
| cqf `ListTypeSpecifier` vs echo `NamedTypeSpecifier` | 1187 |
| cqf `IntervalTypeSpecifier` vs echo `NamedTypeSpecifier` | 929 |
| cqf `ToDecimal(Literal)` vs echo `Literal` | 795 |
| `Null` inside `As` with different type | 684 |
| cqf wraps `Property` in `FunctionRef` (implicit conversion) | 365 |
| cqf `TupleTypeSpecifier` vs echo `NamedTypeSpecifier` | 269 |
| echo wraps `AliasRef` in `FunctionRef` (extra conversion) | 117 |
| cqf `IntervalTypeSpecifier` vs echo `ListTypeSpecifier` | 112 |
| cqf `ExpandValueSet` vs echo union | 110 |
| echo wraps `Property` in `FunctionRef` | 48 |
| cqf wraps `FunctionRef` in `As` | 35 |
| echo wraps `Property` in `ToDateTime` | 34 |
| echo wraps `Add` in `ToDecimal` | 28 |
| cqf `NamedTypeSpecifier` vs echo `ListTypeSpecifier` | 28 |
| cqf `Property` vs echo `ExpressionRef` (#7) | 20 |

Type-name census from `Diff` messages (2025.2.1): `Date`→`Any` 638, `Date`→FHIR `date` 193,
`Claim`→`Any` 141, `Decimal`→`Integer` 63, `System.ValueSet` vs `FHIR.ValueSet` 40.

The per-bundle first-difference appendix is in
[`08-first-differences.md`](08-first-differences.md).

---

## Part 6 — Answers to open issues

| Issue | Answer |
|---|---|
| **#5** bundle reference ELM / `translatorVersion` | Still applies. The embedded reference ELM in all three packages has no `translatorVersion`. This remeasurement therefore used `--fresh-cqf` only. New publisher ask: 2026.1.0 also needs re-export without UTF-8 BOMs (Part 2). |
| **#7** `Property` → `ExpressionRef` | **Trigger found.** An unparenthesized `Alias.field Y` (or `operand.field Y`) used as a query or `with` source. `let` and parenthesized sources match. 20 sites in 5+3 libraries per 2025 release, and 28 sites in 6+3 libraries in 2026.1.0. Licence-free repro: `08-repros/QualifiedQuerySource.cql`. |
| **#8** diagnostic parity | One concrete missing diagnostic: a fluent function called with ordinary call syntax. CQF rejects it ("…is a fluent function and can only be invoked with fluent syntax"); echo-elm accepts it. Repro: `08-repros/FluentNonFluentCall.cql` (`expectedStatus: upstream-error`). 2026.1.0 adds a second, real-content case. CQF rejects a mixed-element `flatten` as `Flatten(list<System.Any>)` ambiguous in 14 libraries (BOM-stripped run; one shared library plus 13 dependents). echo-elm accepts all 14. The error is state-dependent (see #9). Repros: `FlattenAmbiguity` (error), `FlattenOneInstance` and `FlattenMixedOnly` (compile controls). |
| **#9** `SignatureLevel Overloads` state dependence | Signature arrays are not exercised: the measure-bundle profile compiles with signature level `None`, so no `signature` arrays are emitted on either side in any release. **Operator-resolution state dependence is confirmed**, though, independent of signature level. In CQF 5.0.0, the generic `Flatten` instantiations already created in a library change how a later `Flatten(List<Any>)` call resolves. With one or zero prior instantiations it compiles; after `Flatten(List<List<Integer>>)` and `Flatten(List<List<String>>)` it is ambiguous. Measured impact: 14 libraries in 2026.1.0 (W15). Repros: `FlattenAmbiguity` / `FlattenOneInstance` / `FlattenMixedOnly` differ only in which `define`s precede the mixed call. |
| **#11** first-difference data | Delivered: the Part 5 substitution table plus the [`08-first-differences.md`](08-first-differences.md) appendix (per bundle, first ten structural `Diff` lines per library). |
| **#12** `context` | 0 differences in all compared libraries. Close. |
| **#13** fluent overloads across includes | Occurs at scale (W4): 9756 / 9811 / 9630 sites in 213 / 213 / 220 libraries. 99.7 % (2025.x) and 99.5 % (2026.1.0) are name collisions across includes. It depends on W3 backbone naming. Repro: `FluentOverloadMain` + `FluentClaims` + `FluentEncounters`. |
| **#15** FHIRHelpers / `.value` | Confirmed: T1 (`.value` typing), T5 (`Message` not in `As`), W12 (externals), plus the call-site shape of `ToQuantity`, `ToConcept` and `ToRatio`. Repros: `PrimitiveValue`, `FHIRHelpers`, `ExternalFunction`. |

---

## Part 7 — Harness notes

- **Run from the repository root.** The CQF launcher (`tools/cqframework/5.0.0/run.bat`) uses
  relative classpath entries.
- **The 40-line `Diff` cap** truncates large libraries. Family counts in this issue for the
  largest libraries come from a direct ELM walk of `UpstreamJSON`/`EchoJSON`, not from `Diff`
  alone.
- **Overload pairing is positional.** When signatures differ (W3/W4), overloads are paired
  wrongly and show up as M4 unpaired definitions. Pair by name plus operand count plus a
  best-effort operand-type match instead.
- **Suggested `known-gap` status.** Allow a corpus entry to declare `expectedStatus: differ`
  together with a family tag. Then `08-repros/` fixtures can live in `test/corpus` before
  they are fixed, without failing `TestGoldenCorpus`.
- **BOM / input errors.** See Part 2: separate input-encoding failures from upstream
  translator errors.

---

## Status (branch `feat/issue08-remeasure-families`)

Every family with a reproduction is fixed, and each fixture moved into the corpus with its fix.
Against the committed CQF 5.0.0 goldens: **`match=18 differ=0 divergent=1 error=0 total=21`**
(was `match=2 differ=17`). No pre-existing golden changed.

| Family | Outcome | Root cause, where it differed from the pointer above |
|---|---|---|
| W3 backbone types | **Fixed** | The parser kept only the first qualifier, so `FHIR.Claim.Item` became `FHIR.Item`. Inference also resolved only the last segment, which separately made `FHIR.Quantity` System's. |
| W4 #13 fluent across includes | **Fixed** | Operand and return types were recorded for unambiguous names only. Every overload is now recorded and calls select by argument type (exact, FHIR subtype, conversion); a tie stays unresolved. A fluent name in several includes picks its library from the receiver's type at translation time. |
| T1 `.value` | **Fixed** | The ModelInfo table omits System-typed elements; `.value` now takes the primitive's System type. |
| T5, #15 FHIRHelpers | **Fixed** | Case branches unify as CQF accumulates them: compatible types unify, otherwise a Choice; a non-matching single-type branch, or a `Message` over null, is cast; a sub-choice is not. Unqualified `Quantity`/`Ratio` resolve to FHIR with `using FHIR` (checked against the CLI). A library's own `ToQuantity(FHIR.Quantity)` wins over the System operator. |
| W8 missing conversions | **Fixed** | Property lookups did not walk `baseType`, so inherited elements (`id`) had no type and no conversion. |
| W9 extra conversions | **Fixed** | Equality between compatible FHIR types takes the generic `Equal` with no conversion; tuple elements are an unconsumed position. |
| W10/W11 Decimal promotion | **Fixed** | Numeric promotion at if/case branches, list elements, and arguments (against the selected overload). |
| W1 #7 `Alias.field Y` source | **Fixed** | The builder now knows the include set, so a dotted source whose first part is not an include is a property path. Singleton `with` sources now type their alias. |
| W2 sort implicit receiver | **Fixed** | A receiverless fluent call in a sort item binds `$this`, only when it cannot resolve as written. |
| W13 sort double conversion | **Fixed** | `sortFieldCoercion` read a flag `translateExprCore` had already cleared. |
| W5/W6 value sets | **Fixed** | `InValueSet{valuesetExpression}` for `System.ValueSet` operands; `ExpandValueSet` for value sets used as lists. `System.ValueSet` was being read as `FHIR.ValueSet`. |
| W7 `CalculateAgeIn*At` | **Fixed** | Mapped to `CalculateAgeAt` / `CalculateAge`. |
| W12 externals | **Fixed** | `external: true`, no body. |
| W14 inline code system | **Fixed** | Identifier unquoted; reference stamped as CQF does. |
| W15(a) mixed flatten | **Fixed** | List literals with no common element type are `List<Any>` (checked against the CLI for scalars and sub-lists); `Flatten` over `List<Any>` gets CQF's element-wise cast query. |
| W15(b) / #9 state dependence | **Deliberate divergence** | CQF resolves `Flatten(List<Any>)` against instantiations created earlier in the library: after one it casts `{'a'}` to `List<Integer>`, after two it errors. Reproducing that means emitting a wrong cast or rejecting valid CQL depending on unrelated definitions. `FlattenOneInstance` is marked `expectedStatus: divergent` with the reason; the harness reports it as `divergent` and fails if it ever starts matching. |
| #8 fluent called without fluent syntax | **Fixed** | Error with CQF's message verbatim, only when every declaration of that arity is fluent. |
| M1 parameter tags | **Fixed** | Tags on parameters and on the library header; multi-line values keep their own spacing. |
| M2 sort types | **Fixed** | `ByColumn` over a FHIR element carries the field's FHIR type. |
| M3 n-ary union grouping | Not addressed | No reproduction; semantically equivalent. |

Harness (Part 7):

- **BOM / input errors (Part 2):** `parity --bundle` strips a leading UTF-8 BOM from each
  library before either translator sees it and names the affected libraries on stderr as a
  publisher defect. Strict `cqf translate` keeps CQF's error.
- **Overload pairing:** `StructuralDiff` keys overloads by signature, then pairs leftovers by
  name and arity, so a signature difference (the old M4 shape) reads as one field difference.
- **Known-gap status:** `expectedStatus: divergent` plus a required `divergence` reason.

**Remeasure** with the measure bundles before scoping further work. T2–T4 (typed nulls, the
result-type cascade, point `In`) were expected to fall out of W3/W4/T1/W10 and have no
reproduction of their own; the next run will say how much is left.

Licensing: `08-first-differences.md` lists HEDIS library and definition names from licensed
content, so it is not committed (issue 05 Part 8).

---

## Part 8 — Reproduce

```powershell
# Repros, Java-free (committed CQF 5.0.0 goldens)
.\echo-elm.exe parity --tag measure-families --ref-dir test/goldens/cqf

# Licensed measure content (holder only), per release
.\echo-elm.exe parity --bundle <merged-<rel>.bundle.json> --fresh-cqf --tools-dir tools `
  --jobs 12 --run-id <rel> --out parity/runs/<rel>
```
