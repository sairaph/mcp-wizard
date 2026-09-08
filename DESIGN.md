# mcp-wizard Design

A toolkit for building, installing, and distributing MCP servers with minimal
boilerplate. The library gives you everything except the business logic:
harness detection, install wizard TUI, output rendering, CLI parsing, self-update,
doctor checks, project scaffolding, and a full TUI app framework for user-facing
applications.

---

## Library Packages

```
(repository root is the module github.com/sairaph/mcp-wizard)
  cli/        Subcommand parser with credential flags
  budget/     Real BPE token counting, FitLines, Truncate, Paginate
  render/     YAML frontmatter + Markdown output, error classification
  flow/       Step abstraction + Flow runner for install wizards
  tui/        Reusable TUI components (CheckboxList, RadioList, TextInput, ...)
  harness/    detect-harness wrapper (library-owned types, no leak)
  installer/  HarnessStep, LoginStep (multi-stage), TransportStep, ApplyStep, unattended helpers
  secret/     Credential store (FileStore 0600 atomic, EnvStore), Session
  update/     Self-update, semver, SHA256 verification, atomic swap
  doctor/     Health checks (executable, PATH, config, update)

  app/        TUI app framework (step-based AppModel, ActionMsg dispatch)
  app/menu/   Dynamic menu component
  app/list/   Scrollable list with pagination
  app/detail/ Scrollable content viewer
  app/form/   Multi-field form with validation
  app/search/ Search input + results
  app/table/  Tabular data display
  app/confirm/ Confirmation dialog
  app/paginator/ Paginated list with next/prev navigation
  command/    One-shot CLI command registry
  async/      Generic async loading helpers (Result[T], Load[T], Start[T])
  daemon/lock/   File-lock-based daemon lifecycle
  daemon/socket/ Unix-socket daemon with JSON-RPC IPC
  daemon/rpc/    JSON-RPC protocol types (Request, Response, Error)

scaffold/     Nested module building the mcp-wizard binary (project generator)
scaffold/templates/  Install scripts, CI/CD workflows, project skeleton
```

An install wizard is HarnessStep (pick clients) -> LoginStep (credentials) ->
ApplyStep (register, show results, mark the flow Settled). The apply step is
what turns the selection into writes; a wizard without it collects a
selection and discards it.

---

## TUI App Framework (`app/`)

The interactive TUI app (what runs when you type the binary name bare in a
terminal) is separate from the install wizard. The three reference projects
(interactive-terminal-mcp, sana-mcp, favro-mcp) each have a full-screen
Bubble Tea application for daily use. The `app/` package extracts what they
share.

### App vs Install Wizard

| Dimension | Install Wizard (`flow/`) | App Framework (`app/`) |
|---|---|---|
| Navigation | Linear (Next/Back/Skip/Jump) | Step enum + switch in the app's Update |
| Screen model | Handler over shared state | App owns one model with embedded components |
| Data loading | One-shot commands | `async.Load` / `async.Start` messages |
| Lifespan | Transient (run and exit) | Persistent (user quits) |
| Rendering | Step.View(state) wrapped in chrome | Component.View() chosen by step |

### AppModel and steps

There is no screen interface and no navigation stack: the reference apps all
use a step enum with a central switch, and a stack cannot express ephemeral
auto-advancing screens or reloading data on the way back.
An app embeds `app.AppModel`, declares its own steps
from `app.StepCustom` (or its own iota), keeps the components it needs as
fields, and dispatches in a `switch m.Step` in its `Update` and `View`:

```go
type AppModel struct {
    Step    Step
    Width   int
    Height  int
    Status  string // transient status message
    Failure string // fatal error message
    Quit    bool   // set when ctrl+c was handled
}

// Call first in Update. ctrl+c sets Quit and returns tea.Quit; callers must
// return the command.
func (m *AppModel) HandleGlobalKeys(msg tea.Msg) (handled bool, cmd tea.Cmd)

// Keeps a cursor line visible inside a bubbles viewport.
func ScrollIntoView(vp *viewport.Model, line, span int)
```

### Dispatch Pattern

```
bare invocation (no args)
  |- IsInteractive()?
       |- Yes -> app.Run()       // opens TUI
       '- No  -> MCP server      // stdio

known subcommand
  |- "mcp" / "server" / "serve" -> MCP server
  |- "install" / "configure"    -> install wizard (--scope project -> add)
  |- "add"                      -> project-scoped install wizard
  |- "uninstall"                -> remove registration
  |- "login"                    -> login flow
  |- "doctor"                   -> diagnostics
  |- "update"                   -> self-update
  '- registered in command.Registry -> one-shot handler

anything else -> usage, exit 2
```

Note that `cli.Parse` reports no arguments as the `mcp` command, so the
interactive check happens before parsing.

### Components and ActionMsg

Components are plain models with `Init() tea.Cmd`, `Update(tea.Msg) tea.Cmd`
and `View() string`. They talk to the app through a typed message instead
of error strings:

```go
type ActionMsg struct {
    Source string // "menu", "list", "form", "confirm", "detail", ...
    Value  string // "select", "quit", "submitted", "cancelled", "back", ...
    Data   any    // payload, e.g. the selected index or menu action
}
```

| Component | Purpose | Actions |
|---|---|---|
| `menu.Model` | Dynamic menu rebuilt from a function | `select` (Data: action string), `quit` |
| `list.Model` | Cursor list with viewport, optional detail lines | `select` (Data: index), `back` |
| `detail.Model` | Scrollable read-only content | `back` |
| `form.Model` | Multi-field form with validation, secret fields | `submitted`, `cancelled` |
| `search.Model` | Query input + results list | `query` (Data: string), `select`, `cancelled` |
| `table.Model` | Columnar data, `s` cycles sort column and direction | `select`, `back` |
| `confirm.Model` | Two-option prompt | `confirmed`, `cancelled` |
| `paginator.Model` | Page of items with next/prev | `select`, `next`, `prev` |

Async work uses `async.Load(fn)` (delivers `async.Result[T]`) or
`async.Start(fn)` (sends `async.LoadingMsg` first).

### Theme

The app components use the same palette as the install wizard (`tui.DefaultTheme`).

---

## Daemon Lifecycle (`daemon/`)

interactive-terminal-mcp and sana-mcp both have background daemons with
different IPC needs, so there are two concrete packages rather than one
abstraction:

`daemon/lock` (sana-mcp pattern: file lock + periodic work, no IPC):

```go
type Options struct{ LockFile, PIDFile, LogFile string }

func Open(opts Options) (*Instance, error)          // acquire lock, write PID
func (inst *Instance) Close()                       // release lock, remove PID
func IsRunning(lockFile string) bool
func EnsureRunning(executable string, args []string, opts Options) bool
func Stop(pidFile, lockFile string) error           // SIGINT (Unix) / Kill (Windows)
```

`daemon/socket` (interactive-terminal-mcp pattern: Unix socket + JSON-RPC):

```go
func New(socketDir, name string) *Server
func (s *Server) Handle(method string, h Handler)
func (s *Server) Open() error
func (s *Server) Serve(ctx context.Context) error   // returns after handlers finish
func (s *Server) Close()                            // signal only; safe from a handler

func Dial(socketPath string) (*Client, error)
func (c *Client) Call(ctx, method string, params, result any) error
```

`daemon/rpc` holds the wire types (`Request`, `Response`, `Error`, error codes).
Session management, sync loops and config reload stay in the consumer.

---

## One-Shot Commands (`command/`)

favro-mcp has CLI commands like `list-boards`, `get-card`, `create-card` that
share business logic with the TUI app. The `command/` package provides:

```go
type Handler struct {
    Name, Description, Usage string
    Aliases        []string
    RequiresDaemon bool
    Run            func(ctx context.Context, args []string) int
}

func New() *Registry                                  // the zero value also works
func (r *Registry) Register(h Handler)                // panics on duplicates
func (r *Registry) Dispatch(ctx, name string, args []string) (handled bool, exitCode int)
func (r *Registry) PrintUsage(w io.Writer)
```

Business logic lives in package-level functions that both the TUI screens and
the command handlers call - the canonical pattern from all three reference
projects.

---

## Remote Servers and Authentication

Status: designed 2026-09-08 from a survey of 28 hosted MCP providers and every
harness config format. Bridge delivery is implemented (`proxy/`, the
template's `domain.Remote()` and `mcp --remote`). Direct delivery is blocked
on detect-harness gaining a `RemoteServer` (spec in that repo's
`note-from-mcp-wizard.md`).

### What providers actually need

| Auth shape | Share of surveyed providers | What must land in the harness config |
|---|---|---|
| OAuth at first use (DCR / CIMD) | most; six offer nothing else | url + transport tag only; the harness runs the login |
| OAuth with a pre-registered client | GitHub, Slack, Asana, HubSpot | url + client id; the client secret never in the file |
| Static token in `Authorization: Bearer` | second most common; three require it | a header whose value is the secret |
| Custom header or scheme | Sentry, Context7, Exa, Atlassian | same, with a different header name or prefix |
| Key in URL query | Exa, Tavily, Zapier | url only, secret embedded; discouraged |

### The delivery problem

A header value has to reach the harness process. Harnesses disagree on how
a config may reference a secret: `${VAR}` (Claude Code, Gemini, Kiro, Amp,
Copilot CLI), `${env:VAR}` (Cursor, Windsurf, Zoo Code, Amazon Q, VS Code),
`{env:VAR}` (OpenCode), `${{ secrets.VAR }}` from `.env` files (Continue),
an env var name only (Codex `bearer_token_env_var` / `env_http_headers`), a
password prompt (VS Code `inputs`), or nothing at all (Zed, Cline, Junie,
JetBrains AI: literal values only). Claude Desktop cannot hold a remote entry
in its config file.

Env-var references also assume the variable is set where the harness runs.
GUI editors started from a dock or launcher do not inherit shell profile
exports, so "export TOKEN in your .zshrc" is not a reliable instruction.

### Design

`harness.ServerSpec` grows a transport and the remote fields:

```go
type ServerSpec struct {
    Name      string
    Command   string            // stdio
    Args      []string
    Env       map[string]string
    Transport Transport         // TransportStdio (default) | TransportHTTP
    URL       string            // http
    Headers   []Header          // http; values may be secret references
    OAuth     *OAuthClient      // http; nil = harness discovers (DCR/CIMD)
}
```

`Header.Value` is either a literal or `{Prefix, Secret: SecretRef{EnvVar,
Prompt}}`; detect-harness renders the reference in each harness's own syntax
and reports harnesses that cannot express it.

The wizard offers two ways to deliver a static secret, chosen per project in
`domain.AuthConfig` and overridable per install:

1. Direct. The harness config carries the remote entry. The secret is
   referenced through the harness's mechanism where one exists; where none
   exists (Zed, Cline, Junie) the wizard asks before inlining the value, or
   skips the harness. This is the right mode for providers whose users expect
   native harness OAuth, and for OAuth-open servers where no secret exists.
2. Bridge (default for static secrets). The harness is registered with an
   ordinary stdio entry pointing at the generated binary (`<bin> mcp`; the
   endpoint comes from `domain.Remote()`), and the binary forwards stdio to
   the remote over Streamable HTTP, adding the header from its own
   credential store.
   The secret lives once, in `credentials.json` (0600), never in any harness
   file, and every harness including Claude Desktop works identically. Cost:
   the binary must be installed locally, which the one-line installer already
   guarantees.

Implemented (bridge):

- `proxy.Run` connects a local transport (stdio by default) to a remote
  `StreamableClientTransport` and copies JSON-RPC messages both ways without
  interpreting them, injecting configured headers on every request and the
  protocol version negotiated by the initialize handshake. HTTP 401/403 is
  surfaced as `proxy.ErrUnauthorized`. Server-initiated messages outside a
  request (the standalone SSE stream) are not forwarded in this cut.
- `cli`: `mcp --remote <url>`.
- Template: `domain.Remote()` returns a `RemoteConfig{URL, HeaderName,
  HeaderPrefix, CredentialKey, Prompt}` (nil by default); when set, the
  login step collects the token (masked), `mcp` runs the bridge, and doctor
  opens a session against the endpoint with the stored credential. The
  AI-client registration is the ordinary stdio entry, so nothing changes in
  detect-harness.

Remaining (direct):

- `harness/`: the `ServerSpec` fields above; `Harness.Remote` capability
  flags copied from detect-harness so `HarnessStep` can show "bridge" or
  "unsupported" next to each client.
- `installer/`: an `AuthStep` collecting delivery (direct / bridge) when a
  project allows both; `ApplyStep` builds the per-harness remote spec.
  Unattended flags: `--deliver direct|bridge`.
- OAuth through the bridge (the login step currently collects a token).

## Dependency Graph

Actual imports between library packages:

```
tui/        -> flow/, render/
installer/  -> flow/, harness/, secret/, tui/
doctor/     -> update/
app/*       -> app/            (components import only the app root)
daemon/socket/ -> daemon/rpc/

independent: cli/, budget/, render/, flow/, harness/, secret/, update/,
             command/, async/, daemon/lock/, daemon/rpc/
```

Key rules: `flow/` does not depend on `app/`, the install wizard does not
depend on the app framework, and nothing in the library imports `daemon/`;
consumers wire daemons themselves.
