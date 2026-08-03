// Package confirm provides a confirmation dialog screen.
package confirm

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Model is the confirmation dialog state.
type Model struct {
	title   string
	detail  string
	confirm string // label for affirmative
	cancel  string // label for negative
	choice  int    // 0 = confirm, 1 = cancel
}

// New creates a confirmation dialog.
// confirm is the affirmative button label, cancel is the negative.
func New(title, detail, confirm, cancel string) app.Screen {
	return &Model{
		title:   title,
		detail:  detail,
		confirm: confirm,
		cancel:  cancel,
	}
}

func (m *Model) ID() string { return "confirm" }

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Focus() tea.Cmd { return nil }
func (m *Model) Blur() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Cmd, error) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "down", "j":
			m.choice = 1 - m.choice
		case "enter":
			if m.choice == 0 {
				return nil, fmt.Errorf("confirm:confirmed")
			}
			return nil, fmt.Errorf("confirm:cancelled")
		case "esc", "q":
			return nil, fmt.Errorf("confirm:cancelled")
		}
	}
	return nil, nil
}

func (m *Model) View(width, height int) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))

	var out strings.Builder
	out.WriteString("\n")
	out.WriteString(style.Render("  " + m.title))
	out.WriteString("\n\n")
	if m.detail != "" {
		out.WriteString("  " + dimStyle.Render(m.detail) + "\n\n")
	}

	options := []string{m.confirm, m.cancel}
	for i, opt := range options {
		prefix := "  "
		if i == m.choice {
			prefix = cursorStyle.Render("> ")
		}
		fmt.Fprintf(&out, "%s%s\n", prefix, opt)
	}

	out.WriteString("\n")
	out.WriteString(dimStyle.Render("  ↑↓ move · enter select · esc cancel"))
	return out.String()
}
