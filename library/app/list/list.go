// Package list provides a scrollable, paginated list screen.
package list

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Item is one row in the list.
type Item struct {
	Label  string // primary display text
	Detail string // optional secondary line
	Active bool   // green dot vs dim dot
}

// Model is the list state.
type Model struct {
	Title    string
	Items    []Item
	Cursor   int
	Page     int
	Total    int
	PerPage  int
	Viewport viewport.Model
	Width    int
	Height   int
	Loading  bool

	styleTitle  lipgloss.Style
	styleDim    lipgloss.Style
	styleOn     lipgloss.Style
	styleOff    lipgloss.Style
	styleCursor lipgloss.Style
}

// New creates a list model.
func New(title string, items []Item, page, total, perPage int) *Model {
	m := &Model{
		Title:    title,
		Items:    items,
		Page:     page,
		Total:    total,
		PerPage:  perPage,
		Width:    80,
		Height:   24,
		styleTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleOn:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		styleOff:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleCursor: lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	}
	m.Viewport = viewport.New(m.Width, m.Height-4)
	m.Viewport.YPosition = 2
	return m
}

// SetItems updates the item list and resets the cursor.
func (m *Model) SetItems(items []Item) {
	m.Items = items
	if m.Cursor >= len(items) {
		m.Cursor = 0
	}
	m.render()
}

// Selected returns the index of the selected item. Returns -1 if empty.
func (m *Model) Selected() int {
	if len(m.Items) == 0 {
		return -1
	}
	return m.Cursor
}

func (m *Model) render() {
	var out strings.Builder
	for i, item := range m.Items {
		prefix := "  "
		if i == m.Cursor {
			prefix = m.styleCursor.Render("> ")
		}
		dot := m.styleOff.Render("○")
		if item.Active {
			dot = m.styleOn.Render("●")
		}
		line := fmt.Sprintf("%s %s %s", prefix, dot, item.Label)
		out.WriteString(line + "\n")
		if item.Detail != "" {
			out.WriteString("    " + m.styleDim.Render(item.Detail) + "\n")
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
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
				m.render()
			}
		case "pgup":
			m.Viewport.ViewUp()
		case "pgdown":
			m.Viewport.ViewDown()
		case "home", "g":
			m.Cursor = 0
			m.render()
		case "end", "G":
			m.Cursor = len(m.Items) - 1
			m.render()
		case "enter":
			if len(m.Items) > 0 {
				return nil, fmt.Errorf("list:select:%d", m.Cursor)
			}
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

	if m.Loading {
		out.WriteString(m.styleDim.Render("  Loading..."))
		return out.String()
	}

	if len(m.Items) == 0 {
		out.WriteString(m.styleDim.Render("  No items."))
		return out.String()
	}

	out.WriteString(m.Viewport.View())
	out.WriteString("\n")
	out.WriteString(m.styleDim.Render(fmt.Sprintf("  %d items (page %d/%d)  ↑↓ move · enter select", len(m.Items), m.Page+1, (m.Total+m.PerPage-1)/m.PerPage)))
	return out.String()
}
