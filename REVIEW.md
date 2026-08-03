# Critical Design Review: App Framework

Review of the new `app/`, `daemon/`, and `command/` packages against the three
reference projects (interactive-terminal-mcp, sana-mcp, favro-mcp).

---

## GAP 1 — Navigation model is wrong

**Problem:** The library uses a stack (`Push`/`Pop`/`Replace`). No reference
project uses a stack. All three use a `step`/`screen` enum with a central
`switch` statement:

| App | Pattern |
|---|---|
| interactive-terminal-mcp | `screen` enum + `switch m.screen` |
| sana-mcp | Menu loop + `tea.NewProgram` per sub-screen |
| favro-mcp | `step` enum + `switch m.step` |

The stack fails for:
- **Ephemeral screens**: Loading/verifying screens that render without user
  interaction and auto-advance.
- **Data reload on back**: favro-mcp's card detail → esc → reload cards.
  `Pop` has no mechanism to run a command on the resumed screen.
- **sana-mcp's sub-programs**: Each screen is its own `tea.Program`. The
  stack assumes a single program.

**Fix:** Replace the stack with a `step` enum + `switch` dispatcher matching
favro-mcp's pattern. Store sub-component state as optional embedded fields
(rather than independent Screen objects). Make the framework a state machine
host, not a navigation framework.

---

## GAP 2 — Screen interface is incompatible with tea.Model

**Problem:** The `Screen` interface has `View(width, height int) string` and
`Update(msg) (tea.Cmd, error)`. But Bubble Tea's `tea.Model` requires
`View() string` (no args) and `Update(msg) (tea.Model, tea.Cmd)`. This means:

1. Screen components cannot be used outside the framework (e.g. in a
   standalone `tea.Program` like sana-mcp's browser).
2. The `error` return forces all errors through the fatal `base.Failure`
   path. Recoverable errors (validation failure, network timeout) can't be
   handled per-screen.
3. `ID()`, `Focus()`, `Blur()` are dead code in all current components.

**Fix:** Drop the `Screen` interface. Provide a single `AppModel` struct
with a `Step` enum and optional embedded sub-models, matching favro-mcp's
`appModel`. Export helper functions for step transitions.

---

## GAP 3 — Error-based action dispatch is fragile

**Problem:** Components signal actions by returning
`fmt.Errorf("menu:select:%s", action)`. The app model would need to
`strings.Contains` on error strings to dispatch — brittle, type-unsafe, and
breaks if error messages are ever localized.

**Fix:** Replace with a typed `ActionMsg`:

```go
type ActionMsg struct {
    Source string // "menu", "form", "confirm", etc.
    Value  string // "select:boards", "submitted", "confirmed"
    Data   any
}

func Action(source, value string, data ...any) tea.Cmd
```

Components return `app.Action(...)` as a `tea.Cmd`. The app model catches
`ActionMsg` in the main `switch` and dispatches.

Affected files:
- `app/menu/menu.go:72`
- `app/form/form.go:120`
- `app/confirm/confirm.go:47`
- `app/list/list.go:131`
- `app/search/search.go:94`
- `app/table/table.go:139`

---

## GAP 4 — Daemon lifecycle has no IPC protocol

**Problem:** The daemon package provides `Open`/`Close`/`Serve` but no
support for:
- **Unix socket IPC** (interactive-terminal-mcp): JSON-RPC over socket,
  client connections, session registry, idle shutdown.
- **Client connection** (`Dial`, `Call`, stream attach).
- **Heartbeat / sync loop** (sana-mcp): periodic work with backfill
  detection, status file updates.

The three reference projects have fundamentally different daemon
architectures. A single abstraction cannot serve all three.

**Fix:** Remove the abstract daemon package. Provide two concrete
implementations:
- `daemon/socket/` — Unix socket + JSON-RPC + client registry
  (interactive-terminal-mcp pattern)
- `daemon/lock/` — File lock + periodic work loop (sana-mcp pattern)

---

## GAP 5 — Command registry conflates standalone and daemon ops

**Problem:** The command registry maps names to handler functions. This
matches favro-mcp's one-shot commands (`list-boards`, `get-card`). But
interactive-terminal-mcp has commands like `ls`, `new`, `read`, `send`
that require an IPC client connection to the daemon. The registry has no
concept of a required runtime or client.

**Fix:** Add a `RequiresDaemon bool` field to `Handler`. Change `Dispatch`
to accept an optional connection. Or route daemon operations through the
CLI parser directly (matching interactive-terminal-mcp's pattern).

---

## GAP 6 — Component completeness

**Covered (7):**
| Component | Used by |
|---|---|
| Menu | All three |
| List | favro-mcp cards, interactive-terminal-mcp sessions |
| Detail | favro-mcp card detail |
| Form | favro-mcp login |
| Search | sana-mcp search |
| Table | favro-mcp users/tags |
| Confirm | interactive-terminal-mcp delete |

**Missing (~12):**
| Feature | Source | What it does |
|---|---|---|
| Terminal emulator | interactive-terminal-mcp | vterm + composer + raw mode |
| Paginated card list | favro-mcp | Next/prev page navigation |
| Meeting cards | sana-mcp | 3-row card layout |
| Transcript + line editing | sana-mcp | Per-line correction |
| Search highlighting | sana-mcp | Term highlight in results |
| Sync status | sana-mcp | Multi-line daemon status |
| Board picker | favro-mcp | 2-step collection → board |
| Settings screen | interactive-terminal-mcp | Two-panel: harnesses + settings |
| Inline rename | interactive-terminal-mcp | Text input over list item |
| Filterable list | sana-mcp | `/` name filter, `f` status filter |
| Card detail sections | favro-mcp | Description, checklists, comments |

---

## GAP 7 — No async loading pattern

**Problem:** All three reference apps define typed async message structs
and commands (`orgsLoadedMsg`, `meetingsLoadedMsg`, `cardsLoadedMsg`, etc.).
The library components hold data synchronously — `SetItems()`, `SetResults()`
— with no mechanism for triggering async loads or handling loading states.

**Fix:** Add an `Async` helper package:

```go
package async

type Result[T any] struct {
    Value T
    Err   error
}

func Load[T any](fn func() (T, error)) tea.Cmd {
    return func() tea.Msg { v, e := fn(); return Result[T]{v, e} }
}

// LoadingMsg is a generic loading state marker.
type LoadingMsg struct{}
```

This is a convenience, not a framework — apps still define their own typed
messages for domain data. The helper eliminates the boilerplate of writing
the closure and message struct for simple cases.

---

## Summary of fixes needed

| Gap | Severity | Effort | What to change |
|---|---|---|---|
| 1 — Navigation | **High** | Large | Replace stack with step enum + switch |
| 2 — Screen interface | **High** | Large | Drop Screen, use single AppModel |
| 3 — Error dispatch | **High** | Medium | Add ActionMsg, update all components |
| 4 — Daemon IPC | **High** | Large | Split into socket/ and lock/ sub-packages |
| 5 — Command kind | **Medium** | Small | Add RequiresDaemon field |
| 6 — Components | **Medium** | Large | Add paginator, settings, async helpers |
| 7 — Async loading | **Medium** | Small | Add async helper package |
