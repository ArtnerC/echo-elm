# MCP Tools Contract — `echo-elm mcp`

> JSON-RPC 2.0 over stdio. Implements the Model Context Protocol so AI agents
> can drive translation, validation, parity, and local build/test loops as
> native tools.
> **No DB; all state is filesystem-based.** Tools are scoped to `--workdir`.

## Tool inventory (v0)

| Tool | R/W | Purpose |
|---|---|---|
| `translate_cql`              | R | Translate inline CQL → ELM, return diagnostics + ELM blob |
| `translate_file`             | R | Translate a `.cql` file under `--workdir` and return ELM without writing output files |
| `list_libraries`             | R | Walk the workspace and return parsed library headers |
| `read_library`               | R | Read a `.cql` file's content + parsed header |
| `validate_elm_xml`           | R | Validate an ELM XML blob against bundled XSDs |
| `validate_elm_json`          | R | Validate ELM JSON shape |
| `diff_vs_cqframework`        | R | Run echo-elm + upstream JAR on a snippet, return unified diffs |
| `run_parity_suite`           | W | Trigger parity run (full or tagged subset); writes report under `parity/runs/<id>/` |
| `list_parity_runs`           | R | List `parity/runs/*` newest-first |
| `get_parity_report`          | R | Fetch markdown + summary JSON for a run |
| `import_corpus`              | W | Pull a tagged corpus into `test/corpus/` |
| `run_tests`                  | R | `task test`; stream results |
| `build`                      | R | `task build`; report binary path |
| `lint`                       | R | `task lint`; structured results |
| `ui_start` / `ui_stop`       | W | Start/stop background `echo-elm ui` (loopback-only) |

## Tool schema convention

Implementation uses the official Model Context Protocol Go SDK
(`github.com/modelcontextprotocol/go-sdk/mcp`, unless the official module path
changes before implementation). Do not use community/third-party MCP protocol
packages.

Each tool publishes a JSON Schema for both `inputSchema` and `outputSchema`.
Example for `translate_cql`:

```jsonc
{
  "name": "translate_cql",
  "description": "Translate a CQL snippet to ELM.",
  "inputSchema": {
    "type":"object",
    "required":["content"],
    "properties": {
      "content": {"type":"string"},
      "options": {"$ref":"#/definitions/Options"},
      "format":  {"enum":["xml","json","both"], "default":"xml"}
    }
  },
  "outputSchema": {
    "type":"object",
    "properties": {
      "diagnostics": {"type":"array","items":{"$ref":"#/definitions/Diagnostic"}},
      "elmXml":  {"type":"string","nullable":true},
      "elmJson": {"type":"string","nullable":true},
      "stats":   {"$ref":"#/definitions/Stats"}
    }
  }
}
```

## Safety

- Write-capable tools require `--allow-write` on `echo-elm mcp`.
- All filesystem-touching tools sandbox paths to `--workdir`.
- No database access tools — there is no database.

## Tests

- One JSON-RPC round-trip integration test per tool.
- Schema-conformance test ensuring tool outputs validate against `outputSchema`.
