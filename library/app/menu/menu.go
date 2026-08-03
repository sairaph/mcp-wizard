// Package menu provides a dynamic menu screen for the app framework.
package menu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Item is one entry in the menu.
type Item struct {
	// Label is the display text.
	Label string

	// Action is the identifier returned when this item is selected.
	// The app's Update checks this in a pushMsg or popMsg.
	Action string
}

// Model is the menu state.
type Model struct {
	title   string
	items   []Item
	cursor  int
	buildFn func() []Item // called before each View to rebuild the list
}

// New creates a new menu screen. buildFn is called before each render
// to dynamically build the item list based on current state.
func New(title string, buildFn func() []Item) app.Screen {
	return &Model{
		title:   title,
		buildFn: buildFn,
	}
}

func (m *Model) ID() string { return "menu" }

func (m *Model) Init() tea.Cmd {
	if m.buildFn != nil {
		m.items = m.buildFn()
	}
	return nil
}

func (m *Model) Focus() tea.Cmd {
	if m.buildFn != nil {
		m.items = m.buildFn()
	}
	return nil
}

func (m *Model) Blur() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Cmd, error) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				return nil, fmt.Errorf("menu:select:%s", m.items[m.cursor].Action)
			}
		case "q", "ctrl+c":
			return nil, fmt.Errorf("menu:quit")
		}
	}
	return nil, nil
}

func (m *Model) View(width, height int) string {
	if m.buildFn != nil {
		m.items = m.buildFn()
	}

	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var out strings.Builder
	out.WriteString("\n")
	out.WriteString(style.Render("  " + m.title))
	out.WriteString("\n\n")

	for i, item := range m.items {
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("> ")
		}
		fmt.Fprintf(&out, "%s%s\n", prefix, item.Label)
	}

	if height > len(m.items)+4 {
		out.WriteString("\n")
		out.WriteString(dimStyle.Render("  ↑↓ move · enter select · q quit"))
	}

	return out.String()
}
