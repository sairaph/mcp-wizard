// Package table provides a scrollable table view for tabular data.
package table

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Column defines a column in the table.
type Column struct {
	Name   string
	Width  int
	Align  int // -1 left, 0 center, 1 right (text/tabwriter style)
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
	if m.Cursor >= len(rows) {
		m.Cursor = 0
	}
	m.render()
}

func (m *Model) render() {
	var out strings.Builder

	// Header row.
	for i, col := range m.Columns {
		name := col.Name
		if i == m.SortBy {
			if m.SortAsc {
				name += " ▲"
			} else {
				name += " ▼"
			}
		}
		cell := fmt.Sprintf("%-*s", col.Width, name)
		if len(cell) > col.Width {
			cell = cell[:col.Width]
		}
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
				cell := fmt.Sprintf("%-*s", col.Width, row[j])
				if len(cell) > col.Width {
					cell = cell[:col.Width]
				}
				out.WriteString(cell)
			}
		}
		out.WriteString("\n")
	}

	m.Viewport.SetContent(out.String())
}

func (m *Model) Update(msg tea.Msg) (tea.Cmd, error) {
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
		case "down", "j":
			if m.Cursor < len(m.Rows)-1 {
				m.Cursor++
				m.render()
			}
		case "pgup":
			m.Viewport.ViewUp()
		case "pgdown":
			m.Viewport.ViewDown()
		case "enter":
			if len(m.Rows) > 0 {
				return nil, fmt.Errorf("table:select:%d", m.Cursor)
			}
		case "s":
			m.SortAsc = !m.SortAsc
			m.render()
		}
	}

	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return cmd, nil
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
	out.WriteString(m.styleDim.Render("  ↑↓ move · enter select · s sort · esc back"))
	return out.String()
}
