# Frontend Contract — Svelte workbench

> SvelteKit (latest stable) with `@sveltejs/adapter-static`, **lightweight
> and self-contained**: no UI framework, no design system, no client codegen,
> no state-management library. Plain Svelte components + Tailwind + native
> `fetch`. Build output at `web/workbench/build/` is embedded into the Go binary
> via `//go:embed` and served by `echo-elm ui` (see `contracts/local-api.md`).

## v0 scope (locked)

Three views, nothing else:

| Route | Purpose |
|---|---|
| `/`            | **Library browser** — lists CQL files in the workspace; each row shows `name@version`, path, and any `include`d libraries. Click → opens `/translate?path=...` with the content prefilled. |
| `/translate`   | **Interactive translator** — CodeMirror editor (or fallback textarea), options form, run button. Shows diagnostics list + ELM tabs (XML / JSON). |
| `/parity/:id`  | **Parity report viewer** — renders `report.md` and exposes a per-fixture diff drawer. Listed under `/` when reports exist in the workspace. |

Deferred to a later phase: parity-report rich diff viewer, MCP activity tail.

## Diagnostic explorer (cross-cutting)

On `/translate`, diagnostic rows are clickable: clicking one scrolls the
editor to the locator and highlights the range using CodeMirror's
`EditorView.scrollIntoView` + a transient mark decoration.

## Lightweight stack (locked)

| Concern | Choice |
|---|---|
| Framework        | SvelteKit + `adapter-static` |
| Styling          | Tailwind CSS (JIT) |
| Editor           | CodeMirror 6 (`@codemirror/state`, `@codemirror/view`, `@codemirror/language`, `@codemirror/lang-json` for ELM JSON; **no** custom CQL grammar in v0 — plain text mode is fine) |
| API client       | hand-written `lib/api.ts` over native `fetch` |
| State            | Svelte stores + URL state |
| Routing          | SvelteKit file-based |
| Tests            | Vitest + Playwright |

Excluded on purpose: `shadcn-svelte`, `bits-ui`, `melt-ui`, `layerchart`,
`openapi-typescript`, `tanstack/query`, design-system tokens.

## Build & embedding

```
web/workbench/
  package.json
  svelte.config.js     ← adapter-static, fallback=index.html, prerender=true
  tailwind.config.js
  src/
    routes/+layout.svelte
    routes/+page.svelte               ← library browser
    routes/translate/+page.svelte     ← editor
    routes/parity/[id]/+page.svelte   ← report viewer
    lib/api.ts                        ← thin fetch client
    lib/cql-locator.ts                ← CodeMirror locator helpers
  static/
  build/                              ← gitignored, //go:embed target
```

Go side:

```go
//go:embed all:web/workbench/build
var workbenchFS embed.FS
```

Served at `/` by `echo-elm ui`; SPA fallback to `index.html`.

## Dev workflow

```pwsh
task ui:dev    # pnpm -C web/workbench dev (Vite); VITE_PROXY=http://127.0.0.1:8787
task ui:build  # pnpm -C web/workbench build (writes web/workbench/build)
task ui        # ./echo-elm ui --workspace . --open
```

## Tests

- Vitest unit tests for `lib/api.ts` (mocked `fetch`) and `lib/cql-locator.ts`.
- One Playwright smoke against `echo-elm ui --workspace test/fixtures/workspace`:
  - landing → see at least one library row
  - click library → editor route prefilled
  - click "translate" → diagnostics + ELM tabs render
  - open a parity report (fixture) → markdown renders, diff opens

## Bundle budget

Target initial JS bundle ≤ **150 KB** gzipped. CI fails if the budget is
exceeded by more than 20%. Tracked by `task ui:bundle-report`.
