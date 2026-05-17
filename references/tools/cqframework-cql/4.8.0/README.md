# CQFramework CQL tooling 4.8.0 notes

This is the current stable line of the CQFramework reference CQL→ELM
translator. echo-elm targets full CLI/option compatibility with this
release as well as the older 3.29.0 line under `info.cqframework`.

## Sources

- Repository: <https://github.com/cqframework/clinical_quality_language>
- Tag: <https://github.com/cqframework/clinical_quality_language/tree/v4.8.0>
- Java/Kotlin tools README: `Src/java/README.md` in the repo.
- CLI entry: `Src/java/cql-to-elm-cli/src/main/kotlin/org/cqframework/cql/cql2elm/cli/Main.kt`
- Options class: `Src/java/cql-to-elm/src/commonMain/kotlin/org/cqframework/cql/cql2elm/CqlCompilerOptions.kt`
- Maven metadata: <https://repo1.maven.org/maven2/org/cqframework/cql-to-elm-jvm/maven-metadata.xml>

Latest stable artifacts found:

- `org.cqframework:cql-to-elm-jvm:4.8.0`
- `org.cqframework:quick:4.8.0`
- `org.cqframework:qdm:4.8.0`
- `org.cqframework:cql-to-elm-cli:4.8.0` (CLI distribution)

Important: the 4.x line migrated the JVM source from Java to Kotlin and
moved the Maven group from `info.cqframework` (3.x) to
`org.cqframework` (4.x). The CLI flag surface and the
`CqlCompilerOptions` enum did **not** change. The 4.x line adds a
`reportSelectivity` library knob; everything else carries over from
3.29.0.

## Relevant modules

- `cql` — generates and builds Kotlin lexers/parsers from the ANTLR4
  CQL grammar; generates ELM model-info classes.
- `elm` — generates Kotlin ELM classes from the ELM XSDs.
- `shared` — cross-module helpers.
- `elm-fhir` — data requirements processor and FHIR utilities.
- `engine` — ELM runtime (the "CQL engine").
- `engine-fhir` — FHIR runtime support.
- `qdm` — QDM schema / modelinfo resources.
- `quick` — QUICK, FHIR, QI-Core, US Core resources + FHIRHelpers.
- `cql-to-elm` — CQL → ELM (XML/JSON) translator.
- `cql-to-elm-cli` — command-line translator.
- `ucum` — default UCUM service.
- `tools:cql-formatter`, `tools:cql-parsetree`, `tools:rewrite`,
  `tools:xsd-to-modelinfo` — auxiliary tools.

## CLI flags (verified against `Main.kt` at v4.8.0)

| Flag | Type / arg | Default | Notes |
| --- | --- | --- | --- |
| `--input` | path | — (required) | file or directory; recursive `*.cql`/`*.CQL`. |
| `--output` | path | next to input | file or directory. |
| `--format` | `XML`/`JSON`/`COFFEE` | `XML` | output format. |
| `--model` | path | — | ModelInfo XML file. |
| `--root-dir` | path | — | FHIR IG root; reads `ig.ini`, enables NPM providers. |
| `--disable-default-modelinfo-load` | bool | off | |
| `--verify` | bool | off | parse + analyze, no ELM written. |
| `--date-range-optimization` | bool | off | |
| `--annotations` | bool | off | |
| `--locators` | bool | off | |
| `--result-types` | bool | off | |
| `--detailed-errors` | bool | off | |
| `--error-level` | enum | `Info` | `Trace`/`Info`/`Warning`/`Error`. |
| `--disable-list-traversal` | bool | off | |
| `--disable-list-demotion` | bool | off | |
| `--disable-list-promotion` | bool | off | |
| `--enable-interval-demotion` | bool | off | |
| `--enable-interval-promotion` | bool | off | |
| `--disable-method-invocation` | bool | off | |
| `--require-from-keyword` | bool | off | |
| `--strict` | bool | off | shorthand (see below). |
| `--debug` | bool | off | shorthand (see below). |
| `--validate-units` | bool | off (CLI) | library default `true`. |
| `--signatures` | enum | `None` | `None`/`Differing`/`Overloads`/`All`. Note: library API `defaultOptions()` is `Overloads`. |
| `--compatibility-level` | string | `1.5` | `1.3`, `1.4`, or `1.5`. |

Composite flags:

- `--debug` ≡ `--annotations --locators --result-types`.
- `--strict` ≡ `--disable-list-traversal --disable-list-demotion
  --disable-list-promotion --disable-method-invocation
  --require-from-keyword` (and `--validate-units` in 4.x).

## `CqlCompilerOptions.Options` enum (4.8.0)

```
EnableDateRangeOptimization
EnableAnnotations
EnableLocators
EnableResultTypes
EnableDetailedErrors
DisableListTraversal
DisableListDemotion
DisableListPromotion
EnableIntervalDemotion
EnableIntervalPromotion
DisableMethodInvocation
RequireFromKeyword
DisableDefaultModelInfoLoad
```

Other library-level fields:
`validateUnits`, `verifyOnly`, `enableCqlOnly`, `compatibilityLevel`,
`errorLevel`, `signatureLevel`, `analyzeDataRequirements`,
`collapseDataRequirements`, `reportSelectivity` (4.x only).

`defaultOptions()` returns:

- `EnableAnnotations`
- `EnableLocators`
- `DisableListDemotion`
- `DisableListPromotion`
- `errorLevel = Info`
- `signatureLevel = Overloads` *(library default; CLI default is `None`)*
- `compatibilityLevel = "1.5"`
- `validateUnits = true`

## Recommended local dependency baseline (downstream JVM consumers)

```xml
<dependency>
  <groupId>org.cqframework</groupId>
  <artifactId>cql-to-elm-jvm</artifactId>
  <version>4.8.0</version>
</dependency>
<dependency>
  <groupId>org.cqframework</groupId>
  <artifactId>quick</artifactId>
  <version>4.8.0</version>
</dependency>
<dependency>
  <groupId>org.cqframework</groupId>
  <artifactId>qdm</artifactId>
  <version>4.8.0</version>
</dependency>
```

Add `ucum` for unit validation and `elm-fhir` for data-requirements
processing or FHIR Library rendering. echo-elm itself is Go and does
not depend on these, but downstream consumers commonly do.
