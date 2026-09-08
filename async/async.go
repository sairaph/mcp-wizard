// Package async provides helpers for async data loading in TUI apps.
// It reduces the boilerplate of writing typed message structs and
// closure-based commands.
package async

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Result is a generic async result message.
type Result[T any] struct {
	Value T
	Err   error
}

// Load creates a tea.Cmd that calls fn and delivers the result as a
// typed Result[T] message.
func Load[T any](fn func() (T, error)) tea.Cmd {
	return func() tea.Msg {
		v, err := fn()
		return Result[T]{Value: v, Err: err}
	}
}

// LoadingMsg is sent when an async operation starts.
// Apps can use it to show a loading indicator.
type LoadingMsg struct{}

// Start returns a command that sends a LoadingMsg immediately,
// then runs the actual load. Use it to show a spinner before
// the data arrives.
func Start[T any](fn func() (T, error)) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return LoadingMsg{} },
		Load(fn),
	)
}
