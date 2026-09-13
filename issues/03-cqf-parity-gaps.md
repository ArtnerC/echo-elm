# Issue: CQF Parity Gaps — ELM Node Emission and Operator Translation Differences

**Priority:** High
**Labels:** parity, elm-emission, translator, spec-conformance, firely
**Status:** Verified and largely resolved — see "Verification Results" below.

---

## Summary

A comparison of `echo-elm` output against ELM embedded in HEDIS FHIR measure bundles (compiled by the CQF `cql-to-elm` 4.8.x toolchain) reported systematic differences in both ELM structure and expression-node translation, with **0/383 (2025.1.0) and 0/382 (2025.2.1) libraries matching** after stripping volatile fields.

Every claim below has since been checked against ground truth produced by running the pinned CQF `cql-to-elm` CLI (3.29.0 and 4.8.0) on minimal fixtures. **The original analysis used the wrong baseline for Part A and most of Part B**, and the actual root cause of the 0/383 result was different from — and much simpler than — what was reported. This document has been rewritten to record what was verified, what was fixed, and what remains open.

Regression fixtures for everything confirmed here live in `test/corpus/cqframework/elm-nodes/`, with CQF reference goldens under `test/goldens/cqf/<profile>/elm-nodes/`.

---

## Verification Results

### Root cause of 0/383: library includes were never resolved

`echo-elm translate` never configured a `LibrarySource`, so `include` declarations resolved to nothing. Every HEDIS measure library includes a shared terminology library, and each qualified reference into it (`Terminology."Diabetes VS"`) silently degraded to a `Property` access on an `ExpressionRef` instead of the typed `ValueSetRef` / `CodeRef` / `CodeSystemRef` node. That alone guarantees a mismatch in essentially every HEDIS library.

**Fixed.** `translate` now searches the input file's own directory (matching the CQF CLI) plus any `--lib-dir` directories, and the qualified-reference paths resolve terminology, code and expression references through the include.

### Part A was measured against the wrong serializer

The bundle ELM used as reference is serialized by the JAXB/MOXy `elm-json` writer used for FHIR `Library` resources, which stamps a `"type"` discriminator on **every** node. The `cql-to-elm` **CLI** does not: its JSON writer omits `type` on `VersionedIdentifier`, `UsingDef`, `IncludeDef`, `ContextDef`, `ParameterDef`, `ValueSetDef`, `CodeSystemDef`, `CodeDef`, `ExpressionDef`, `OperandDef`, `CaseItem`, `AliasedQuerySource`, `LetClause`, `ReturnClause`, `SortClause`, `AggregateClause`, `TupleElementDefinition` and the `Library$*` containers — exactly the list reported as missing.

Verified with CQF 4.8.0 CLI output for a `case` expression:

```json
"caseItem": [ { "when": { "type": "Less", ... }, "then": { "type": "Literal", ... } } ]
```

No `"type": "CaseItem"`. Echo-elm matches the CLI here.

**Consequence:** adding `type` everywhere would *break* CQF CLI parity, which is echo-elm's stated compatibility target. The fully-discriminated form belongs in an opt-in output mode, not in default output.

**Implemented** as `--target bundle` (default `cqf`), and `TranslateResult.BundleJSON()` on the Go API. It is a post-serialization pass over the ELM JSON — the same positional knowledge the CLI serializer relies on, spelled back out — so it needs nothing from the translator and cannot perturb default output. Nodes that already carry a `type` keep it, so `FunctionDef`, the `With`/`Without` relationship clauses and the sort-by variants are untouched.

**Correction (issues/04):** this was first shipped as `--target firely`, described as the shape the Firely CQL SDK deserializes. That was wrong, and the name has been retired. Firely *writes* the lean shape and discards these discriminators on read — its `CorrectLegacyConstructs` pass treats the fat shape as legacy input, the synthetic `type` property is validated and thrown away, and Firely's own round-trip fixture carries zero implied discriminators. The shape belongs to CQF's JAXB/MOXy `elm-json` writer, so the target is named for where it is observed: inside FHIR `Library` resources. Firely consumption is precisely the case that does *not* want it.

The genuine motivation is parity, not compatibility: `parity --ref-dir` and `parity --bundle` cannot produce a meaningful diff while the two sides are in different serializer shapes. On that path the harness now uses the **inverse**, `elm.StripImpliedTypes`, reducing the reference down rather than inflating echo-elm's output up — lossless, and it keeps the comparison from depending on a table that tracks a Java class hierarchy. `--target bundle` remains for writing ELM back into a bundle.

The property that matters to a deserializer dispatching on `type` is that no node with structure is left bare, and one fixture cannot show that holds for every construct: `TestBundleTargetTypesEveryNode` asserts it across the whole corpus. It immediately caught a rule a hand-written sample had missed (`Ratio.numerator` / `.denominator`). `TestImpliedTypeRoundTrip` additionally proves the two directions are exact inverses corpus-wide.

`--target bundle` is rejected with `--format XML`, where every node already carries `xsi:type`.

### Part B claims that were artifacts of the above

`AliasedQuerySource` (B1), `ValueSetDef`/`ValueSetRef` (B3), `CodeRef`/`CodeDef`/`CodeSystemDef`/`CodeSystemRef` (B4), `ListTypeSpecifier`/`IntervalTypeSpecifier` (B5), `SortClause` (B7), `CaseItem` (B9), `AggregateClause` (B10), `TupleElementDefinition`/`TupleTypeSpecifier` (B12), and `LetClause`/`QueryLetRef`/`InValueSet` (B13) are all **emitted correctly** and always were. They were counted as missing because the analysis counted `"type": "X"` occurrences, and these nodes carry no `type` in CLI output.

`ToDecimal` (B6) was also already correct: `5 + 1.0` emits `Add(ToDecimal(5), 1.0)`, matching CQF exactly.

`As` (B2) was already correct: every form in `TestCasts` (`as`, `cast as`, `as Choice<…>`, `as List<…>`, `as Interval<…>`, `convert to`) matches CQF.

`Or`/`And` expansion (C1) matches CQF; no over-expansion was found.

### Confirmed and fixed

| Ref | Gap | Fix |
|-----|-----|-----|
| B8 / C4 | `date from` / `time from` / `timezoneoffset from` emitted `DateTimeComponentFrom` with a `Date`/`Time`/`TimezoneOffset` precision | Emit `DateFrom` / `TimeFrom` / `TimezoneOffsetFrom`; only Year–Millisecond use `DateTimeComponentFrom` |
| — | `DateTimeComponentFrom.operand` was an array | It is a UnaryExpression; operand is a single object |
| — | `date from <Date>` did not convert its operand | The three dedicated operators are DateTime-only, so a Date operand is wrapped in `ToDateTime` |
| B11 / C3 | Identifiers in a sort-by expression emitted `ExpressionRef` / `QueryThisRef` | They are scoped to the query result element, which the engine resolves; emit `IdentifierRef` (`$this` included) |
| C3 | Aggregate accumulator emitted `ExpressionRef` | The accumulator is in scope in the body; emit `QueryLetRef` |
| — | `aggregate … starting 0` dropped the starting value, emitting `Null` | The grammar allows `simpleLiteral` and `quantity`, not just a parenthesised expression; all three are now built |
| — | `aggregate all` did not emit `distinct: false` | `Distinct` is now a `*bool`, so plain/all/distinct emit nothing/false/true |
| — | `return distinct` did not emit `distinct: true` | Same: the explicit flag is preserved in both directions |
| C6 | `!=` / `!~` emitted `NotEqual` / `NotEquivalent` | No such ELM operator; lower to `Not(Equal(…))` / `Not(Equivalent(…))` |
| C2 | `in` wrapped its right operand in `ToList` whenever that operand was not *syntactically* a list or interval literal, so `2 in Numbers` was wrongly promoted | Promotion is now decided from the inferred operand types: promote only when the right operand cannot already satisfy an In overload for the left operand's type |
| C5 | `during` / `included in` always emitted `IncludedIn` | Select `In` / `ProperIn` when the left operand is a point, `IncludedIn` / `ProperIncludedIn` when it is an interval |
| — | A quoted identifier used as a query source kept its quotes, producing `ExpressionRef` named `"\"My Items\""` | Query sources route through the same unquoting/qualifier path as every other reference |
| B3 | Library-qualified value set references were not recognised as `InValueSet`, and bare `ValueSetRef` expressions lacked `preserve` | Value-set resolution accepts qualified forms; every `ValueSetRef` is stamped `preserve=true` under CQL 1.5 (suppressed at 1.4) |
| B4 | `concept` declarations dropped the display clause, kept quotes on code names, and stamped a spurious `"type": "CodeRef"` / `xsi:type` | Codes are unquoted (with optional library qualifier), the display clause is captured, and the discriminator is dropped to match CQF |

Also fixed while verifying, though not reported in the original analysis:

- **Implicit FHIRHelpers conversion was applied at every property access.** CQF inserts it only where a FHIR value is *consumed* as an operand of a typed operator. A property that is a define body, a query return expression, a list element, or the source of a further navigation keeps its raw FHIR value — so `define "SDE Sex": Patient.gender` is a bare `Property`, while `Patient.gender = 'female'` converts with `ToString`, and `E.period.start` converts only the resulting `dateTime`, never the intermediate `period`.
- **Block-comment `@tag` annotations were filtered through an allowlist** of spec-defined names. CQF has no allowlist: it emits any tag whose name is a valid CQL identifier and drops the rest, so `@author` and `@customThing` are emitted while `@measure-component` and `@dotted.name` are not.
- `Patient.birthDate` was missing from the FHIR property-type table, so it never received its `ToDate` conversion. The table has since been replaced wholesale — see below.

### FHIR ModelInfo tables are now generated

`internal/typesystem/fhir_modelinfo.go` carried a hand-maintained subset of the FHIR model: 275 property types and 99 primary code paths. Anything absent silently skipped its implicit conversion, and several entries were simply wrong.

`internal/tools/modelinfogen` now generates `fhir_modelinfo_gen.go` from `org/hl7/fhir/fhir-modelinfo-4.0.1.xml` — the same ModelInfo the CQF toolchain compiles against, read out of the `quick` JAR that `task parity:jar` installs. Regenerate with `task generate:modelinfo`; the output is committed so building echo-elm never needs the JARs.

| Table | Before | After |
|-------|--------|-------|
| `FHIRPropertyType` | 275 | 4500 |
| `FHIRPrimaryCodePath` | 99 | 63 |
| `FHIRListProperty` | — | 1464 |
| `FHIRPrimitiveValueType` | — | 267 |

What that corrected, each verified against the CQF CLI:

- **8 wrong primary code paths.** `Communication` was `category` (actually `reasonCode`), `Library` was `type` (actually `topic`), `ClinicalImpression` was `statusReason` (actually `code`), and five more.
- **43 invented primary code paths.** `AuditEvent`, `DocumentReference`, `Contract` and others have no `primaryCodePath` in FHIR 4.0.1 at all — CQF warns "the type of the retrieve does not have a primary code path defined" and emits no `codeProperty`. Echo-elm was emitting one.
- **51 bogus expanded choice properties.** `Patient.deceasedBoolean`, `Condition.onsetDateTime` and similar were in the hand-maintained table, but CQF rejects them outright: "Member deceasedBoolean not found for type Patient." The generated table omits choice elements, matching CQF, which leaves `Patient.deceased` uncoerced.
- **Binding types are now recorded as declared** (`Patient.gender` is `AdministrativeGender`, not `code`), which is what CQF puts in the conversion signature. Their conversion is resolved through the System type of the type's `value` element, so primitives and enum bindings are handled by one rule and `FHIRPropertyBinding` — a one-entry hand-maintained map — is gone.
- **List-valued properties are tracked**, so a `List<FHIR.string>` is no longer at risk of being wrapped in a scalar conversion now that the table covers it.
- **Nested backbone elements resolve** (`Patient.contact` is `Patient.Contact`, not an opaque `BackboneElement`), so properties beneath them are typed.

### Result types are now recorded

`--result-types` (and therefore `--debug`) previously had no implementation at all: `EnableResultTypes` was a declared option that nothing read. It now records `resultTypeName` / `resultTypeSpecifier` following CQF's model, which the ELM was reverse-engineered to establish:

- **Only expressions that came from source are stamped.** Synthesized nodes carry no result type — the implicit context accessor (`SingletonFrom(Retrieve)` for `context Patient`), inserted FHIRHelpers conversions, and promotion wrappers. `translateExpr` is the funnel for source-derived expressions, so stamping there reproduces the distinction; the conversion sites opt out explicitly and stamp the node they wrap instead.
- **A FHIR property reports its FHIR type, not the converted one.** `Patient.gender` is `{http://hl7.org/fhir}AdministrativeGender`; the conversion above it is what produces the `String`, and that node is unstamped.
- **Declarations carry fixed system types** — `CodeSystemDef` → `CodeSystem`, `ValueSetDef` → `ValueSet`, `CodeDef` → `Code`, `ConceptDef` → `Concept`, and the `codeSystem` reference inside a `CodeDef` likewise.
- **Type specifiers are stamped only where written in source** — an `as`/`is` target, a parameter or operand type, and their nested element/point types. A specifier synthesized to *carry* a result type is metadata, not a node, and is left alone.
- **Query clauses are resolved while their scopes are live.** A query's alias and let scopes are popped as soon as it returns, so the query, its sources, its lets and its return clause are stamped inside `translateQuery` rather than by the caller.

All 24 `debug` fixtures match CQF exactly.

Inference rules worth knowing, each established from CQF's output rather than assumed:

- An aggregate reduces `List<T>` to `T`, except `Count` (Integer) and the statistical ones (Decimal). `Floor`/`Ceiling`/`Truncate` are Integer-valued, not Decimal.
- `width of Interval<T>` is `T`, not a Quantity.
- A definition declared in a retrieve context is **list-valued when referenced from `Unfiltered`** — it yields one value per context member. `define Patients: Patient` is `List<Boolean>` when `Patient` was defined under `context Patient`.
- `Coalesce` yields the common type of its alternatives, so a single untyped null among them makes the whole expression `Any`.
- An untyped null takes its sibling's type: as an interval bound, as the other operand of a binary operator, and as `collapse`'s omitted `per` argument (a Quantity).
- `Any` is otherwise treated as "not resolved" and left unstamped. Emitting it wherever inference fell short was tried and measured: it does not reduce the diff, and it asserts a wrong type to a consumer rather than staying silent.
- A qualified reference resolves against the included library's definition types, which requires translating that library once. Only result types need this, so it runs only when they are enabled.

Two classes of latent bug surfaced while doing this and are fixed:

- **Several operators had no result-type inference at all** — `Upper`, `Substring`, `IndexOf`, `Combine`, `Split`, `PositionOf`, every aggregate, `Floor`/`Ceiling`, `Is`. These were invisible while result types were normalized away.
- **Five nodes hand-rolled their `MarshalJSON`** and so silently dropped any field added to their struct: `InValueSetNode`, `ListNode` and `NamedOperatorExpressionNode` (which built their JSON field by field), plus `LiteralNode` and `NullNode` (which were missing `resultTypeSpecifier` outright). All now derive their output from the struct via an alias. `TestMarshalPreservesResultType` round-trips a result type through every one of the 48 expression nodes, and `TestAllExpressionTypesCovered` fails if a new node type escapes that list — so the class cannot recur silently.

Two internal conversions were also silently lossy and are fixed: `astTypeSpecToTypeSpec` and `elmTypeSpecToTypeSpec` both dropped `TupleTypeSpecifier`, degrading every tuple type to `Any` on a round-trip.

`--result-types` and `--debug` are now also exposed on the CLI, matching the CQF flags.

Two conversions were fixed alongside, both newly reachable once the property types resolved:

- **Date→DateTime in comparisons.** `Specimen.receivedTime before @2020-01-01` converts the Date literal; two Dates are compared as Dates.
- **Date→DateTime through interval bounds.** `dateTime in Interval[@2020-01-01, @2020-12-31]` converts each bound rather than the interval as a whole.

Regression coverage: `elm-nodes/ModelInfoCoverage.cql` and `elm-nodes/IntervalBoundPromotion.cql`.

### Normalization review

The golden comparison normalizes both sides before diffing. That normalization was itself audited by capturing raw CQF and echo-elm output for all 291 fixture × profile pairs and re-comparing under progressively weaker normalization. Four steps were removing things a consumer of the ELM would care about, and each was hiding a real defect:

| Step | Pairs it hid | Verdict |
|------|--------------|---------|
| `sortDefinitionArrays` | 4 | **Removed.** Was hiding statement emission order. |
| `resultTypeName` / `resultTypeSpecifier` strip | 30 | **Removed.** Result types are the point of `--result-types`. |
| `translatorOptions` / `signatureLevel` / `compatibilityLevel` strip | 482 field instances | **Removed.** Describes how the ELM was produced. |
| `CqlToElmError` drop | 178 annotations | Still dropped, now documented; see Open below. |

What that unmasked, and what was done:

- **Statement emission order.** CQF resolves definitions lazily, so a define referenced by an earlier one is emitted before it; echo-elm emitted source order. `Initial Population` forward-references `Has Two or More Qualifying Encounters`, and the two came out swapped. **Fixed** — statements are now emitted in dependency order with source order as the tiebreaker (`statementEmissionOrder`, backed by a new reflective `ast.Walk`).
- **`translatorOptions` was hardcoded to `""` in CQF mode** and never reported anything. It now emits CQF's exact vocabulary and order (`EnableDateRangeOptimization, EnableAnnotations, EnableLocators, EnableResultTypes, EnableDetailedErrors, DisableListTraversal, DisableListDemotion, DisableListPromotion, EnableIntervalDemotion, EnableIntervalPromotion, DisableMethodInvocation, RequireFromKeyword`). **Fixed.**
- **`compatibilityLevel` was emitted; CQF never emits it.** The level shapes translation but is not described in the ELM. **Fixed.**
- **`--strict` set `RequireFromKeyword`; CQF's does not.** Verified directly: `--strict` reports exactly `DisableListTraversal,DisableListDemotion,DisableListPromotion,DisableMethodInvocation`, while `--require-from-keyword` on its own does report `RequireFromKeyword`. echo-elm was rejecting queries CQF accepts under `--strict`. **Fixed**, in the CLI and in the `strict` corpus profile.
- **`resultTypeSpecifier` was emitted on every `FunctionDef` with a `returns` clause**, regardless of `--result-types`. **Fixed** — it is result-type information and is now gated on the option.

Two reductions remain, both deliberate and now documented in `normalizeJSON`:

- **`localId` and the `r` references into it.** An internal node index: CQF numbers from a pre-order ANTLR rule visit, echo-elm from its own counter. No semantic content.
- **Annotation s-tree shape.** CQF splits a definition's source into per-grammar-rule segments (`["", "define ", "\"Initial Population\"", ":\n  ", "true"]`); echo-elm emits one span with the same concatenation. The *text* is still compared, so wrong annotation content still fails — only the segmentation is unverified. `TestAnnotationTextInvariant` checks that the retained text really is a slice of the `.cql` source, so the reduction is not a blank cheque.

### Open

- **`sig-overloads` signature emission for membership operators** — see below; unchanged.
- **Diagnostic annotations are not emitted.** CQF writes `CqlToElmError` annotations into the ELM (178 across the corpus, all `errorSeverity: warning` — overload-ambiguity notices under `SignatureLevel=None`, "List-valued expression was demoted to a singleton", "Interval-valued expression was demoted to a point"). echo-elm reports diagnostics on stderr but emits none into the ELM, so normalization drops them. A measure author reading the ELM would want these.
- **`sig-overloads` signature emission for membership operators.** CQF emits a signature on some `In` expressions where no coercion occurred — `2 in {1,2,3}` and `"Day" included in "Period"` carry one, while `5 in Interval[1,10]` and `"Day" in "Period"` do not. Literal-vs-reference and rewrite-vs-direct both fit part of the data and neither fits all of it. The emitted ELM matches at every other profile including `sig-all`, so this is signature metadata only. `elm-nodes/MembershipOperators.cql` excludes `sig-overloads` with this note.
- **Conversion at the consumption site.** echo-elm attaches the implicit FHIRHelpers conversion where a property is *translated*, whereas CQF applies it where the value is *consumed*. The direct list case is now handled — `'Main St' in A.line` emits CQF's per-element conversion query (previously it wrapped the list in `ToList`, producing a `List<List<String>>` that no engine could evaluate) — but two shapes remain:
  - A scalar extracted from a FHIR list: `First(A.line) = 'x'` should be `ToString(First(…))`; echo-elm emits `First(…)` with no conversion.
  - Implicit list traversal: `'z' in C.category.text` where `category` is `List<CodeableConcept>` needs a nested `$this` traversal query inside the conversion query.
- **Implicit list traversal** more broadly — navigating a property through a list-valued complex element.
- **CQF version divergence.** CQF 3.29.0 and 4.8.0 lower an `Interval<DateTime>` parameter default differently (3.29.0 emits `Property` accessors over the source interval plus `lowClosedExpression`/`highClosedExpression`; 4.8.0 folds the promoted bounds in directly). Fixtures hitting this are marked `versionDivergent: true` in `corpus.yaml` and take the 4.8.0 form.

---

## Part D — Firely SDK Design Considerations

The following issues in the **Firely CQL SDK** (`FirelyTeam/firely-cql-sdk`) remain relevant if echo-elm adds a Firely-targeted output mode. These are *runtime* deviations in the Firely engine, not translator gaps — echo-elm emits no evaluator, so nothing here is actionable in the translator today beyond matching CQF, which it does.

### D1. `EndsWith` — missing bounds check and empty-suffix handling [HIGH]

**Spec (CQL 1.5.3 §EndsWith):** "If the suffix is the empty string, the result is true."
Firely throws `ArgumentOutOfRangeException` when `suffix.Length > argument.Length`.

```cql
define "EmptySuffix": EndsWith('ABC', '')    // expected: true
define "LongSuffix":  EndsWith('AB', 'ABCD') // expected: false, not exception
```

### D2. `Matches` — partial matching semantics [MEDIUM]

**Spec:** Matches uses partial matching (`\\d,\\d\\w+` matches within `'1,2three'`). Firely auto-prepends `^` and appends `$`, forcing full-string matching.

```cql
define "PartialMatch": Matches('http://fhir.org/Library', 'Library') // expected: true
define "FullMatch":    Matches('ABC', '^ABC$')                        // expected: true
```

### D3. `Split` — full-string separator semantics [HIGH]

**Spec:** `Split(str, sep)` splits on the literal separator string, not on each character in it.
Firely uses `separator.ToCharArray()`, so multi-character separators are treated as character sets.

```cql
define "SplitByComma":   Split('a,b,c', ',')    // expected: {'a', 'b', 'c'}
define "SplitByToken":   Split('a::b::c', '::') // expected: {'a', 'b', 'c'}
// Firely behavior for the second: treats ':' as delimiter → {'a', '', 'b', '', 'c'}
```

### D4. `Round` negative midpoint semantics [HIGH]

**Spec:** Round uses "round half away from zero". `Round(-0.5)` → `-1.0`, `Round(-1.5)` → `-2.0`.
Firely's runtime is correct (`MidpointRounding.AwayFromZero`) but its test expectations are wrong (`0.0` and `-1.0`), leaving those tests permanently skipped.

### D5. `Power` — decimal overload non-representable handling [HIGH]

**Spec:** Non-representable Power results return null. Firely has had regressions in the `Power(decimal?, decimal?)` overload.

### D6. `ProperContains` / `ProperIn` — precision mismatch null vs false [MEDIUM]

**Spec:** Unclear whether a precision mismatch should return `null` or `false`. Track the CQF reference implementation.

---

## Relation to Other Issues

- [Issue 01](01-fhir-bundle-cql-elm-support.md): bundle support would let this analysis run directly without manual extraction.
- [Issue 02](02-parity-ref-elm-support.md): the `--ref-dir` flag used to produce the original data. Note that reference ELM taken from a FHIR bundle is serialized differently from CLI output — see Part A — so a bundle-sourced parity run needs `type` discriminators normalized away before comparison, or it will report every library as mismatched.
