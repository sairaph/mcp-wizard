# mcp-wizard

[![release](https://img.shields.io/github/v/release/sairaph/mcp-wizard?include_prereleases&label=release)](https://github.com/sairaph/mcp-wizard/releases)
[![license](https://img.shields.io/github/license/sairaph/mcp-wizard)](#license)
[![platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue)](#what-it-does)

Build, install, and distribute MCP servers with zero boilerplate.

Write your domain logic and MCP tools. The library gives you everything else:
harness detection, install wizard TUI, credential UX, output rendering, CLI
parsing, self-update, doctor checks, project scaffolding, and a full TUI app
framework for user-facing applications.

macOS / Linux:

```bash
curl -fsSL https://github.com/sairaph/mcp-wizard/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://github.com/sairaph/mcp-wizard/releases/latest/download/install.ps1 | iex
```

The installer downloads the scaffold binary, then run `mcp-wizard new` to
generate a new MCP server project with install scripts, CI/CD, and a complete
TUI application - all wired together and ready to build.

Linux x64/ARM64, macOS x64/Apple Silicon, and Windows x64/ARM64 are published.

## What it does

Building an MCP server means writing the same boilerplate every time: AI client
detection and registration, install wizard, credential storage, output
rendering, CLI dispatch, self-update, diagnostics, and often a TUI app for
users. mcp-wizard provides all of it as a Go library, plus a scaffold that
generates a complete project so you only write your domain logic and MCP tools.

- **detect-harness** powers client detection - finds Claude Desktop, Claude Code,
  Cursor, VS Code, Windsurf, Zed, Cline, Roo Code, and 6 more on your machine.
- **The install wizard** walks users through selecting clients, signing in, and
  configuring settings - in a terminal UI that matches the install script's look.
- **The app framework** gives you reusable components (menu, list, form, search,
  table, confirm, paginator) for building your own user-facing TUI application.
- **Everything is optional** - use just the packages you need.

## Features

- **Project scaffold** - `mcp-wizard new` generates a complete project with
  `install.sh`/`install.ps1`, GitHub CI/CD workflows, and a wired `main.go`.
- **24 library packages** - CLI parsing, token-budget pagination, output
  rendering, install wizard steps, credential management, self-update, health
  checks, TUI components, daemon lifecycle, one-shot command registry, and more.
- **AI client detection** - registers your server with 13+ AI clients via the
  same `detect-harness` library used by all three reference MCP servers.
- **TUI install wizard** - harness selection, credential login (multi-stage),
  settings configuration, unattended mode.
- **TUI app framework** - screen components (menu, list, detail, form, search,
  table, confirm, paginator) with `ActionMsg` dispatch and step-based navigation.
- **Daemon lifecycle** - lock-based (file lock + periodic work) or socket-based
  (Unix socket + JSON-RPC IPC), with autostart and graceful shutdown.
- **One-shot CLI commands** - register standalone commands that share business
  logic with the TUI app.
- **Self-update** - semver version comparison, GitHub release checking, SHA256
  verification, atomic binary swap with cross-device copy fallback.
- **Doctor** - health checks for executable, PATH, config, and update status.
- **Install scripts** - `install.sh` and `install.ps1` with OS/arch detection,
  SHA256 verification, PATH setup, and interactive configure launch.

## Quick Start

```sh
# Install the scaffold
curl -fsSL https://github.com/sairaph/mcp-wizard/releases/latest/download/install.sh | sh

# Generate a new MCP server project
mcp-wizard new --name my-server --owner myuser --dir ./my-server
cd ./my-server

# Implement your domain logic and MCP tools
# Then build and ship
```

## Packages

| Package | Description |
|---|---|
| `cli` | Subcommand parser with credential flags |
| `budget` | Real BPE token counting, FitLines, Truncate, Paginate |
| `render` | YAML frontmatter + Markdown output, error classification |
| `flow` | Step abstraction + Flow runner for install wizards |
| `tui` | Reusable Bubble Tea components (CheckboxList, RadioList, TextInput, ...) |
| `harness` | detect-harness wrapper (library-owned types, no leak) |
| `installer` | HarnessStep, LoginStep (multi-stage), unattended helpers |
| `secret` | Credential store (FileStore 0600 atomic, EnvStore), Session |
| `update` | Self-update, semver, SHA256 verification, atomic swap |
| `doctor` | Health checks (executable, PATH, config, update) |
| `app` | TUI app framework (step-based AppModel, ActionMsg dispatch) |
| `app/menu` | Dynamic menu component |
| `app/list` | Scrollable list with pagination |
| `app/detail` | Scrollable content viewer |
| `app/form` | Multi-field form with validation |
| `app/search` | Search input + results |
| `app/table` | Tabular data display |
| `app/confirm` | Confirmation dialog |
| `app/paginator` | Paginated list with next/prev |
| `command` | One-shot CLI command registry |
| `async` | Generic async loading helpers (Result[T], Load[T]) |
| `daemon/lock` | File-lock-based daemon lifecycle |
| `daemon/socket` | Unix-socket daemon with JSON-RPC IPC |
| `daemon/rpc` | JSON-RPC protocol types |

## Design

See [DESIGN.md](DESIGN.md) for the full architecture.

## License

MIT
