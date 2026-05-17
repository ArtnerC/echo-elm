# Agent: frontend-workbench

> Use when the task touches `web/workbench/` or any embedded workbench UI behavior.

## Source of truth
- `contracts/frontend.md` — routes, components, build/embed strategy
- `contracts/local-api.md` — the local HTTP API the workbench talks to

## Conventions
- SvelteKit + `adapter-static`. Output embedded via `//go:embed`.
- Lightweight stack: Tailwind + CodeMirror 6 + native `fetch`. **No** UI
  framework (`shadcn-svelte`, `bits-ui`, etc.), no client codegen, no state
  library.
- Tests: Vitest for `lib/`, one Playwright smoke against
  `echo-elm ui --workspace test/fixtures/workspace`.

## Definition of done
- Route or component documented in `contracts/frontend.md`.
- Vitest unit coverage for new logic in `lib/`.
- Playwright smoke updated if the route is user-visible.
- `task ui:build` succeeds; `task build` embeds without errors.
- `task ui:bundle-report` within budget (<= 150 KB gzipped initial JS).
