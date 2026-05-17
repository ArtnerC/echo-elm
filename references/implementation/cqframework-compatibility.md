# CQFramework cql-to-elm CLI compatibility matrix

echo-elm provides a compatibility mode that mirrors the CQFramework
`cql-to-elm` CLI. Both publish lines must be supported because the wider
ecosystem still pins to either.

| Stream | Maven coordinates | Source layout | Notes |
| --- | --- | --- | --- |
| 3.x  | `info.cqframework:cql-to-elm:3.29.0` (last in this group) | `Src/java/cql-to-elm-cli/src/main/java/.../cli/Main.java` | Pure Java CLI using `jopt-simple`. |
| 4.x  | `org.cqframework:cql-to-elm-jvm:4.8.0` and current `main` | `Src/java/cql-to-elm-cli/src/main/kotlin/.../cli/Main.kt` | Kotlin/JVM CLI; same flags, identical behavior; Maven groupId moved to `org.cqframework`. |

Internally both versions use `CqlCompilerOptions` whose option enum and
defaults are identical, so flag mapping is shared.

## Canonical CLI flags

Source: `Main.java` (v3.29.0) and `Main.kt` (v4.8.0). All flags below
exist in both releases with the same semantics; flags marked **(library
only)** are honored by the Java/Kotlin API but not exposed as CLI flags by
the CQFramework reference CLI — echo-elm should still accept the
corresponding library options.

### Required

- `--input <path>` — file or directory. If a directory, recursively
  process every file ending in `.cql` or `.CQL`.

### IO

- `--output <path>` — file or directory. Defaults: if `--input` is a
  directory, write into the same directory; otherwise write next to the
  input. When the output target is a directory, the output filename is
  `<input-basename>.<ext>` where `<ext>` is one of `xml` / `json` /
  `coffee`.
- `--format XML|JSON|COFFEE` — default `XML`. `COFFEE` wraps the JSON in
  `module.exports = ...;`.
- `--model <model-info.xml>` — load a `ModelInfo` XML file as the
  primary ModelInfoProvider.
- `--root-dir <ig-root>` — root of a FHIR IG (`ig.ini` discovery); enables
  NPM package source + model providers.

### Behavior toggles (mapped from `CqlCompilerOptions.Options`)

- `--disable-default-modelinfo-load` → `DisableDefaultModelInfoLoad`
- `--date-range-optimization`      → `EnableDateRangeOptimization`
- `--annotations`                  → `EnableAnnotations`
- `--locators`                     → `EnableLocators`
- `--result-types`                 → `EnableResultTypes`
- `--detailed-errors`              → `EnableDetailedErrors`
- `--disable-list-traversal`       → `DisableListTraversal`
- `--disable-list-demotion`        → `DisableListDemotion`
- `--disable-list-promotion`       → `DisableListPromotion`
- `--enable-interval-demotion`     → `EnableIntervalDemotion`
- `--enable-interval-promotion`    → `EnableIntervalPromotion`
- `--disable-method-invocation`    → `DisableMethodInvocation`
- `--require-from-keyword`         → `RequireFromKeyword`
- `--validate-units`               → `validateUnits = true`
- `--verify`                       → `verifyOnly = true` (no ELM written)
- `--strict`                       → shorthand: enables
  `disable-list-traversal`, `disable-list-demotion`,
  `disable-list-promotion`, `disable-method-invocation`,
  `require-from-keyword` (and, in 4.x, also `validate-units`).
- `--debug`                        → shorthand: enables `annotations`,
  `locators`, `result-types`.

### Diagnostics

- `--error-level Trace|Info|Warning|Error` — default `Info`. Minimum
  severity reported to stderr.
- `--signatures None|Differing|Overloads|All` — default **`None`** at the
  CLI level (note: the library API default `defaultOptions()` is
  `Overloads`). `Differing` includes invocation signatures that differ
  from the declared signature; `Overloads` includes declaration
  signatures when the operator/function has more than one overload with
  the same arity as the invocation.
- `--compatibility-level <1.3|1.4|1.5>` — set the compatibility level
  recorded in the ELM header. Default `1.5`.

### Library-only knobs (no CLI flag, both versions)

- `analyzeDataRequirements` *(bool)*
- `collapseDataRequirements` *(bool)*
- `reportSelectivity` *(bool, 4.x only)*
- `enableCqlOnly` *(bool)* — skip ELM generation entirely, parse only

## Output format details to match

- ELM XML default. JSON when `--format JSON`. `--format COFFEE` writes
  `module.exports = <json>;` followed by a trailing newline.
- Trailing newline after the document, always.
- File extension chosen from format when output is a directory.
- Stderr messages use exactly this pattern (line endings = LF):

```
================================================================================
TRANSLATE <inputPath>
Translation completed successfully.
ELM output written to: <outputPath>

```

  On errors: `Translation failed due to errors:` followed by one line per
  exception: `<Severity>:[<sl>:<sc>, <el>:<ec>] <message>`. If the
  locator is missing, use `[n/a]`.

## echo-elm CLI mapping

The echo-elm CLI exposes the same flag names. Implementation notes:

- Accept both `--flag value` and `--flag=value`.
- For boolean toggles, the *presence* of the flag enables the option (no
  argument); this matches `jopt-simple` behavior.
- When `--strict` and `--debug` are present, expand them to the same set
  of options the CQFramework Main.kt expands them to (see source) before
  constructing `CqlCompilerOptions`.
- The CLI defaults differ from the library API defaults for
  `signatureLevel` (CLI `None`, library `Overloads`); preserve this
  asymmetry in compatibility mode.

## Parity test plan

1. Bundle a small CQL corpus (the CQFramework `Examples/` directory plus
   FHIRHelpers and a measure or two).
2. Run both `cql-to-elm-cli` (4.8.0) and `echo-elm` with identical flags.
3. Normalize ELM JSON (drop `translatorVersion`, sort attribute order,
   collapse insignificant whitespace) and diff.
4. Fail CI on any non-cosmetic difference.
5. Repeat against 3.29.0 with the same corpus to guard the legacy
   compatibility surface.

## Source pointers

- 3.29.0 CLI: `Src/java/cql-to-elm-cli/src/main/java/org/cqframework/cql/cql2elm/cli/Main.java`
- 3.29.0 options: `Src/java/cql-to-elm/src/main/java/org/cqframework/cql/cql2elm/CqlCompilerOptions.java`
- 4.8.0 CLI: `Src/java/cql-to-elm-cli/src/main/kotlin/org/cqframework/cql/cql2elm/cli/Main.kt`
- 4.8.0 options: `Src/java/cql-to-elm/src/commonMain/kotlin/org/cqframework/cql/cql2elm/CqlCompilerOptions.kt`
