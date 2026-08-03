// Package app provides the TUI application framework for user-facing
// interactive apps (not the install wizard). It uses stack-based
// navigation where each screen owns its own model.
package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Screen is one view in the app. Unlike flow.Step[T], it owns its own
// model and receives the full tea.Msg, not just shared state.
type Screen interface {
	// ID returns a stable identifier for routing.
	ID() string

	// Init returns the initial command(s) when the screen is first shown.
	Init() tea.Cmd

	// Update handles a message.
	Update(msg tea.Msg) (tea.Cmd, error)

	// View renders the screen content (without chrome — the app wraps it).
	View(width, height int) string

	// Focus is called when the screen becomes active (pushed or popped to).
	Focus() tea.Cmd

	// Blur is called when the screen becomes inactive (another screen pushed).
	Blur() tea.Cmd
}
