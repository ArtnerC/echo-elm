# Dev Environment

> Everything an agent or human needs to build, test, and run echo-elm
> reproducibly on Windows, macOS, and Linux.

## Required toolchains

| Tool | Version | Install |
|---|---|---|
| Go            | 1.26.3            | system (already installed) |
| Node.js + pnpm| LTS + pnpm 9      | `corepack enable && corepack prepare pnpm@latest --activate` |
| Python (uv)   | uv ≥ 0.4          | `winget install astral-sh.uv` / `pipx install uv` |
| Task          | latest            | `go install github.com/go-task/task/v3/cmd/task@latest` |
| ANTLR4        | 4.13.x            | downloaded by `task install:antlr` |
| JDK (parity)  | Temurin 17 LTS    | `task install:jdk` (project-local under `tools\jdk\`) |
| Docker        | optional          | for release container builds |

## One-time bootstrap

```pwsh
task install:deps     # Go modules + pnpm install for web/workbench
task install:antlr    # downloads antlr-4.13.x jar into tools/antlr/
task install:jdk      # Temurin 17 into tools/jdk/ (parity only)
task install:cqframework  # upstream CLI JARs 3.29.0 + 4.8.0 into tools/cqframework/<version>/
```

## Common loops

```pwsh
task build           # go build ./cmd/echo-elm
task test            # go test ./...
task test:integration # ui + mcp integration tests against filesystem fixtures
task lint            # golangci-lint + prettier check
task fmt             # gofmt -s -w + prettier --write
task ui              # ./echo-elm ui --workspace . --open
task ui:dev          # pnpm -C web/workbench dev (proxy to :8787)
task mcp             # ./echo-elm mcp --workdir .
task parity          # run parity harness against tagged subset
```

## Python scripts

- Long-lived: managed in `tools/py/` with `uv` (`pyproject.toml`, `uv sync`).
- One-shot scripts: `uvx <tool>` (no project install).
- No Python at runtime in the Go binary.

## Repository conventions

- Branches: trunk-based on `main`; short-lived feature branches.
- Commits: Conventional Commits; **always** include
  `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`
  when an AI agent made the commit.
- PR template: links spec section, contract section touched, tests added.

## File-system layout (gitignored)

```
tools/jdk/               ← Temurin 17 (per-OS layout)
tools/cqframework/<v>/   ← upstream CLI JARs
tools/antlr/             ← antlr-4.13.x.jar
web/workbench/node_modules/
web/workbench/build/     ← built into binary via go:embed
.bin/                    ← any locally-built helper binaries
```
