# echo-elm

A fully spec-compliant CQL→ELM translator for Go — usable as a CLI tool or importable Go package.

## Overview

**echo-elm** translates [Clinical Quality Language (CQL) 1.5](https://cql.hl7.org/) to
[Expression Logical Model (ELM)](https://cql.hl7.org/elm.html) in XML or JSON format,
targeting modern CQL engines such as [echo-qm](https://github.com/echo-health/echo-qm)
and the [Firely/Microsoft CQL SDK](https://github.com/FirelyTeam/firely-cql-sdk).

It also provides a compatibility shim (`echo-elm cqf translate`) for drop-in replacement
of the CQFramework `cql-to-elm` CLI (versions 3.29.0 and 4.8.0+).

## Features

- Full CQL 1.5.3 grammar and type-system
- ELM R1 output (XML Schema and JSON)
- Bundled FHIR R4 / FHIRHelpers model info; pluggable QI-Core / US Core / QDM providers
- Modern CLI: `echo-elm translate`
- CQFramework-compatible CLI: `echo-elm cqf translate` (byte-exact parity tests)
- Importable Go package: `github.com/artnerc/echo-elm/pkg/echoelm`
- MCP server: `echo-elm mcp`
- Lightweight SvelteKit workbench: `echo-elm ui` (loopback-only)

## Quick start

```sh
# Translate a CQL file to ELM JSON
echo-elm translate --format json MyLibrary.cql

# CQFramework-compatible invocation
echo-elm cqf translate -f MyLibrary.cql --format JSON
```

## Documentation

- [Specification](specs/echo-elm/spec.md)
- [Architecture](docs/architecture.md)
- [CLI reference](contracts/cli.md)
- [Go API reference](contracts/go-api.md)
- [Testing strategy](docs/testing-strategy.md)

## Development

Requires Go 1.26.3+, [Task](https://taskfile.dev), and optionally Node 22+.

```sh
task build   # compile
task test    # unit tests
task lint    # golangci-lint
task demo:0  # Phase 0 milestone demo
```

See [docs/dev-environment.md](docs/dev-environment.md) for full setup.

## License

Apache 2.0 — see [LICENSE](LICENSE).
