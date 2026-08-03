// Package app provides the TUI application framework for user-facing
// interactive apps. It uses a step-enum dispatch model matching the
// pattern used by interactive-terminal-mcp, sana-mcp, and favro-mcp.
package app

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Step identifies the current screen. Apps define their own steps
// starting from StepCustom.
type Step int

const (
	StepMenu    Step = iota // built-in menu screen
	StepCustom              // first app-defined step
)

// AppModel is the single model for the entire application.
// Apps embed this and add their own steps and data fields.
type AppModel struct {
	Step    Step
	Width   int
	Height  int
	Status  string // transient status message
	Failure string // fatal error message
	Quit    bool   // set to exit the app
}

// ActionMsg is a typed message for component-to-app communication.
// Components return an ActionMsg via tea.Cmd instead of parsing error strings.
type ActionMsg struct {
	Source string // e.g. "menu", "list", "form", "confirm"
	Value  string // e.g. "select:boards", "submitted", "confirmed"
	Data   any    // optional payload (e.g. selected item index)
}

// Action returns a tea.Cmd that delivers an ActionMsg.
func Action(source, value string, data ...any) tea.Cmd {
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	return func() tea.Msg {
		return ActionMsg{Source: source, Value: value, Data: d}
	}
}

// Options controls the application.
type Options struct {
	Title   string
	Version string
}

// Run starts the application with the given model.
// opts is reserved for future use (Title, Version).
func Run(ctx context.Context, model tea.Model, opts Options) int {
	program := tea.NewProgram(model, tea.WithContext(ctx),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout))
	if _, err := program.Run(); err != nil {
		return 1
	}
	return 0
}
