// Package menu provides a dynamic menu component.
package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Item is one entry in the menu.
type Item struct {
	Label  string
	Action string // identifier returned when selected
}

// Model is the menu state.
type Model struct {
	Title   string
	Items   []Item
	Cursor  int
	BuildFn func() []Item

	styleTitle  lipgloss.Style
	styleCursor lipgloss.Style
	styleDim    lipgloss.Style
}

// New creates a menu model. buildFn is called before each render
// to dynamically rebuild the item list. May be nil.
func New(title string, buildFn func() []Item) *Model {
	return &Model{
		Title:   title,
		BuildFn: buildFn,
		styleTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleCursor: lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
		styleDim:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	}
}

func (m *Model) rebuild() {
	if m.BuildFn != nil {
		m.Items = m.BuildFn()
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if len(m.Items) == 0 {
		return
	}
	if m.Cursor >= len(m.Items) {
		m.Cursor = len(m.Items) - 1
	}
}

func (m *Model) Init() tea.Cmd {
	m.rebuild()
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	m.rebuild()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case "q", "ctrl+c":
			return app.Action("menu", "quit")
		case "enter":
			if m.Cursor >= 0 && m.Cursor < len(m.Items) {
				return app.Action("menu", "select", m.Items[m.Cursor].Action)
			}
		}
	}
	return nil
}

func (m *Model) View() string {
	m.rebuild()
	var out strings.Builder
	out.WriteString(m.styleTitle.Render("  " + m.Title) + "\n\n")
	for i, item := range m.Items {
		prefix := "  "
		if i == m.Cursor {
			prefix = m.styleCursor.Render("> ")
		}
		out.WriteString(prefix + item.Label + "\n")
	}
	out.WriteString("\n" + m.styleDim.Render("  \u2191\u2193 move \u00b7 enter select \u00b7 q quit"))
	return out.String()
}
