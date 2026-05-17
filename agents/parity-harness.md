# Agent: parity-harness

> Use when the task touches `internal/parity/`, `cmd/echo-elm/parity`,
> `test/corpus/`, or `parity/baseline.json`.

## Source of truth
- `contracts/parity-harness.md` — driver behavior, statuses, tolerated diffs
- `agents/cqframework-compatibility.md` — CLI flag tables & banner format
- `references/tools/cqframework-cql/{3.29.0,4.8.0}/README.md`

## Conventions
- Never modify imported corpus content. Copy verbatim, preserve
  `NOTICE` + `PROVENANCE.md`.
- Upstream JARs and JDK are project-local (`tools\cqframework\`, `tools\jdk\`)
  and installed via `task install:cqframework` / `task install:jdk`.
- Normalize stderr (paths, timestamps, line endings) before diffing.
- Tolerated structural diffs live in `parity/tolerated.json` with a written
  justification per entry.
- Regressions block PRs; improvements update `parity/baseline.json` only
  after manual review.

## Definition of done
- Driver behavior unchanged or documented as a contract update.
- Baseline diff is zero-regression or reviewed/approved.
- Report generated under `parity/runs/<timestamp>/`.
- Commit with `test(parity): ...` + Copilot trailer.
