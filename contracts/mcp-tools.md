# MCP Tools Contract — `echo-elm mcp`

> JSON-RPC 2.0 over stdio. Implements the Model Context Protocol so AI agents
> can drive the CQL→ELM translator directly: translate inline CQL or files,
> inspect available translator options, validate ELM output, and browse
> workspace libraries.
> **No DB; all state is filesystem-based.**

## Tool inventory

| Tool | R/W | Purpose |
|---|---|---|
| `translate_cql`          | R | Translate inline CQL → ELM; returns diagnostics + ELM XML/JSON |
| `translate_file`         | R | Translate a `.cql` file (workspace-relative or absolute) and return ELM without writing output files |
| `validate_elm`           | R | Structurally validate a serialized ELM document (XML or JSON) |
| `get_translator_options` | R | Return all available translator options with types, defaults, and descriptions |
| `list_libraries`         | R | Walk the workspace and return parsed library headers |
| `read_library`           | R | Read a `.cql` file's content + parsed header |

## `translate_cql` input schema

```jsonc
{
  "content":  "string (required) — CQL source text",
  "format":   "xml|json|both (default: both)",
  "options": {
    "annotations":            "bool",
    "locators":               "bool",
    "signatureLevel":         "None|Differing|Overloads|All",
    "cqfMode":                "bool — cqframework-compatible output",
    "compatibilityLevel":     "1.3|1.4|1.5",
    "validateUnits":          "bool",
    "disableListDemotion":    "bool",
    "disableListPromotion":   "bool",
    "disableListTraversal":   "bool",
    "disableMethodInvocation":"bool",
    "requireFromKeyword":     "bool"
  }
}
```

## `translate_cql` output schema

```jsonc
{
  "diagnostics": [{"severity":"string","message":"string","locator":"string"}],
  "elmXml":      "string|omitted",
  "elmJson":     "string|omitted",
  "parsedName":  "string|omitted",
  "hasErrors":   "bool"
}
```

## `translate_file` input schema

Same as `translate_cql` except `content` is replaced by `path`:
- Workspace-relative path when `--workdir` is set
- Absolute path accepted when no workspace sandbox is configured

## `validate_elm` input/output

```jsonc
// input
{"content":"string (ELM text)", "format":"xml|json"}
// output
{"valid":true|false, "issues":["string …"]}
```

## `get_translator_options` output

```jsonc
{
  "options": [
    {"name":"string","type":"string","default":any,"description":"string"}
  ]
}
```

## Path resolution policy

`translate_file` and `read_library` resolve paths as follows:
1. If path is absolute → used directly (no sandbox)
2. If path is relative → resolved against `--workdir`; escapes blocked

## Safety

- Write-capable tools require `--allow-write` on `echo-elm mcp`.
- No parity/comparison tools in MCP — those belong in the test harness and
  CI pipeline, not in the agent-facing tool surface.
- No database access tools — there is no database.

## Tests

Integration test per tool in `internal/mcpserver/server_test.go`. Each test
uses a real subprocess MCP connection to the built `echo-elm.exe` binary.
