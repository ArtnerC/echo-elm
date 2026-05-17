# Local UI API Contract — `echo-elm ui`

> Loopback-only HTTP server backing the Svelte workbench.
> **No persistence, no auth, no multi-user.** v0 binds to `127.0.0.1`.
> Every endpoint is a thin wrapper over `pkg/echoelm` and the local filesystem.

## Subcommand

```
echo-elm ui [--addr 127.0.0.1:8787] [--workspace .] [--open]
```

- `--addr`: loopback address; refuses non-loopback binds unless
  `--allow-remote` is set (documented as unsupported, for power users).
- `--workspace`: root directory the UI is scoped to (CQL libraries +
  parity reports live under here).
- `--open`: open default browser at the listen URL.

The server serves the embedded SvelteKit static build at `/` and the
endpoints below at `/api/*`.

## Endpoints

### Workspace

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/workspace`             | Returns `{root, cqlFiles[], modelInfos[]}` |
| GET | `/api/libraries`             | Walks the workspace and returns parsed library headers `{name,version,path,includes[]}` |
| GET | `/api/libraries/{path...}`   | Returns `{path, content}` for a single URL-encoded `.cql` relative path |

### Translate

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/translate` | Body: `{content, path?, options}`. Synchronous. Returns `{diagnostics[], elmXml, elmJson, stats}`. |

### Parity reports (read-only viewer)

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/parity/runs`               | Lists `parity/runs/*` directories in the workspace, newest first |
| GET | `/api/parity/runs/{id}`          | Returns `report.json` |
| GET | `/api/parity/runs/{id}/report.md`| Raw markdown |
| GET | `/api/parity/runs/{id}/diffs/{fixture}.{ext}` | Raw diff bytes (`stderr`, `xml`, `json`) |

### Meta

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/healthz` | `{status:"ok", version}` |
| GET | `/api/version` | `{version,gitSha,goVersion,cqlSpec}` |

## Payload shapes

```jsonc
// POST /api/translate request
{
  "content": "library Demo version '1.0.0' ...",
  "path": "Demo/Demo-1.0.0.cql",         // optional, used for diagnostics
  "options": { /* echoelm.Options */ }
}

// POST /api/translate response
{
  "diagnostics": [
    { "severity":"Error", "message":"...", "locator":{"sl":12,"sc":3,"el":12,"ec":17} }
  ],
  "elmXml":  "<?xml version=\"1.0\"...",
  "elmJson": "{\"library\":...}",
  "stats":   { "parseMs":12, "compileMs":34, "serializeMs":5 }
}
```

## Errors

RFC 7807 (`application/problem+json`). No 401/403 in v0 (no auth).

All file paths are workspace-relative, cleaned, and rejected if they escape
`--workspace`.

## Tests

- `httptest` per endpoint.
- One Playwright smoke against `echo-elm ui --workspace test/fixtures/workspace`.

## Out of scope (deliberately)

- Authentication, sessions, API keys
- Multi-tenancy or remote access
- Persistent translation history (the CLI emits ELM files; the UI just runs
  translations on demand)
- Job queue (translations are synchronous and fast)
- Library mutation via API (UI is read-only on the filesystem; create/edit
  CQL with your editor of choice)
