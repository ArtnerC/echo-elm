# Agent: mcp-server

> Use when the task touches `internal/mcp/` or `cmd/echo-elm/mcp`.

## Source of truth
- `contracts/mcp-tools.md` — full tool inventory + schemas
- Spec: Model Context Protocol (latest stable)
- SDK: official Model Context Protocol Go SDK
  (`github.com/modelcontextprotocol/go-sdk/mcp`, unless the official module
  path changes before implementation)

## Conventions
- JSON-RPC 2.0 over stdio. One process = one workspace (`--workdir`).
- Use the official Go SDK for protocol types, server loop, and schema
  integration. Do not use community/third-party MCP protocol packages.
- Each tool: name, description, `inputSchema`, `outputSchema`. Outputs
  validate against `outputSchema` in tests.
- Read-only tools work without `--allow-write`. Write tools fail loudly
  with a clear error when the flag is absent.
- All filesystem-touching tools sandbox paths to `--workdir`.

## Definition of done
- Tool entry in `contracts/mcp-tools.md` with both schemas.
- Round-trip integration test under `test/integration/mcp/`.
- Schema-conformance test for output.
- `task test` green; commit with `feat(mcp): ...` + Copilot trailer.
