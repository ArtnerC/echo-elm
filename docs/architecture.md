# Architecture

> Companion to `specs/echo-elm/spec.md` (what) and `plan.md` (when).
> This doc is the **how**: component boundaries and dependency rules.

## Layering

```
cmd/echo-elm/           ← thin: flag parsing, wires subcommand
  ├── translate         ← imports pkg/echoelm only
  ├── cqf               ← imports pkg/echoelm only; CQFramework-compatible CLI facade
  ├── ui                ← imports internal/ui + pkg/echoelm
  ├── mcp               ← imports internal/mcp + pkg/echoelm
  ├── parity            ← imports internal/parity
  └── version

pkg/echoelm/            ← public API facade; imports the internal translator
                          stack but never UI/MCP/parity packages

internal/
  parser/    lexer/  ast/                ← phase 1
  symtab/    types/                      ← phase 2
  elm/       elm/xml   elm/json          ← phase 3
  options/   modelinfo/  libsrc/  ucum/  terminology/  ← phase 4
  ui/                                    ← phase 7 (loopback HTTP + //go:embed SPA)
  mcp/                                   ← phase 8
  parity/                                ← phase 10
```

**Dependency rules** (enforced by `golangci-lint`'s `depguard`):

- `pkg/echoelm` may import the internal translator stack (`parser`, `types`,
  `elm`, `options`, `modelinfo`, `libsrc`, `terminology`, `ucum`) to provide a
  cohesive public facade.
- `pkg/echoelm` **must not** transitively import `internal/ui`,
  `internal/mcp`, `internal/parity`, or release/tooling packages.
- `internal/ui` imports `pkg/echoelm`; never the reverse.
- `cmd/*` never imports another `cmd/*`.

## Dependency policy

echo-elm may use third-party libraries when they add reliability,
compatibility, or avoid fragile reimplementation. Keep dependencies scoped:

- translator core: smallest practical set; avoid broad frameworks;
- CLI/UI/MCP/parity tooling: may use ecosystem packages when justified;
- MCP: use the official Model Context Protocol Go SDK
  (`github.com/modelcontextprotocol/go-sdk/mcp`, unless the official module
  path changes), not community protocol packages;
- before adding a dependency, inspect transitive imports and avoid packages
  that pull large dependency trees for narrow needs.

## Concurrency model

- Translator is **goroutine-safe per call**; the `Translator` struct may be
  shared across goroutines once constructed.
- `echo-elm ui` is a small stateless HTTP server; each request runs a
  synchronous translation (or filesystem walk) and returns. No background
  jobs, no worker pool, no queue.

## Configuration

- CLI flags > env vars > config file (`echo-elm.toml` if present in cwd) >
  defaults.
- Env prefix `ECHO_ELM_*`.

## Logging & observability

- `slog` with JSON handler in `ui`/`mcp`, text handler in `translate`.
- Request ID middleware emits `X-Request-Id` and stamps every log line.

## Errors

- Translator returns `*Result` with `Diagnostics`; only infrastructural
  errors return Go `error`.
- HTTP layer translates Go errors → RFC 7807 problem+json.

## Security

- Loopback-only by default. Non-loopback bind requires `--allow-remote`
  (documented as unsupported).
- No auth, no sessions, no API keys — single-user developer tool.
- CORS off by default.
