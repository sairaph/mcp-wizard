# mcp-wizard

Toolkit for building, installing, and distributing MCP servers with minimal
boilerplate. The library gives you everything except the business logic and
content: harness detection, install wizard TUI, credential UX, output
rendering, CLI parsing, self-update, doctor checks, and project scaffolding.

## Repository Structure

```
library/          Go module github.com/sairaph/mcp-wizard
  cli/            Subcommand parser
  budget/         Token-budget-aware pagination (real BPE tokenizer)
  render/         YAML frontmatter + Markdown output, error classification
  flow/           Step abstraction + Flow runner for TUI wizards
  tui/            Bubble Tea components, layout, theme, spinner
  harness/        detect-harness wrapper (library-owned types)
  installer/      HarnessStep, LoginStep, unattended helpers
  secret/         Credential store (FileStore, EnvStore), session
  update/         Self-update, semver, SHA256 verification, atomic swap
  doctor/         Health checks (executable, PATH, config, update)
scaffold/         `mcp-wizard new` project generator
```

## Packages

| Package | Imports | Description |
|---|---|---|
| `cli` | none | Subcommand parser with credential flags |
| `budget` | tiktoken-go | Real BPE token counting, FitLines, Truncate, Paginate |
| `render` | yaml.v3, MCP SDK, budget | Document, error codes, Classify, Fence, ProgressBar |
| `flow` | bubbletea | Step[T] interface, Directive, Flow[T] runner |
| `tui` | flow, bubbletea, lipgloss | Components, layout, theme, spinner, key hints |
| `harness` | detect-harness | Wrapped boundary types, Detector, ResolveExecutable |
| `installer` | harness, tui, flow, secret | HarnessStep, LoginStep, PrintResults |
| `secret` | stdlib | Store interface, FileStore (0600 atomic), EnvStore, Session |
| `update` | stdlib, net/http | SelfUpdate, Check, Version.Parse/Compare, atomic swap |
| `doctor` | update | Check interface, Runner, 4 built-in checks |

## Quick Start

```sh
mcp-wizard new --name my-mcp --owner myuser --dir ./my-mcp
cd ./my-mcp
# Implement your domain logic in internal/domain/
# Define MCP tools in internal/mcpserver/
# Wire steps in main.go
```

## License

MIT
