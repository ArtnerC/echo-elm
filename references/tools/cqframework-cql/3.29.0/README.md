# CQFramework CQL tooling 3.29.0 notes

This is the last release line under the `info.cqframework` Maven group.
echo-elm must remain CLI-compatible with this version because many
existing FHIR IGs and tooling pipelines are still pinned to it.

## Coordinates

- `info.cqframework:cql-to-elm:3.29.0`
- `info.cqframework:quick:3.29.0`
- `info.cqframework:qdm:3.29.0`
- `info.cqframework:cql-to-elm-cli:3.29.0`

Group moved to `org.cqframework` starting with the 4.x line.

## Source

- Repo tag: <https://github.com/cqframework/clinical_quality_language/tree/v3.29.0>
- CLI entry: `Src/java/cql-to-elm-cli/src/main/java/org/cqframework/cql/cql2elm/cli/Main.java`
- Options: `Src/java/cql-to-elm/src/main/java/org/cqframework/cql/cql2elm/CqlCompilerOptions.java`

## Runtime

- Pure Java; ANTLR4 generated parser/lexer.
- CLI uses `jopt-simple` for argument parsing.
- Same flag surface and defaults as 4.8.0 — see
  `references\implementation\cqframework-compatibility.md` for the
  complete flag map.

## Distinguishing characteristics vs. 4.x

- Maven groupId: `info.cqframework` (4.x is `org.cqframework`).
- CLI implemented in Java (4.x is Kotlin/JVM).
- `CqlCompilerOptions` lacks the `reportSelectivity` field present in
  4.x. Other option fields are identical.

## echo-elm compatibility mode

- `--compat=3.29` selects the 3.29.0 personality.
- Stderr message format, file extensions, and exit semantics are
  identical to 4.8.0. The only deltas worth tracking are:
  - drop the `reportSelectivity` knob,
  - emit the `translator` info element in the ELM header with a value
    string that the legacy ecosystem still parses.
