# Agent: CQFramework cql-to-elm CLI compatibility

You are working on the **CQFramework compatibility surface** of
echo-elm. The goal is that, for the same CQL inputs and options, the
echo-elm CLI produces ELM that diffs cleanly against the CQFramework
`cql-to-elm` reference CLI at **5.0.0** (group `org.cqframework`), the
single pinned parity target. Rows below that name 3.29.0 or 4.8.0 describe
those releases historically; they are not install targets.

Read `AGENTS.md` and
`references\implementation\cqframework-compatibility.md` first; this
file inlines the parts an agent needs to make day-to-day decisions.

## Compat profiles

| Profile flag | Maven coords | CLI source language | Notable delta |
| --- | --- | --- | --- |
| `--compat=3.29` | `info.cqframework:cql-to-elm:3.29.0` | Java | No `reportSelectivity` knob. Legacy `translator` ELM header info element. |
| `--compat=4.8` *(default)* | `org.cqframework:cql-to-elm-jvm:4.8.0` | Kotlin | Adds `reportSelectivity` library knob; `--strict` also implies `--validate-units`. |
| `--compat=latest` | tracks current `main` of `cqframework/clinical_quality_language` | Kotlin | Same flag surface; treat new flags as additive. |

The option enum and CLI flag set are otherwise **identical** between
3.29.0 and 4.8.0.

## CLI flag map (verbatim)

Source: `Main.java` (v3.29.0) and `Main.kt` (v4.8.0). All flags exist
in both releases with the same semantics.

| Flag | Type / arg | Default | Maps to |
| --- | --- | --- | --- |
| `--input` | path (file or dir) | — (required) | input selection |
| `--output` | path (file or dir) | next to input / per-dir | output target |
| `--format` | `XML`/`JSON`/`COFFEE` | `XML` | output format |
| `--model` | path to ModelInfo XML | — | `ModelInfoProvider` |
| `--root-dir` | path to IG root | — | NPM IG context |
| `--disable-default-modelinfo-load` | bool | off | `DisableDefaultModelInfoLoad` |
| `--verify` | bool | off | `verifyOnly = true` (no ELM written) |
| `--date-range-optimization` | bool | off | `EnableDateRangeOptimization` |
| `--annotations` | bool | off | `EnableAnnotations` |
| `--locators` | bool | off | `EnableLocators` |
| `--result-types` | bool | off | `EnableResultTypes` |
| `--detailed-errors` | bool | off | `EnableDetailedErrors` |
| `--error-level` | `Trace`/`Info`/`Warning`/`Error` | `Info` | minimum severity reported |
| `--disable-list-traversal` | bool | off | `DisableListTraversal` |
| `--disable-list-demotion` | bool | off | `DisableListDemotion` |
| `--disable-list-promotion` | bool | off | `DisableListPromotion` |
| `--enable-interval-demotion` | bool | off | `EnableIntervalDemotion` |
| `--enable-interval-promotion` | bool | off | `EnableIntervalPromotion` |
| `--disable-method-invocation` | bool | off | `DisableMethodInvocation` |
| `--require-from-keyword` | bool | off | `RequireFromKeyword` |
| `--strict` | bool | off | shorthand (see below) |
| `--debug` | bool | off | shorthand (see below) |
| `--validate-units` | bool | off (CLI) / on (lib default) | `validateUnits` |
| `--signatures` | `None`/`Differing`/`Overloads`/`All` | `None` | `signatureLevel` |
| `--compatibility-level` | `1.3`/`1.4`/`1.5` | `1.5` | `compatibilityLevel` |

Composite flags:

- `--debug` ≡ `--annotations --locators --result-types`.
- `--strict` ≡ `--disable-list-traversal --disable-list-demotion
  --disable-list-promotion --disable-method-invocation
  --require-from-keyword` (4.x also adds `--validate-units`).

Library-only knobs (no CLI flag, but echo-elm should expose):
`analyzeDataRequirements`, `collapseDataRequirements`,
`reportSelectivity` (4.x only), `enableCqlOnly`.

## Stderr message format we must reproduce

```
================================================================================
TRANSLATE <inputPath>
Translation completed successfully.
ELM output written to: <outputPath>

```

On error: `Translation failed due to errors:` then one line per
diagnostic, formatted `<Severity>:[<sl>:<sc>, <el>:<ec>] <message>` —
use `[n/a]` when the locator is absent. End the per-file section with a
blank line. Use LF line endings.

## Output format details

- ELM XML by default. JSON when `--format JSON`. `--format COFFEE`
  wraps the JSON as `module.exports = <json>;`, then a trailing newline.
- Always end the output document with a single trailing newline.
- When `--output` is a directory, the filename is
  `<input-basename>.<xml|json|coffee>`.

## Parity test plan

1. Bundle a small CQL corpus (the CQFramework `Examples/` directory plus
   FHIRHelpers and a representative measure).
2. Run both the CQFramework CLI (4.8.0) and `echo-elm` with identical
   flags.
3. Normalize ELM JSON (drop `translatorVersion`, sort attribute order,
   collapse insignificant whitespace) and diff.
4. Fail CI on any non-cosmetic difference.
5. Repeat against 3.29.0 with `--compat=3.29` to guard the legacy
   surface.

## What to do when behavior differs

- If we disagree with the reference on a point that the CQL spec
  decides, side with the spec and record the divergence in
  `references\implementation\cqframework-compatibility.md` plus a
  release note. Add a regression test capturing the chosen behavior.
- If we disagree on a cosmetic point (attribute order, whitespace),
  fix the normalizer rather than the translator.
