package tui

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
)

type Options struct {
	Title string
	Theme Theme
}

func Run[T any](ctx context.Context, f *flow.Flow[T], opts Options) int {
	if f == nil {
		return 1
	}
	// opts.Theme and opts.Title are reserved for future use.

	model := f.Model()
	program := tea.NewProgram(model, tea.WithContext(ctx),
		tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	if _, err := program.Run(); err != nil {
		return 1
	}
	return f.ExitCode()
}
