// Package table provides a scrollable table view for tabular data.
package table

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Column defines a column in the table.
type Column struct {
	Name  string
	Width int
}

// Row is one data row. Values correspond to columns by index.
type Row []string

// Model is the table state.
type Model struct {
	Title    string
	Columns  []Column
	Rows     []Row
	Cursor   int
	SortBy   int
	SortAsc  bool
	Viewport viewport.Model
	Width    int
	Height   int

	styleTitle  lipgloss.Style
	styleDim    lipgloss.Style
	styleCursor lipgloss.Style
	styleHeader lipgloss.Style
}

// New creates a table model.
func New(title string, columns []Column, rows []Row) *Model {
	m := &Model{
		Title:   title,
		Columns: columns,
		Rows:    rows,
		Width:   80,
		Height:  24,
		styleTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleCursor: lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
		styleHeader: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244")),
	}
	m.Viewport = viewport.New(m.Width, m.Height-4)
	m.Viewport.YPosition = 2
	m.render()
	return m
}

// SetRows updates the data and resets scroll.
func (m *Model) SetRows(rows []Row) {
	m.Rows = rows
	if m.Cursor >= len(rows) || m.Cursor < 0 {
		m.Cursor = 0
	}
	m.render()
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) render() {
	var out strings.Builder

	// Header row.
	out.WriteString("  ") // align with data rows
	for i, col := range m.Columns {
		name := col.Name
		if i == m.SortBy {
			if m.SortAsc {
				name += " \u25b2"
			} else {
				name += " \u25bc"
			}
		}
		cell := truncateCell(fmt.Sprintf("%-*s", col.Width, name), col.Width)
		out.WriteString(m.styleHeader.Render(cell))
	}
	out.WriteString("\n")

	// Data rows.
	for i, row := range m.Rows {
		if i == m.Cursor {
			out.WriteString(m.styleCursor.Render("> "))
		} else {
			out.WriteString("  ")
		}
		for j, col := range m.Columns {
			if j < len(row) {
				cell := truncateCell(fmt.Sprintf("%-*s", col.Width, row[j]), col.Width)
				out.WriteString(cell)
			}
		}
		out.WriteString("\n")
	}

	m.Viewport.SetContent(out.String())
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Viewport.Width = msg.Width
		m.Viewport.Height = msg.Height - 4
		m.render()

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
				m.render()
			}
			return nil
		case "down", "j":
			if m.Cursor < len(m.Rows)-1 {
				m.Cursor++
				m.render()
			}
			return nil
		case "home", "g":
			m.Cursor = 0
			m.Viewport.GotoTop()
			m.render()
			return nil
		case "end", "G":
			if len(m.Rows) > 0 {
				m.Cursor = len(m.Rows) - 1
				m.Viewport.GotoBottom()
				m.render()
			}
			return nil
		case "pgup":
			m.Viewport.ViewUp()
			m.render()
			return nil
		case "pgdown":
			m.Viewport.ViewDown()
			m.render()
			return nil
		case "enter":
			if len(m.Rows) > 0 {
				return app.Action("table", "select", m.Cursor)
			}
		case "esc":
			return app.Action("table", "back")

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

	if len(m.Rows) == 0 {
		out.WriteString(m.styleDim.Render("  No data."))
		return out.String()
	}

	out.WriteString(m.Viewport.View())
	out.WriteString("\n")
	out.WriteString(m.styleDim.Render("  \u2191\u2193 move \u00b7 enter select \u00b7 esc back"))
	return out.String()
}

func truncateCell(s string, width int) string {
	if width < 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width])
}
