# Parity Harness Contract

> Validates echo-elm against upstream `cql-to-elm` CLI for
> **5.0.0** (`org.cqframework`), the single pinned parity target.
> Driven by `echo-elm parity` and CI.

## Inputs

- **Corpus root**: `test/corpus/cqframework/` (verbatim copy of upstream
  cql-to-elm tests, Apache-2.0; provenance + NOTICE preserved).
- Optional: `test/corpus/cql-tests/` (from `cqframework/cql-tests`, conditional
  on license).
- A `corpus.yaml` per corpus listing fixtures with: `path`, `options`,
  `expectedStatus`, `tags`.

## Toolchain

- JDK: Temurin 17, installed by `task install:jdk` into `tools\jdk\`
  (Windows: `tools\jdk\jdk-17\bin\java.exe`).
- JARs: downloaded by `task install:cqframework` into
  `tools\cqframework\<version>\cql-to-elm-cli.jar`.

## Driver behaviour

For each fixture:

1. Run upstream `java -jar cql-to-elm-cli.jar <flags> --input <fixture>`,
   capture stderr + emitted ELM (XML and JSON).
2. Run `echo-elm translate <same flags> --input <fixture>`, capture stderr
   + emitted ELM.
3. Normalize:
   - Strip absolute paths (replace with `${FIXTURE}` placeholder).
   - Strip timestamps if any.
   - LF-normalize.
4. Diff stderr (banner + diagnostics) with `diff -u`.
5. Diff ELM XML with canonicalized whitespace; tolerated diffs (xsi:type
   ordering) listed in `parity/tolerated.json`.
6. Diff ELM JSON with object-key-sorted normalization; tolerated diffs likewise.

## Output

Per parity run:

- `report.md` — human summary (counts, top failures).
- `report.json` — machine-readable (one entry per fixture).
- `diffs/<fixture>.{stderr,xml,json}.diff` — raw diffs when status≠match.

## Status taxonomy

| Status | Meaning |
|---|---|
| `match`         | Byte-identical (or only tolerated diffs) |
| `differ-stderr` | stderr differs only |
| `differ-elm`    | ELM differs only |
| `differ-both`   | both differ |
| `echo-error`    | echo-elm failed unexpectedly |
| `upstream-error`| upstream JAR failed unexpectedly |

## CI

- PR: run tagged subset (`tags: [smoke]` in `corpus.yaml`) — fail if any
  fixture regresses vs `parity/baseline.json`.
- Nightly: full corpus; updates `parity/baseline.json` on success only after
  manual review.

## License hygiene

- `test/corpus/cqframework/NOTICE` includes upstream copyright + Apache-2.0
  pointer + commit SHA pulled from.
- `test/corpus/cqframework/PROVENANCE.md` records: source repo, commit, date,
  copy command, any modifications (should be none).
- If `cql-tests` license blocks copying, the harness instead clones it at test
  time under `tools\cql-tests\` (gitignored) and references it.
