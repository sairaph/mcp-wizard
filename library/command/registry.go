// Package command provides a one-shot CLI command registry.
// Commands registered here share business logic with the TUI app
// and are dispatched from main.go's default branch.
package command

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Handler is a one-shot CLI command.
type Handler struct {
	// Name is the primary command name (e.g. "list-boards").
	Name string

	// Aliases are alternative names (e.g. "boards").
	Aliases []string

	// Description is shown in usage output.
	Description string

	// Usage is a one-line syntax hint (e.g. "list-boards [--page N]").
	Usage string

	// RequiresDaemon is true if the command needs a running daemon.
	RequiresDaemon bool

	// Run executes the command. Return 0 for success, 1 for failure.
	Run func(ctx context.Context, args []string) int
}

// Registry maps command names to handlers.
type Registry struct {
	handlers map[string]*Handler
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{handlers: make(map[string]*Handler)}
}

// Register adds a handler. Panics if the name or any alias is already registered.
func (r *Registry) Register(h Handler) {
	if h.Name == "" {
		panic("command: Register requires a non-empty Name")
	}
	if h.Run == nil {
		panic("command: Register requires a non-nil Run function")
	}
	// Validate all names up front before any insertion.
	if _, exists := r.handlers[h.Name]; exists {
		panic(fmt.Sprintf("command: %q is already registered", h.Name))
	}
	for _, alias := range h.Aliases {
		if _, exists := r.handlers[alias]; exists {
			panic(fmt.Sprintf("command: alias %q is already registered", alias))
		}
	}
	// All clear - insert atomically.
	r.handlers[h.Name] = &h
	for _, alias := range h.Aliases {
		r.handlers[alias] = &h
	}
}

// Dispatch runs a command by name. Returns (handled, exitCode).
func (r *Registry) Dispatch(ctx context.Context, name string, args []string) (bool, int) {
	h, ok := r.handlers[name]
	if !ok {
		return false, 0
	}
	return true, h.Run(ctx, args)
}

// IsOneShot reports whether name is a registered command.
func (r *Registry) IsOneShot(name string) bool {
	_, ok := r.handlers[name]
	return ok
}

// List returns all unique handlers sorted by name.
func (r *Registry) List() []Handler {
	seen := make(map[string]bool)
	var result []Handler
	for _, h := range r.handlers {
		if seen[h.Name] {
			continue
		}
		seen[h.Name] = true
		result = append(result, *h)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// PrintUsage writes usage text for all registered commands to w.
func (r *Registry) PrintUsage(w io.Writer) {
	commands := r.List()
	if len(commands) == 0 {
		return
	}
	maxLen := 0
	for _, c := range commands {
		if l := len(c.Name); l > maxLen {
			maxLen = l
		}
	}
	fmt.Fprintf(w, "Commands:\n")
	for _, c := range commands {
		padding := strings.Repeat(" ", maxLen-len(c.Name)+2)
		fmt.Fprintf(w, "  %s%s%s\n", c.Name, padding, c.Description)
	}
}
