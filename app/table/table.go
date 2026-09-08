// Package table provides a scrollable table view for tabular data.
package table

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
//
// SortBy is the index of the column the rows are sorted by, or -1 when the
// rows are shown in insertion order. SortAsc selects the direction. Pressing
// "s" cycles through the columns ascending, then descending, then back to
// insertion order; Sort applies the current setting programmatically.
type Model struct {
	Title    string
	Columns  []Column
	Rows     []Row
	Cursor   int
	SortBy   int
	SortAsc  bool
	original []Row // insertion order, kept so sorting can be undone
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
		Title:       title,
		Columns:     columns,
		Rows:        rows,
		original:    append([]Row(nil), rows...),
		SortBy:      -1,
		Width:       80,
		Height:      24,
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

// SetRows replaces the data. The current sort setting is re-applied.
func (m *Model) SetRows(rows []Row) {
	m.original = append([]Row(nil), rows...)
	m.Rows = rows
	if m.Cursor >= len(rows) || m.Cursor < 0 {
		m.Cursor = 0
	}
	m.Sort(m.SortBy, m.SortAsc)
}

// Sort orders the rows by column col (-1 restores insertion order) and
// re-renders. Cells are compared as strings; rows too short to have the
// column sort as empty. Rows must be supplied through New or SetRows; a
// model that was never given rows that way sorts whatever Rows holds.
func (m *Model) Sort(col int, asc bool) {
	if m.original == nil {
		m.original = append([]Row(nil), m.Rows...)
	}
	if col < 0 || col >= len(m.Columns) {
		m.SortBy = -1
		m.SortAsc = false
		m.Rows = append([]Row(nil), m.original...)
		m.render()
		return
	}
	m.SortBy = col
	m.SortAsc = asc
	cell := func(r Row) string {
		if col < len(r) {
			return r[col]
		}
		return ""
	}
	sorted := append([]Row(nil), m.original...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := cell(sorted[i]), cell(sorted[j])
		if asc {
			return a < b
		}
		return a > b
	})
	m.Rows = sorted
	m.render()
}

// cycleSort advances the sort setting: col 0 asc, col 0 desc, col 1 asc, ...,
// and finally back to insertion order.
func (m *Model) cycleSort() {
	if len(m.Columns) == 0 {
		return
	}
	switch {
	case m.SortBy < 0:
		m.Sort(0, true)
	case m.SortAsc:
		m.Sort(m.SortBy, false)
	case m.SortBy+1 < len(m.Columns):
		m.Sort(m.SortBy+1, true)
	default:
		m.Sort(-1, false)
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) followCursor() {
	// Line 0 is the header row; data row i is on line i+1. At the first row
	// keep the header in view as well.
	if m.Cursor <= 0 {
		m.Viewport.SetYOffset(0)
		return
	}
	app.ScrollIntoView(&m.Viewport, m.Cursor+1, 1)
}

// moveCursor moves the cursor by delta, clamped to the row range.
func (m *Model) moveCursor(delta int) {
	if len(m.Rows) == 0 {
		m.Cursor = 0
		return
	}
	m.Cursor += delta
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= len(m.Rows) {
		m.Cursor = len(m.Rows) - 1
	}
}

func (m *Model) render() {
	var out strings.Builder

	// Header row.
	out.WriteString("  ") // align with data rows
	for i, col := range m.Columns {
		name := col.Name
		if i == m.SortBy && m.SortBy >= 0 && len(m.original) > 0 {
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
	m.followCursor()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Viewport.Width = msg.Width
		h := msg.Height - 4
		if h < 1 {
			h = 1
		}
		m.Viewport.Height = h
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
			m.render()
			return nil
		case "end", "G":
			if len(m.Rows) > 0 {
				m.Cursor = len(m.Rows) - 1
				m.render()
			}
			return nil
		case "pgup":
			m.moveCursor(-m.Viewport.Height)
			m.render()
			return nil
		case "pgdown":
			m.moveCursor(m.Viewport.Height)
			m.render()
			return nil
		case "s":
			m.cycleSort()
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
		out.WriteString(m.styleTitle.Render("  "+m.Title) + "\n")
	}

	if len(m.Rows) == 0 {
		out.WriteString(m.styleDim.Render("  No data."))
		return out.String()
	}

	out.WriteString(m.Viewport.View())
	out.WriteString("\n")
	out.WriteString(m.styleDim.Render("  \u2191\u2193 move \u00b7 s sort \u00b7 enter select \u00b7 esc back"))
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
