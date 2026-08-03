# mcp-wizard Design

A toolkit for building, installing, and distributing MCP servers with minimal
boilerplate. The library gives you everything except the business logic:
harness detection, install wizard TUI, output rendering, CLI parsing, self-update,
doctor checks, project scaffolding, and a full TUI app framework for user-facing
applications.

---

## Library Packages

```
library/
  cli/        Subcommand parser with credential flags
  budget/     Real BPE token counting, FitLines, Truncate, Paginate
  render/     YAML frontmatter + Markdown output, error classification
  flow/       Step abstraction + Flow runner for install wizards
  tui/        Reusable TUI components (CheckboxList, RadioList, TextInput, ...)
  harness/    detect-harness wrapper (library-owned types, no leak)
  installer/  HarnessStep, LoginStep (multi-stage), unattended helpers
  secret/     Credential store (FileStore 0600 atomic, EnvStore), Session
  update/     Self-update, semver, SHA256 verification, atomic swap
  doctor/     Health checks (executable, PATH, config, update)

  app/        TUI app framework (see below)
  app/menu/   Dynamic menu component
  app/list/   Scrollable list with pagination
  app/detail/ Scrollable detail/view screen
  app/form/   Multi-field form with validation
  app/search/ Search input + results
  app/table/  Tabular data display
  daemon/     Daemon lifecycle (start, stop, autostart, IPC)
  command/    One-shot CLI command registry

scaffold/     mcp-wizard new project generator with go:embed templates
templates/    Install scripts, CI/CD workflows, project skeleton
```

---

## TUI App Framework (`app/`)

The interactive TUI app (what runs when you type the binary name bare) is
separate from the install wizard. The three reference projects
(interactive-terminal-mcp, sana-mcp, favro-mcp) each have a full-screen
Bubble Tea application for daily use. The `app/` package extracts what they
share.

### App vs Install Wizard

| Dimension | Install Wizard (`flow/`) | App Framework (`app/`) |
|---|---|---|
| Navigation | Linear (Next/Back/Jump) | Stack-based (push/pop screen) |
| Screen model | Handler over shared state | Owns its own model |
| Data loading | One-shot commands | Polling, streaming, async |
| Lifespan | Transient (run and exit) | Persistent (user quits) |
| Rendering | Step.View(state) | Screen.View(width, height) |

### Screen Interface

```go
// Screen is one view in the app. Unlike flow.Step[T], it owns its own
// model and receives the full tea.Msg.
type Screen interface {
    ID() string
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Cmd, error)
    View(width, height int) string
    Focus() tea.Cmd
    Blur() tea.Cmd
}
```

### Navigation

Stack-based: `Push(screen)`, `Pop()`, `Replace(screen)`. Each screen is
independent. Focus/Blur hooks let screens pause polling when hidden.

### Dispatch Pattern

```
bare invocation (no args or unknown subcommand)
  └─ IsInteractive()?
       ├─ Yes → app.Run()       // opens TUI
       └─ No  → MCP server      // defaults to stdio

known subcommand
  ├─ "mcp" / "server" → MCP server
  ├─ "install" / "configure" → install wizard
  ├─ "login" → login flow
  ├─ "doctor" → diagnostics
  ├─ "update" → self-update
  ├─ "daemon" → daemon lifecycle
  └─ registered in command.Registry → one-shot handler
```

### App Components

| Component | Purpose | Source |
|---|---|---|
| `menu.Model` | Dynamic menu that adapts to auth state | All three apps |
| `list.Model` | Arrow-key-navigable list with viewport sync | All three apps |
| `detail.Model` | Scrollable read-only content viewer | All three apps |
| `form.Model` | Multi-field form with tab navigation, validation | favro-mcp login, sana-mcp sign-in |
| `search.Model` | Query input + results list with pagination | sana-mcp search, favro-mcp card lookup |
| `table.Model` | Columnar data with sorting | favro-mcp users/tags |
| `confirm.Model` | y/n prompt with safe default | All three apps |
| `loading.Model` | Spinner with status label | All three apps |

### Theme

The app uses the same palette as the install wizard, extended with app-specific
styles (frame border, badge, selection highlight, running/ended indicators).

---

## Daemon Lifecycle (`daemon/`)

interactive-terminal-mcp and sana-mcp both have background daemons. The
`daemon/` package provides the shared pattern:

```go
type Lifecycle struct {
    Paths      config.Paths
    Executable string
    Version    string
}

func (l *Lifecycle) Open() (*Instance, error)
func EnsureRunning(runtime *Runtime) bool
func StopByPID(path string) error
```

Key differences between the two are handled by the consumer:
- IPC protocol (socket JSON-RPC vs file-lock + SQLite)
- Session management vs sync loop
- Config reload vs no reload

The library provides the lifecycle mechanics; the domain logic stays in the
consumer.

---

## One-Shot Commands (`command/`)

favro-mcp has CLI commands like `list-boards`, `get-card`, `create-card` that
share business logic with the TUI app. The `command/` package provides:

```go
type Registry struct{}
func (r *Registry) Register(h Handler)
func (r *Registry) Dispatch(ctx, name string, args []string) (bool, int)

type Handler struct {
    Name        string
    Description string
    Run         func(ctx context.Context, args []string) int
}
```

Business logic lives in package-level functions that both the TUI screens and
the command handlers call — the canonical pattern from all three reference
projects.

---

## Dependency Graph

```
app/ ──→ tui/ ──→ lipgloss
  │         │
  │         └──→ flow/ ──→ bubbletea
  │
  └──→ daemon/ ──→ flock, bootstrap

command/ ──→ independent, no library deps

installer/ ──→ flow/, tui/, harness/, secret/
  │
  └──→ daemon/  (for EnsureRunning)
```

Key rule: `flow/` does NOT depend on `app/`. The install wizard does NOT
depend on the app framework. Apps adopt the framework incrementally,
screen by screen.
