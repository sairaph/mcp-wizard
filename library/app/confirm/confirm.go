package confirm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

type Model struct {
	Title   string
	Detail  string
	Confirm string
	Cancel  string
	Choice  int // 0 = confirm, 1 = cancel

	styleTitle  lipgloss.Style
	styleDim    lipgloss.Style
	styleCursor lipgloss.Style
}

func New(title, detail, confirm, cancel string) *Model {
	return &Model{
		Title:   title,
		Detail:  detail,
		Confirm: confirm,
		Cancel:  cancel,
		styleTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleCursor: lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "down", "j":
			m.Choice = 1 - m.Choice
		case "enter":
			if m.Choice == 0 {
				return app.Action("confirm", "confirmed")
			}
			return app.Action("confirm", "cancelled")
		case "esc", "q":
			return app.Action("confirm", "cancelled")
		}
	}
	return nil
}

func (m *Model) View() string {
	if m.Choice < 0 || m.Choice > 1 {
		m.Choice = 0
	}
	var out strings.Builder
	out.WriteString(m.styleTitle.Render("  " + m.Title) + "\n\n")
	if m.Detail != "" {
		out.WriteString("  " + m.styleDim.Render(m.Detail) + "\n\n")
	}
	options := []string{m.Confirm, m.Cancel}
	for i, opt := range options {
		prefix := "  "
		if i == m.Choice {
			prefix = m.styleCursor.Render("> ")
		}
		out.WriteString(prefix + opt + "\n")
	}
	out.WriteString("\n" + m.styleDim.Render("  \u2191\u2193 move \u00b7 enter select \u00b7 esc cancel"))
	return out.String()
}
