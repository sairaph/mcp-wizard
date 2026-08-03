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
	theme := opts.Theme
	// Only default when no theme fields are set — partial overrides are respected.
	if theme == (Theme{}) {
		theme = DefaultTheme
	}
	if opts.Title != "" {
		theme.Copy.Title = opts.Title
	}

	model := f.Model()
	program := tea.NewProgram(model, tea.WithContext(ctx),
		tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	if _, err := program.Run(); err != nil {
		return 1
	}
	return f.ExitCode()
}
