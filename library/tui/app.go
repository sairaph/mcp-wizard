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
	if opts.Theme.Copy.Title == "" {
		opts.Theme = DefaultTheme
		if opts.Title != "" {
			opts.Theme.Copy.Title = opts.Title
		}
	}

	model := f.Model()
	program := tea.NewProgram(model, tea.WithContext(ctx),
		tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	if _, err := program.Run(); err != nil {
		return 1
	}
	return f.ExitCode()
}
