// Package paginator provides a paginated list component for navigating
// through pages of data. Used by favro-mcp for card lists.
package paginator

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Model is the paginator state.
type Model struct {
	Title   string
	Items   []string
	Cursor  int
	Page    int
	PerPage int
	Total   int

	styleTitle  lipgloss.Style
	styleDim    lipgloss.Style
	styleCursor lipgloss.Style
}

// New creates a paginator.
func New(title string, items []string, page, perPage, total int) *Model {
	return &Model{
		Title:   title,
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
		styleTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleCursor: lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	}
}

// TotalPages returns the total number of pages.
func (m *Model) TotalPages() int {
	if m.PerPage <= 0 || m.Total <= 0 {
		return 0
	}
	return (m.Total + m.PerPage - 1) / m.PerPage
}

// HasNext reports whether there is a next page.
func (m *Model) HasNext() bool { return m.Page+1 < m.TotalPages() }

// HasPrev reports whether there is a previous page.
func (m *Model) HasPrev() bool { return m.Page > 0 }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) SetItems(items []string) {
	m.Items = items
	if m.Cursor >= len(items) {
		m.Cursor = len(items) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
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
		case "n", "right":
			if m.HasNext() {
				return app.Action("paginator", "next")
			}
		case "p", "left":
			if m.HasPrev() {
				return app.Action("paginator", "prev")
			}
		case "enter":
			if len(m.Items) > 0 {
				return app.Action("paginator", "select", m.Cursor)
			}
		}
	}
	return nil
}

func (m *Model) View() string {
	var out strings.Builder
	out.WriteString(m.styleTitle.Render("  " + m.Title) + "\n\n")
	for i, item := range m.Items {
		prefix := "  "
		if i == m.Cursor {
			prefix = m.styleCursor.Render("> ")
		}
		out.WriteString(prefix + item + "\n")
	}
	// Footer
	totalPages := m.TotalPages()
	var footer string
	if totalPages > 0 {
		footer = fmt.Sprintf("  page %d/%d (%d items)", m.Page+1, totalPages, m.Total)
	} else {
		footer = fmt.Sprintf("  %d items", m.Total)
	}
	if m.HasNext() {
		footer += "  n next"
	}
	if m.HasPrev() {
		footer += "  p prev"
	}
	out.WriteString("\n" + m.styleDim.Render(footer))
	return out.String()
}
