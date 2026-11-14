# CQFramework CQL tooling 5.0.0 notes

The single pinned parity target for echo-elm (`internal/parity.PinnedCQFVersion`).
echo-elm's `cqf translate` surface and every committed golden under `test\goldens\cqf\`
are measured against this release; `task install:cqframework` installs it under
`tools\cqframework\5.0.0\`.

Earlier releases are documented in `..\4.8.0\README.md` and `..\3.29.0\README.md`
for history only — they are no longer installed or measured.

## Sources

- Repository: <https://github.com/cqframework/clinical_quality_language>
- Tag: <https://github.com/cqframework/clinical_quality_language/tree/v5.0.0>
- Maven metadata: <https://repo1.maven.org/maven2/org/cqframework/cql-to-elm-cli/maven-metadata.xml>

Artifacts echo-elm uses:

- `org.cqframework:cql-to-elm-cli:5.0.0` — the CLI the parity harness runs.
- `org.cqframework:quick:5.0.0` — source of the FHIR 4.0.1 ModelInfo that
  `internal\tools\modelinfogen` generates `fhir_modelinfo_gen.go` from, and of
  `FHIRHelpers-4.0.1.cql` (vendored unchanged in `test\corpus\cqframework\layered\`).

## CLI flags

Identical to 4.8.0 — see the table in `..\4.8.0\README.md`. Verified by diffing the
option names compiled into the two releases' `cli/Main.class`; neither release adds
or removes a flag. The CLI has no `--help` option.

## Output differences from 4.8.0 that matter to echo-elm

Measured when the pin moved (echo-elm matched both before and after):

- No committed golden changed: regenerating the whole corpus from 5.0.0 produced
  a byte-identical tree to the 4.8.0-derived set.
- The FHIR ModelInfo tables are unchanged.
- `CqlToElmInfo.translatorVersion` reads `5.0.0`. Reference ELM with no
  `translatorVersion` was not produced by this CLI; `parity --bundle` warns on it.

Behaviors the parity work established against this release, each pinned by a
corpus fixture, are recorded in `issues\03`–`issues\07` and in the comments of
`internal\translator\implicitconv.go`.
