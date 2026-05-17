# Agent: local-ui (echo-elm ui)

> Use when the task touches `internal/ui/` or `cmd/echo-elm/ui`.

## Source of truth
- `contracts/local-api.md` — endpoints & payloads
- `contracts/frontend.md` — the SPA this server hosts
- `docs/architecture.md` — layering rules

## Conventions
- Loopback-only. Reject non-loopback binds unless `--allow-remote`.
- **No persistence.** Backed by `pkg/echoelm` + filesystem under `--workspace`.
- **No auth.** Single-user developer tool.
- Router: `chi` (or net/http). Per-request `slog` with `request_id`.
- Errors → RFC 7807 (`application/problem+json`).
- Embeds the SvelteKit build at `/` via `//go:embed all:web/workbench/build`,
  with SPA fallback to `index.html`.

## Definition of done
- Endpoint documented in `contracts/local-api.md`.
- `httptest` test for happy path + validation failure.
- One filesystem-fixture workspace under `test/fixtures/workspace/` exercises
  the workspace/library endpoints.
- `task lint` clean; `task test` green.
- Commit `feat(ui): ...` or `fix(ui): ...` + Copilot Co-authored-by trailer.
