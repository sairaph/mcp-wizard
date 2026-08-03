# mcp-wizard

Build, install, and distribute MCP servers with zero boilerplate.

Write your domain logic and MCP tools. The library gives you everything else:
harness detection, install wizard TUI, credential UX, output rendering, CLI
parsing, self-update, doctor checks, and project scaffolding.

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

## What You Get

The scaffold generates a complete project:

| What | How |
|---|---|
| Install scripts | `install.sh` + `install.ps1` with SHA256 verification |
| CI/CD | GitHub workflows for test, build, release (6 targets) |
| MCP server skeleton | `internal/mcpserver/` with tool registration pattern |
| TUI install wizard | Harness detection, credential login, settings |
| Self-update | `<bin> update` with semver check + atomic swap |
| Doctor | `<bin> doctor` for diagnostics |
| AGENTS.md + docs | Tool contract and agent guidelines |

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
         MCP Server
  [10] Serve tools over stdio
  [11] Handle tool calls with your domain logic
```

## Your Code

Your MCP server has three parts:

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
func registerBoards(srv *mcp.Server, s *Server) {
    addTool(s, srv, TierRead, &mcp.Tool{
        Name:        "list_boards",
        Description: "List all boards...",
        InputSchema: objectSchema(),
    }, s.listBoards)
}

func (s *Server) listBoards(ctx context.Context, req *mcp.CallToolRequest, args listBoardsArgs) (*mcp.CallToolResult, any, error) {
    boards, err := s.client.GetBoards(ctx)
    if err != nil {
        return render.ErrorResult(render.Classify(err, errorTable)), nil, nil
    }
    return render.SuccessResult(front, body), nil, nil
}
```

### 3. Main dispatch and install wizard

```go
func runInstall(ctx context.Context, command cli.Command) int {
    exec, _ := harness.ResolveExecutable()
    detector, _ := harness.New(harness.ServerSpec{Name: "my-server", Command: exec, Args: []string{"mcp"}})
    credStore := secret.FileStore(domain.CredentialPath)

    state := &AppState{}
    steps := []flow.Step[AppState]{
        installer.HarnessStep(ctx, detector, func(s *AppState) *installer.HarnessState { return &s.Harness }, installer.HarnessStepOptions{AllDetected: true}),
        installer.LoginStep(ctx, domain.LoginConfig(credStore), func(s *AppState) *installer.LoginState { return &s.Login }),
    }

    if tui.IsInteractive() {
        return tui.Run(ctx, flow.New(steps, state), tui.Options{Title: "my-server setup"})
    }
    return runUnattended(ctx, detector, credStore, command)
}
```

## Packages

| Package | Description |
|---|---|
| `cli` | Subcommand parser with credential flags |
| `budget` | Real BPE token counting, FitLines, Truncate, Paginate |
| `render` | YAML frontmatter + Markdown output, error classification |
| `flow` | Step abstraction + Flow runner for TUI wizards |
| `tui` | Bubble Tea components, layout, theme, spinner, key hints |
| `harness` | detect-harness wrapper (library-owned types, no leak) |
| `installer` | HarnessStep, LoginStep (multi-stage), unattended helpers |
| `secret` | Credential store (FileStore 0600 atomic, EnvStore), Session |
| `update` | Self-update, semver, SHA256 verification, atomic swap |
| `doctor` | Health checks (executable, PATH, config, update) |

## Design

See [DESIGN.md](DESIGN.md) for the full architecture and [TOOLS.md](TOOLS.md)
for agentic tool design conventions.

## License

MIT
