# Issue: ELM Serializer Shapes — Correct the Firely Claim, Fix the Parity Harness, and Re-Baseline the Gap List

**Priority:** High (supersedes Part A of `03-cqf-parity-gaps.md`)
**Labels:** parity, elm, serialization, correction
**Related:** `03-cqf-parity-gaps.md`, branch `feat/firely-type-discriminators`

---

## Summary

Three things, in dependency order:

1. **Part A of issue 03 is wrong.** The claim that the Firely CQL SDK requires `"type"`
   discriminators on every ELM node is incorrect. Firely does not require them, does not
   write them, and actively discards them on read. Issue 03 already caught half of this
   (the JAXB-vs-CLI writer distinction); this corrects the remaining half and removes the
   stated motivation for the `feat/firely-type-discriminators` branch.
2. **The parity harness was comparing against the wrong option profile.** Reference ELM
   embedded in FHIR bundles declares its own translator options; the harness was inventing
   its own. This produced roughly 7,100 phantom differences.
3. **With both corrected, the residual gap list is much smaller and completely different.**
   The re-baselined list is at the end of this document and should replace the tables in
   issue 03.

---

## Part 1 — Three ELM JSON writers, three different shapes

The confusion is that "ELM JSON" is not one format. Three writers emit structurally
different JSON for the same library.

| Writer | Where it is used | `type` on non-polymorphic nodes? |
|---|---|---|
| CQF `cql-to-elm` **CLI** (`-f JSON`) | `tools/cqframework/*/run.sh`, the parity reference | No |
| CQF **JAXB/MOXy `elm-json`** writer | Embeds ELM into FHIR `Library.content[]` | Yes, on nearly everything |
| **Firely** `Library.SerializeToJson()` | .NET round-trip | No |

The JAXB writer is the odd one out, and it is the one whose output ends up in published
measure bundles. It emits JVM binary class names for JAXB-generated nested classes:

```json
{
  "library": {
    "identifier": { "type": "VersionedIdentifier", "id": "ExampleLib", "version": "1.0.0" },
    "usings": {
      "type": "Library$Usings",
      "def": [ { "type": "UsingDef", "localIdentifier": "System", "uri": "urn:hl7-org:elm-types:r1" } ]
    }
  }
}
```

`Library$Usings` is the binary name of the nested class `Usings` inside `Library`. It is a
Java object-model artifact. It is not in the ELM specification, and no other writer
produces it. Firely's model does not even have a corresponding type — it flattens the
containers to plain arrays:

```csharp
// Cql/Elm/Elm.g.cs
public ContextDef[] contexts;
```

---

## Part 2 — What the Firely SDK actually does

Evidence from the vendored SDK source.

### It writes the lean shape and reads either

```csharp
// Cql/Elm/Library.Serialization.cs
private static JsonSerializerOptions _jsonSerializerOptions   = BuildSerializerOptions(allowOldStyleTypeDiscriminators: false); // write
private static JsonSerializerOptions _jsonDeserializerOptions = BuildSerializerOptions(allowOldStyleTypeDiscriminators: true);  // read
```

The read-path preprocessing method is named `CorrectLegacyConstructs`. It reorders `type`
to first position, drops `type` when its value is an object, and strips empty
objects/arrays — i.e. it treats the fat shape as a legacy input to be normalized away.

### The synthetic `type` property is validate-then-discard

```csharp
// AllowOldStyleTypeDiscriminators(JsonTypeInfo ti)
bool isElmNodeOutsideInheritanceStructure =
    ti.PolymorphismOptions is null &&
    ti.Kind == JsonTypeInfoKind.Object &&
    ti.Type.GetCustomAttributes(typeof(XmlTypeAttribute), false).Length != 0;

if (isElmNodeOutsideInheritanceStructure) addTypeProperty(ti, ti.Type.Name);
// addTypeProperty: ShouldSerialize = (_, _) => false;  Set = validate, then throw the value away.
```

Nothing downstream ever reads it.

### The container converter skips it explicitly

```csharp
// Cql/Elm/Serialization/PolymorphicArrayJsonConverter.cs
// Read:  if allowOldStyleDefinitionTypeDiscriminators and the first property is "type",
//        assert the value StartsWith("Library$") and skip it.
// Write: emit { "def": [ ... ] } with no "type".
```

### The resolver documents the modern shape as the expected one

```csharp
// Cql/Elm/Serialization/PolymorphicTypeResolver.cs
// "In newer serializations of ELM, if the declared type of an element is a concrete type,
//  and the runtime type is the same as the declared type, the type discriminator is not emitted."
```

### The decisive test

Firely's own `Elm_Deserialize_FhirHelpers` test deserializes
`CoreTests/Input/ELM/Libs/FHIRHelpers-4.0.1.json`, re-serializes with Firely's writer, and
asserts JSON equality with only two acceptable errors. Discriminator counts in that fixture:

```
VersionedIdentifier 0    Library$Usings 0    UsingDef 0    ContextDef 0
IncludeDef 0             OperandDef 0        ExpressionDef 0             Annotation 0
FunctionDef 265
```

Zero implied discriminators, 265 genuine polymorphic ones. **That is exactly the shape
echo-elm already emits.** A published-bundle copy of the same library has
`VersionedIdentifier 2, Library$Usings 1, UsingDef 2, OperandDef 267, FunctionDef 267` —
the JAXB shape.

### The downstream .NET consumer removes them

The CVS `cql-executor` patches Firely's deserializer to strip the synthetic properties
outright:

```csharp
// MeasureCompilationService.PatchElmDeserializerOptions()
//   reflect BuildSerializerOptions(true) via NonPublic | Static
//   newOptions.UnmappedMemberHandling = JsonUnmappedMemberHandling.Skip
//   add modifier RemoveSyntheticTypeProperties:
//       remove the "type" property from every JsonTypeInfo where PolymorphismOptions is null
//   write back to the private static _jsonDeserializerOptions field
```

Its own doc comment gives the reason: System.Text.Json 9 introduced validation that a
derived type may not declare a property named the same as an ancestor's polymorphic
discriminator, which the SDK's legacy shim trips. So the executor is not adding
discriminators — it is deleting them to keep the fat shape loadable at all.

### Where `type` is genuinely required

Only where the declared type has `XmlInclude` subtypes and the runtime type is a subtype:

- every `Expression` subtype
- every `TypeSpecifier` subtype
- `FunctionDef` inside `statements.def[]` — omitting it silently deserializes as a plain
  `ExpressionDef`, which is a real, silent correctness bug
- `With` / `Without` relationship clauses
- `ByDirection` / `ByColumn` / `ByExpression` sort variants

echo-elm already emits all of these.

### Conclusion

There is no Firely-driven requirement. B1, B2, B3, B4, B5, B6, B7, B9, B10, B12, B13 and C1
in issue 03 were measurement artifacts of comparing against the JAXB writer, as issue 03
now notes. The remaining correction is that issue 03's phrasing —

> "Emitting the fully-discriminated form for Firely SDK consumption is a legitimate but
> separate feature"

— should not say *for Firely SDK consumption*. Firely consumption is precisely the case
that does **not** want it.

---

## Part 3 — Feedback on `feat/firely-type-discriminators`

The branch should land, but with a different name and a different rationale.

### 3.1 Rename the target

`--target firely` is misleading; the shape it produces is the one Firely rejects hardest.
The code's own doc comment already says the right word:

```go
// AddTypeDiscriminators rewrites lean ELM JSON into the shape produced by the
// JAXB/MOXy elm-json writer ...
```

Suggested: `--target cqf|bundle` (or `jaxb`). `bundle` reads best because that is where the
shape is actually observed.

### 3.2 Fix the doc comment

```go
// before
// ... the shape that appears inside FHIR Library resources and that the Firely CQL SDK deserializes.

// after
// ... the shape that appears inside FHIR Library resources, produced by the CQF
// JAXB/MOXy elm-json writer. It is not required by, and is discarded by, the Firely CQL SDK.
```

### 3.3 The real justification is parity, not compatibility

The genuine value of `AddTypeDiscriminators` is that `parity --bundle` and `parity --ref-dir`
cannot produce a meaningful diff when the two sides are in different serializer shapes.
Without shape alignment every bundle-sourced fixture drowns in implied-discriminator noise.
That should be the stated motivation in the commit message and in issue 03.

### 3.4 Prefer normalizing the reference down, not echo's output up

Inflating echo's output to match the reference is lossy in intent — it makes the harness
depend on a table (`impliedTypes`) that has to stay in sync with a Java class hierarchy.
The inverse is cleaner and strictly lossless: in `NormalizeForGolden`, strip any `type`
whose value is implied by position.

```go
// Strip discriminators that carry no information: the containing node type
// and field name already determine the child's type.
func stripImpliedTypes(node any, parentType, field string) any
```

Position determines type for every entry in `impliedTypes`, so nothing is lost. Once that
exists, `--target bundle` is only needed by `bundle write`, and the parity path has no
dependency on the mapping table at all.

### 3.5 Keep the exhaustiveness test

`TestFirelyTargetTypesEveryNode` / `untypedNodes` is the right guard for `bundle write`
output. Rename with the target.

---

## Part 4 — Harness bug: reference option profiles were invented, not derived

### The bug

ELM embedded in a FHIR bundle records the exact options it was translated with:

```json
{
  "library": {
    "annotation": [
      {
        "translatorOptions": "EnableLocators,EnableResultTypes,DisableListTraversal,DisableListDemotion,DisableListPromotion",
        "type": "CqlToElmInfo"
      }
    ]
  }
}
```

The extraction script was generating its own `default` (no locators, no result types) and
`locators` profiles and comparing those against that reference. Every `locator`,
`resultTypeName` and `resultTypeSpecifier` node therefore mismatched by construction.

### The fix

Parse `translatorOptions` from the reference, require it to be identical across all
libraries in the corpus, and emit exactly one matching profile:

```python
flag_for = {
    "EnableAnnotations":    ("--annotations",            "enableAnnotations"),
    "EnableLocators":       ("--locators",               "enableLocators"),
    "EnableResultTypes":    ("--result-types",           "enableResultTypes"),
    "DisableListDemotion":  ("--disable-demotion",       "disableListDemotion"),
    "DisableListPromotion": ("--disable-promotion",      "disableListPromotion"),
    "DisableListTraversal": ("--disable-list-traversal", "disableListTraversal"),
}
```

Generated corpus profile:

```yaml
option_profiles:
  bundle:
    description: "Matches the reference bundle compile options"
    cliFlags: [--locators, --result-types, --disable-list-traversal, --disable-demotion, --disable-promotion]
    translatorOptions:
      enableLocators: true
      enableResultTypes: true
      disableListTraversal: true
      disableListDemotion: true
      disableListPromotion: true
      signatureLevel: None
```

### Suggested upstream change

`parity --bundle` should do this derivation itself rather than trusting `corpus.yaml`. When
the reference comes from a bundle, its declared `translatorOptions` are authoritative and
any mismatch with the active profile should be a hard error, not a silent diff:

```
parity: bundle declares EnableLocators,EnableResultTypes,... but profile "default"
        translates with enableLocators=false. Refusing to compare mismatched profiles.
        Use --profile bundle or omit --profile to derive it.
```

### Impact

Roughly 7,100 phantom differences removed: `resultTypeName` ref-only went from 4,790 to
200, all `locator` rows disappeared entirely, and the `translatorOptions` row disappeared.

---

## Part 5 — Re-baselined gap list

Method: current `main` (`56498af`), 382-library corpus, `bundle` profile, both sides
compared with **all** `type` discriminators stripped, so what remains is structural only.
Result: `match=0 differ=382 error=0`. Type-blind match is also 0, confirming the remaining
gaps are real and not serializer shape.

Ordered by expected effort-to-value.

### G1 — Empty collections ~~are emitted instead of omitted~~ — WITHDRAWN (~3,300 diffs)

**This is Part A's mistake, one level down.** G1 said "CQF omits the key" — but the
CQF *CLI* emits it. Only the JAXB writer omits it, and the JAXB writer is the
reference these counts came from. Measured directly:

```
version      signature  include  codeFilter  dateFilter  otherFilter  codeSystem   sigLevel
3.29.0               2        2           2           2            2           1     None
4.8.0                2        2           2           2            2           1     None
5.0.0                2        2           2           2            2           1     None
```

All three CLI releases emit the empty arrays *and* `signatureLevel: "None"`. JAXB/MOXy
omits them for the same reason it invented `Library$Usings`: an empty collection maps to
"no XML elements", so it round-trips out of the object model as an absent key.

echo-elm already has an explicit mechanism for this — `Translator.cqfEmptyArrayField()`,
gated on `CQFMode` — and emits exactly what the CLI emits. **Acting on G1 would have
broken CLI parity**, which is precisely what Part A established about adding `type`
everywhere. The correct fix is on the comparison path, not the emission path: reduce the
bundle shape to the lean CLI shape, the same inversion §3.4 prescribes for discriminators.

### G2 — `signatureLevel: "None"` is emitted — WITHDRAWN (382 diffs, one per library)

Same finding: the CQF CLI emits `"signatureLevel":"None"`. See the table above. Withdrawn
for the same reason as G1.

### Generalized lesson

Both G1/G2 and the original Part A were produced by diffing echo-elm's CLI-shaped output
against JAXB-shaped reference ELM and attributing every difference to the translator.
Before any remaining G-item is acted on, it must be re-measured with **both sides in the
same serializer shape**. `parity --bundle` now does this automatically.

### G3 — Result-type inference divergence (~1,700 diffs)

The real translator work in this list.

| Shape | Count |
|---|---|
| `resultTypeSpecifier` only in ref | 577 |
| `NamedTypeSpecifier.name` Boolean vs. other | 321 |
| `resultTypeName` value mismatch (Boolean) | 311 |
| `resultTypeName` only in ref | 200 |
| `resultTypeName` only in echo | 142 |
| `NamedTypeSpecifier.name` Integer vs. other | 117 |
| `resultTypeSpecifier` only in ref (tuple) | 101 |
| `resultTypeSpecifier` only in echo | 93 |
| `resultTypeName` value mismatch (Integer) | 85 |

Two distinct sub-problems: (a) echo attaches a result type where CQF attaches none and
vice versa — an emission-policy difference; (b) echo infers a *different* type — a genuine
inference bug. Suggest splitting these into separate issues; (b) is the one that affects
correctness.

### G4 — Annotation tag blocks not emitted (~365 diffs)

Reference carries structured tag annotations that echo omits (271 ref-only) or emits in a
different shape (48 ref-only / 46 echo-only). Shape:

```json
{
  "annotation": [
    { "t": [ { "name": "uri",    "value": "http://example.org/Measure/Example" } ] },
    { "t": [ { "name": "domain", "value": "Example" } ] }
  ]
}
```

These come from CQL `// @uri:` / `// @domain:` tag comments. The near-equal ref-only /
echo-only pair suggests ordering or grouping differs where echo does emit them.

### G5 — Context resolution — DOES NOT REPRODUCE (205 diffs)

Re-measured against the CQF 5.0.0 CLI. Both halves of the claim were checked directly:

```
context Unfiltered      CQF contexts=[Unfiltered]    echo contexts=[Unfiltered]     stmt ctx identical
context Practitioner    CQF contexts=[Practitioner]  echo contexts=[Practitioner]   stmt ctx identical
```

A non-`Patient` context declaration is honoured, retrieves are scoped to it, and the
implicit context-accessor statement is generated with the same name, context and
`SingletonFrom` shape. Nothing here defaults to `Unfiltered`.

This is **not** closed — it is unreproduced. The claim came from the same
mismatched-profile run as G1/G2, so the likeliest explanation is that the `Measure`
context came from a construct not exercised here (a measure-specific context defined by an
included library, most plausibly). Anyone picking this up should produce a failing
reproducer against the CLI **first**; without one there is nothing to fix.

### G6 — `libraryName` missing on qualified cross-library references — CONFIRMED and FIXED

Re-measured against the CQF 5.0.0 CLI, so both sides are in the same serializer shape.
The gap is real but narrower than described: `ExpressionRef` already carried `libraryName`;
`FunctionRef` did not.

```
CQF   UsesShared  ExpressionRef  name='SharedValue'  libraryName='H'
CQF   UsesFunc    FunctionRef    name='Doubled'      libraryName='H'
echo  UsesShared  ExpressionRef  name='SharedValue'  libraryName='H'
echo  UsesFunc    FunctionRef    name='Doubled'      libraryName=None   ← gap
```

**Root cause.** CQL spells a library-qualified call and a method invocation identically —
`<term>.<name>(...)` — so the parser cannot distinguish them and builds both as
`FunctionRef{Name: "Doubled", Operands: [H, 21]}`. Only the include aliases separate the
two cases, and those are not known at parse time. It is a correctness bug: the call emits
a reference to a function *in the current library*, which may not exist, or may exist and
be a different function.

**Fixed** by recovering the qualifier from the include aliases in a pre-pass. Regression
fixture: `elm-nodes/CrossLibraryFunction.cql`, where a local `"Doubled"` deliberately
shadows the included one so the unqualified form resolves to the wrong function rather
than failing outright.

### G7 — Ordering, not omission — CONFIRMED, and it is a consequence of G6

The re-measurement reproduced this and explains the equal-and-opposite counts.

The fix for G6 has to happen **before** statement ordering, not at emission. Two consumers
must agree about what `H."Doubled"(21)` refers to: emission needs `libraryName` on the
node, and `statementEmissionOrder` needs to *not* see a dependency on a local definition of
the same name. Recovering the qualifier at emission time alone leaves the ordering pass
believing the call depends on the local `"Doubled"`, which hoists that definition ahead of
its referent and shifts every subsequent entry in `statements.def[]` — producing exactly
the "equal ref-only and echo-only counts across unrelated fields" signature G7 describes.

This was caught only because the fix was verified end-to-end against the CLI: the emitted
nodes were already correct while the document order was wrong.

### G-adjacent — `"operand": []` on zero-argument functions — FIXED

Found while building the G6 fixture. CQF emits `"operand": []` on a `FunctionDef` that
takes no parameters; echo-elm omitted the key. The field is what distinguishes a
`FunctionDef` from an `ExpressionDef`, which never carries one.

Note the direction: this is the mirror image of the withdrawn G1. There, echo-elm emitted
empty collections the JAXB reference omitted. Here, echo-elm omits an empty collection the
CLI emits. Which direction is a bug depends entirely on which serializer is the target,
which is the whole point of this document.

### G8 — Aggregate / operand shape divergence

Sampled detail from the first differing fixture:

```
...where.operand[0].name    : only in echo ("Count")
...where.operand[0].operand : kind dict (ref) vs list (echo)
```

echo is emitting an aggregate operator's operand as a list where CQF emits a single object.
Low count but worth a targeted test.

### G9 — Over-expansion of `Or` / `And` and spurious implicit conversions

Carried over from the previous run and **not yet re-measured under the corrected profile**.
Previously: `Or` 534 and `And` 496 echo-only, plus echo-only `ToDateTime`, `IncludedIn`,
`ToList`, `Before`. Re-measure before opening an issue — a share of these were profile
artifacts.

---

## Suggested issue split

| Issue | Scope | Size |
|---|---|---|
| ~~A~~ | ~~G1 + G2~~ — **withdrawn**, the CQF CLI emits both; echo-elm already matches | none |
| ~~B~~ | ~~G7~~ — **fixed**, it was G6's ordering consequence | done |
| ~~C~~ | ~~G6~~ — **fixed**, `FunctionRef` was the gap | done |
| ~~D~~ | ~~G5~~ — **does not reproduce** against the CLI; needs a reproducer before any work | blocked |
| E | G4 — tag annotations | medium |
| F | G3 — result-type inference (split emission-policy vs. wrong-type) | large |
| G | G8 + G9 — aggregate operand shape, re-measure boolean expansion | medium |

---

## Part 6 — Single CQF pin (5.0.0)

The parity target is now one release, 5.0.0, rather than 3.29.0 and 4.8.0 together.

The two pins shared one committed golden set, which only worked because the harness
normalized away every field the releases disagreed on. That made those normalizations
load-bearing for the collapse rather than justified on their own terms, and left the
harness unable to distinguish "the two upstreams differ here" from "echo-elm is wrong
here" — the same category error as Part A, in the version dimension.

Measured before changing anything: echo-elm matches CQF 5.0.0 at **291/291 across every
option profile**, and regenerating the goldens from 5.0.0 produces a byte-identical tree.
The ModelInfo tables are unchanged too. So the retarget carried no behavioral fallout.

Removed with the second pin: `CompareVersionGoldens`, the `versionDivergent` corpus field,
and `demo/showdiff`, whose whole purpose was explaining where the two releases disagreed.

One normalization outlived its stated reason and is now documented honestly rather than
removed: stripping empty `"annotation": []` and `"t": []`. It was attributed to
3.29.0-vs-4.8.0 asymmetry, but it is an echo-elm emission gap — CQF emits the empty
container on nearly every node and echo-elm omits it. Disabling the strip drops parity
from 291/291 to **161/294**. An empty container carries no information for an ELM
consumer, so it still satisfies the normalization contract, but note the direction: this
is the mirror image of G1. There, echo-elm emitted empty collections the JAXB reference
omitted; here, echo-elm omits empty collections the CLI emits.

---

## Harness changes made while producing this

- Reference option profile is now derived from the bundle's declared `translatorOptions`
  rather than invented, and a mismatch across the corpus is a hard error.
- A type-blind comparison pass (strip all `type` from both sides before diffing) was added
  so serializer shape can be separated from structural gaps.
- A residual-difference reporter groups diffs by `(field, shape)` and prints the most
  common forms, which is what produced the tables above.

The first of these is the one worth upstreaming into `parity --bundle`; see 4.3.

---

## Status after re-measurement

Everything below was re-run against the **CQF 5.0.0 CLI**, with both sides in the same
serializer shape. Parity across the whole corpus and every option profile is **299/299**.

| Item | Outcome |
|---|---|
| Part 1–2 (Firely claim) | Confirmed and applied; `--target firely` renamed to `--target bundle` |
| 3.1–3.5 | Done; the exhaustiveness test is joined by a round-trip test |
| Part 4 (option profiles) | Done; derived from the bundle, mismatch is a hard error |
| G1, G2 | **Withdrawn** — the CQF CLI emits both; echo-elm already matched |
| G3 | Not re-measured; result types are profile-sensitive, so measure under the derived profile |
| G4 | Not re-measured |
| G5 | **Does not reproduce**; needs a failing reproducer before any work |
| G6, G7 | **Fixed**, with a regression fixture |
| G8, G9 | Not re-measured |

The four unmeasured items are left open deliberately rather than guessed at. Every claim in
the original Part 5 that has now been checked turned out to be either a serializer-shape
artifact (G1, G2), unreproducible (G5), or real but different in shape from the
description (G6, G7) — so the remaining counts should be treated as unverified until they
are re-run the same way.

---

## Actions

- [x] Amend Part A of `03-cqf-parity-gaps.md` to remove the Firely justification entirely
- [x] Rename `--target firely` to `--target bundle` and fix the `AddTypeDiscriminators` doc comment
- [x] Add reference-side implied-discriminator stripping to `NormalizeForGolden` (4.4) — `elm.StripImpliedTypes`, with a corpus-wide round-trip proof
- [x] Derive and validate the option profile in `parity --bundle` (4.3)
- [x] A withdrawn (see G1/G2). B–G stand but need re-measuring in a single serializer shape first
- [ ] Re-measure G3–G9 under the corrected profile *and* a single serializer shape before acting
