# CLI Contract — `echo-elm`

> Main interface is modern echo-elm: focused on XML/JSON ELM generation for
> current SDKs and downstream CQL/ELM consumers. CQFramework legacy CLI parity is
> isolated under `echo-elm cqf translate` so compatibility quirks do not leak
> into the main interface.

## Subcommands

```
echo-elm [global-flags] <subcommand> [flags] [args]
```

| Subcommand | Purpose |
|---|---|
| `translate` (default) | Modern CQL → ELM XML/JSON for SDKs and downstream consumers |
| `cqf translate`       | CQFramework-compatible `cql-to-elm` surface |
| `ui`                  | Start loopback-only HTTP server + embedded Svelte workbench |
| `mcp`                 | Start MCP server over stdio |
| `parity`              | Run parity harness vs upstream JARs |
| `version`             | Print version + build info |
| `help`                | Per-subcommand help |

## `translate` — modern echo-elm surface

This is the primary interface for Firely-style SDK workflows and downstream
CQL/ELM consumers:
generate ELM XML/JSON with current CQL 1.5 defaults. It intentionally excludes
legacy output formats and old compatibility-level switches that modern ELM consumers
do not consume.

| Long | Short | Type | Default | Notes |
|---|---|---|---|---|
| `--input`               | `-i` | path        | (required) | File or directory |
| `--model`               | `-m` | path        | —          | ModelInfo XML |
| `--output`              | `-o` | path        | input dir  | Output file or dir |
| `--format`              | `-f` | enum        | `xml`      | `xml` / `json` / `both` |
| `--root-dir`            |      | path        | —          | Optional FHIR IG/package root for library/modelinfo resolution |
| `--verify`              |      | bool        | false      | Verify only; no output |
| `--annotations`         | `-a` | bool        | false      | `EnableAnnotations` |
| `--locators`            |      | bool        | false      | `EnableLocators` |
| `--result-types`        |      | bool        | false      | `EnableResultTypes` |
| `--detailed-errors`     |      | bool        | false      | `EnableDetailedErrors` |
| `--disable-list-traversal` |   | bool        | false      | |
| `--disable-list-demotion`  |   | bool        | false      | |
| `--disable-list-promotion` |   | bool        | false      | |
| `--enable-interval-demotion` | | bool        | false      | |
| `--enable-interval-promotion`|| bool        | false      | |
| `--disable-method-invocation`|| bool        | false      | |
| `--require-from-keyword`|     | bool        | false      | |
| `--disable-default-modelinfo-load` || bool  | false      | |
| `--validate-units`      |      | bool        | true       | FHIR/ELM-suitability default |
| `--error-level`         | `-e` | enum        | `Info`     | `Trace`/`Info`/`Warning`/`Error` |
| `--signature-level`     | `-s` | enum        | `Overloads`| `None`/`Differing`/`Overloads`/`All` |
| `--compatibility-level` | `-c` | string      | `1.5`      | `1.5` only in v0 main interface; CQL 2 is future experimental |
| `--report-selectivity`  |      | bool        | false      | **4.x only** |

## `cqf translate` — CQFramework compatibility surface

Use this only for pinned workflows expecting CQFramework `cql-to-elm` behavior.
It accepts the upstream flag names/defaults from 3.29.0 and 4.8.0:

- `--format XML|JSON|COFFEE` where `COFFEE` emits a CommonJS/CoffeeScript-era
  wrapper: `module.exports = <ELM JSON>;`. This is legacy JavaScript tooling
  compatibility, not a Firely SDK or modern ELM-consumer output target.
- `--target-format` as the 4.x alias for `--format`.
- `--signatures None|Differing|Overloads|All` (canonical upstream name);
  `--signature-level` may be accepted as an echo-elm alias.
- `--compatibility-level 1.3|1.4|1.5` for historical CQFramework behavior.
  There is no CQL 1.2 target. The main echo-elm interface targets CQL 1.5.
- `--validate-units` defaults to the upstream CLI default; in 4.x,
  `--strict` also enables it.
- `--debug` ≡ `--annotations --locators --result-types`.
- `--strict` (3.29.0) ≡ `--disable-list-traversal --disable-list-demotion --disable-list-promotion --disable-method-invocation --require-from-keyword`.
- `--strict` (4.8.0+) ≡ above + `--validate-units`.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | All inputs translated successfully |
| `1` | At least one input had diagnostics at or above `--error-level` |
| `2` | Usage error / invalid flag combination |

## CQFramework stderr banner — exact bytes

For `echo-elm cqf translate`, use LF line endings and one banner per input
file to match upstream exactly. The modern `translate` command may use cleaner
human-readable output, but must keep machine-readable results available through
files/stdout.

Success:
```
================================================================================
TRANSLATE <inputPath>
Translation completed successfully.
ELM output written to: <outputPath>
```

Failure:
```
================================================================================
TRANSLATE <inputPath>
Translation failed due to errors:
<Severity>:[<sl>:<sc>, <el>:<ec>] <message>
```

When a diagnostic has no locator: `[n/a]`.

## Stdout

Modern `translate` writes rendered ELM to stdout when no `--output` is supplied
and exactly one input is selected. `cqf translate` follows upstream behavior.

## `ui` flags

| Flag | Default | Notes |
|---|---|---|
| `--addr` | `127.0.0.1:8787` | Loopback listen address |
| `--workspace` | cwd | Root scanned for `.cql` files and `parity/runs/` |
| `--open` | false | Open default browser at listen URL |
| `--allow-remote` | false | Permit non-loopback bind (unsupported/power-user) |

No auth, no database — see `contracts/local-api.md`.

## `mcp` flags

| Flag | Default | Notes |
|---|---|---|
| `--workdir` | cwd | Root for filesystem tools |
| `--allow-write` | false | Permit write-capable tools |
| `--log-file` | stderr | JSONL log destination |

## `parity` flags

See `contracts/parity-harness.md`.

## Tests

- Golden stderr per fixture under `test/golden/cli/`.
- Parity smoke vs upstream JAR over a tagged subset in CI.
- Each alias has a unit test asserting the expanded flag set.
