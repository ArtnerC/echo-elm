# Testing Strategy

## Layers

| Layer | Tooling | Scope |
|---|---|---|
| Unit              | Go `testing`               | every package under `internal/` and `pkg/echoelm/` |
| Golden            | Go `testing` + `cmp`       | ELM XML/JSON per fixture under `test/golden/` |
| CLI behaviour     | `testscript` (rsc.io/script)| stdin/stdout/stderr/exit-code tests under `test/cli/` |
| Local UI API      | `httptest` + filesystem fixtures | every endpoint under `test/integration/ui/` |
| MCP               | stdio harness              | every MCP tool under `test/integration/mcp/` |
| Frontend unit     | Vitest                     | `web/workbench/src/**/*.test.ts` |
| Frontend e2e      | Playwright                 | `web/workbench/tests/e2e/` against `echo-elm ui --workspace test/fixtures/workspace` |
| Parity            | `internal/parity` driver   | echo-elm vs upstream JARs (tagged in `corpus.yaml`) |

## Public API rule

A symbol is "public" if it's exported from `pkg/echoelm`, served from the
local UI API, or exposed via MCP. Every such symbol/endpoint/tool ships with
at least one test in the same commit. CI fails if a new public surface lacks
a referencing test (enforced by a small `task test:coverage-api` check).

## Golden management

- Goldens live under `test/golden/<corpus>/<fixture>/Library.{xml,json}`.
- Regenerated with `task test:update-goldens` (interactive guard).
- PRs that change goldens require an explicit `goldens-changed: yes` label
  and a justifying note in the commit body.

## Parity baseline

- `parity/baseline.json` records the last-good per-fixture status.
- CI smoke fails if any fixture regresses (status worsens) vs baseline.
- Nightly full sweep proposes a new baseline; reviewed manually.

## Determinism

- Translator tests run with `GOFLAGS=-count=1`. Any non-determinism (e.g.,
  map iteration leaking into output) is treated as a bug.
- ELM serializers sort sibling elements where the spec permits, to keep
  output stable.

## Performance budget (Phase 11)

- Parse + compile + serialize a 1 KLoC FHIR library in **< 250 ms** on a
  modern laptop. Tracked by `task bench` and a CI benchmark trendline.
