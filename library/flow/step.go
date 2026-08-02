// Package flow provides the Step abstraction and Flow runner for
// multi-screen wizards. Steps are handlers over shared state, not
// swappable models — the Flow owns the single tea.Model.
package flow

import tea "github.com/charmbracelet/bubbletea"

// Directive tells the Flow what to do after a step handles a message.
type Directive int

const (
	Continue Directive = iota // stay on this step
	Next                      // advance to the next step
	Back                      // return to the previous step
	Skip                      // advance without rendering (conditional)
	Jump                      // state.NextStep names the target step ID
	Quit                      // user cancelled
	Fail                      // fatal; state.Failure carries the error
)

// Step is one screen in a wizard. View returns content only —
// the framework wraps it in chrome (header, section title, footer).
type Step[T any] interface {
	// ID returns a stable identifier for this step.
	ID() string

	// Title returns the section heading shown for this step.
	Title(state *T) string

	// Hints returns the key-hint entries shown in the footer.
	Hints(state *T) []struct{ Key, Label string }

	// Init returns the initial command(s) for this step.
	Init(state *T) tea.Cmd

	// Update handles a message and returns a directive plus commands.
	Update(msg tea.Msg, state *T) (Directive, tea.Cmd)

	// View renders the step's content (without chrome).
	View(state *T) string
}
