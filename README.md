# mcp-wizard

Build, install, and distribute MCP servers with zero boilerplate.

Write your domain logic and MCP tools. The library gives you everything else:
harness detection, install wizard TUI, credential UX, output rendering, CLI
parsing, self-update, doctor checks, project scaffolding, and a full TUI app
framework for user-facing applications.

---

## Quick Start

```sh
# Install the scaffold
curl -fsSL https://github.com/sairaph/mcp-wizard/raw/main/install.sh | sh

# Generate a new MCP server project
mcp-wizard new --name my-server --owner myuser --dir ./my-server
cd ./my-server

# Implement your domain logic and MCP tools
# Then build and ship
```

---

## What You Get

The scaffold generates a complete project with:

| What | How |
|---|---|
| Install scripts | `install.sh` + `install.ps1` with SHA256 verification |
| CI/CD | GitHub workflows for test, build, release (6 targets) |
| MCP server skeleton | `internal/mcpserver/` with tool registration |
| TUI install wizard | Harness detection, credential login, settings |
| TUI user app | Full-screen app with menu, lists, forms, search |
| One-shot CLI commands | Standalone commands sharing logic with the app |
| Self-update | `<bin> update` with semver check + atomic swap |
| Doctor | `<bin> doctor` for diagnostics |
| Daemon lifecycle | Lock-based or socket-based background process |
| AGENTS.md + docs | Tool contract and agent guidelines |

---

## How It Works

```
User runs: curl ... | sh
              |
              v
         install.sh
   [1] Detect OS/arch
   [2] Download binary from GitHub release
   [3] Verify SHA256 checksum
   [4] Swap binary atomically
   [5] Launch: <bin> configure
              |
              v
         TUI Install Wizard
   [6] Detect AI clients (Claude, Cursor, VS Code, etc.)
   [7] Let user choose which to register with
   [8] Collect credentials (if needed)
   [9] Write client configs
              |
              v
         Binary dispatch (main.go)
  [10] Bare invocation (TTY) → TUI app
  [11] Bare invocation (pipe) → MCP server
  [12] Subcommand "install" → install wizard
  [13] Subcommand "list-items" → one-shot CLI command
  [14] Subcommand "daemon" → background daemon
```

---

## Library Packages

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
| `app` | TUI app framework (step-based, single AppModel) |
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

---

## Your Code

Your MCP server has four parts:

### 1. Domain logic (`internal/domain/`)

```go
type Client struct {
    BaseURL string
    Token   string
}

func NewClient(token string) *Client { ... }
func (c *Client) GetBoards(ctx context.Context) ([]Board, error) { ... }
```

### 2. MCP tools (`internal/mcpserver/`)

```go
mcp.AddTool(srv, &mcp.Tool{
    Name:        "list_boards",
    Description: "List all boards...",
    InputSchema: ...,
}, s.handleListBoards)

func (s *Server) handleListBoards(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    boards, err := s.client.GetBoards(ctx)
    if err != nil {
        return render.ErrorResult(render.Classify(err, errorTable)), nil
    }
    return render.SuccessResult(front, body), nil
}
```

### 3. App screens (`internal/app/`)

```go
type screen int
const (
    screenMenu  screen = iota
    screenList
)

type State struct {
    app.AppModel
    items []Item
}

func (m *State) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.HandleGlobalKeys(msg) {
        return m, nil
    }
    switch m.Step {
    case screenMenu:
        cmd := m.Menu.Update(msg)
        // handle menu actions via ActionMsg
    }
    return m, nil
}
```

### 4. Main dispatch (`main.go`)

```go
func main() {
    cmd, err := cli.Parse(os.Args[1:])
    switch cmd.Name {
    case "mcp":      runMCPServer(ctx)
    case "install":  runInstall(ctx, cmd)
    case "list-items": oneShotCommands.Dispatch(ctx, cmd.Name, cmd.Args)
    default:
        if tui.IsInteractive() {
            os.Exit(app.Run(ctx, &State{...}, app.Options{Title: "my-server"}))
        }
        runMCPServer(ctx)
    }
}
```

---

## Design

See [DESIGN.md](DESIGN.md) for the full architecture.
