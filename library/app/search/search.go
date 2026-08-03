// Package search provides a search screen with query input and results.
package search

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Result is one search result item.
type Result struct {
	Label   string
	Detail  string
	MatchID string // identifier for selection
}

// Model is the search state.
type Model struct {
	Title    string
	Query    string
	Input    textinput.Model
	Results  []Result
	Cursor   int
	Viewport viewport.Model
	Width    int
	Height   int
	Searching bool

	styleTitle  lipgloss.Style
	styleDim    lipgloss.Style
	styleCursor lipgloss.Style
}

// New creates a search model.
func New(title string) *Model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()

	return &Model{
		Title:   title,
		Input:   ti,
		Width:   80,
		Height:  24,
		styleTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleCursor: lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	}
}

// SetResults updates the result list.
func (m *Model) SetResults(results []Result) {
	m.Results = results
	m.Cursor = 0
	m.Searching = false
	m.renderResults()
}

func (m *Model) renderResults() {
	var out strings.Builder
	for i, r := range m.Results {
		prefix := "  "
		if i == m.Cursor {
			prefix = m.styleCursor.Render("> ")
		}
		fmt.Fprintf(&out, "%s%s\n", prefix, r.Label)
		if r.Detail != "" {
			fmt.Fprintf(&out, "    %s\n", m.styleDim.Render(r.Detail))
		}
	}
	m.Viewport.SetContent(out.String())
}

func (m *Model) Update(msg tea.Msg) (tea.Cmd, error) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Viewport.Width = msg.Width
		m.Viewport.Height = msg.Height - 6

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.Searching {
				return nil, nil
			}
			m.Searching = true
			m.Query = m.Input.Value()
			return nil, fmt.Errorf("search:query:%s", m.Query)
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
				m.renderResults()
			}
		case "down", "j":
			if m.Cursor < len(m.Results)-1 {
				m.Cursor++
				m.renderResults()
			}
		case "esc":
			return nil, fmt.Errorf("search:cancelled")
		}
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return cmd, nil
}

func (m *Model) View() string {
	var out strings.Builder
	out.WriteString(m.styleTitle.Render("  " + m.Title) + "\n\n")
	out.WriteString("  " + m.Input.View() + "\n")

	if m.Searching {
		out.WriteString("\n" + m.styleDim.Render("  Searching..."))
		return out.String()
	}

	if len(m.Results) > 0 {
		out.WriteString("\n")
		out.WriteString(m.Viewport.View())
		out.WriteString("\n")
		out.WriteString(m.styleDim.Render(fmt.Sprintf("  %d results  ↑↓ move · enter select · esc back", len(m.Results))))
	}

	return out.String()
}
