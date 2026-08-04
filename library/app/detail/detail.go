// Package detail provides a scrollable read-only content viewer.
package detail

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Model is the detail view state.
type Model struct {
	Title    string
	Content  string
	Viewport viewport.Model
	Width    int
	Height   int

	styleTitle lipgloss.Style
	styleDim   lipgloss.Style
}

// New creates a detail view with the given title and content.
func New(title, content string) *Model {
	m := &Model{
		Title:   title,
		Content: content,
		Width:   80,
		Height:  24,
		styleTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	}
	m.Viewport = viewport.New(m.Width, m.Height-3)
	m.Viewport.YPosition = 1
	m.Viewport.SetContent(content)
	return m
}

// SetContent updates the displayed content and resets scroll.
func (m *Model) SetContent(content string) {
	m.Content = content
	m.Viewport.SetContent(content)
	m.Viewport.GotoTop()
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Viewport.Width = msg.Width
		h := msg.Height - 3
		if h < 1 {
			h = 1
		}
		m.Viewport.Height = h
		m.Viewport.SetContent(m.Content)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return app.Action("detail", "back")
		}
	}

	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return cmd
}

func (m *Model) View() string {
	var out strings.Builder
	if m.Title != "" {
		out.WriteString(m.styleTitle.Render("  " + m.Title) + "\n")
	}
	out.WriteString(m.Viewport.View())
	out.WriteString("\n")
	out.WriteString(m.styleDim.Render(fmt.Sprintf("  %.0f%%  \u2191\u2193 pgup/pgdn scroll \u00b7 esc back", m.Viewport.ScrollPercent()*100)))
	return out.String()
}
