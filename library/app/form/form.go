// Package form provides a multi-field form component.
package form

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/app"
)

// Field is one input in a form.
type Field struct {
	Label    string
	Value    string
	Secret   bool
	Validate func(string) error
}

// Model is the form state.
type Model struct {
	Title   string
	Fields  []Field
	Inputs  []textinput.Model
	Focused int
	Error   string
	Submitted bool

	styleTitle lipgloss.Style
	styleDim   lipgloss.Style
	styleError lipgloss.Style
}

// New creates a form with the given title and fields.
func New(title string, fields []Field) *Model {
	m := &Model{
		Title:   title,
		Fields:  fields,
		Focused: 0,
		styleTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		styleDim:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		styleError: lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
	}

	for _, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Label
		if f.Secret {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '\u2022'
		}
		if f.Value != "" {
			ti.SetValue(f.Value)
		}
		m.Inputs = append(m.Inputs, ti)
	}

	if len(m.Inputs) > 0 {
		m.Inputs[0].Focus()
	}

	return m
}

// Values returns the form values as a map keyed by field label.
func (m *Model) Values() map[string]string {
	result := make(map[string]string, len(m.Fields))
	for i, f := range m.Fields {
		if i < len(m.Inputs) {
			result[f.Label] = m.Inputs[i].Value()
		}
	}
	return result
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.Submitted {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if len(m.Inputs) == 0 {
				break
			}
			m.Focused = (m.Focused + 1) % len(m.Inputs)
			for i := range m.Inputs {
				if i == m.Focused {
					m.Inputs[i].Focus()
				} else {
					m.Inputs[i].Blur()
				}
			}
			return nil
		case "shift+tab", "up":
			if len(m.Inputs) == 0 {
				break
			}
			m.Focused = (m.Focused - 1 + len(m.Inputs)) % len(m.Inputs)
			for i := range m.Inputs {
				if i == m.Focused {
					m.Inputs[i].Focus()
				} else {
					m.Inputs[i].Blur()
				}
			}
			return nil
		case "enter":
			// Validate all fields.
			for i, f := range m.Fields {
				if f.Validate != nil {
					if err := f.Validate(m.Inputs[i].Value()); err != nil {
						m.Error = err.Error()
						m.Focused = i
						for j := range m.Inputs {
							if j == i {
								m.Inputs[j].Focus()
							} else {
								m.Inputs[j].Blur()
							}
						}
						return nil
					}
				}
			}
			m.Submitted = true
			return app.Action("form", "submitted")
		case "esc":
			return app.Action("form", "cancelled")
		}
	}

	if len(m.Inputs) == 0 {
		return nil
	}
	var cmd tea.Cmd
	m.Inputs[m.Focused], cmd = m.Inputs[m.Focused].Update(msg)
	return cmd
}

func (m *Model) View() string {
	var out strings.Builder
	out.WriteString(m.styleTitle.Render("  " + m.Title) + "\n\n")

	for i, input := range m.Inputs {
		label := m.Fields[i].Label
		prefix := "  "
		if i == m.Focused {
			prefix = "> "
		}
		out.WriteString(fmt.Sprintf("%s%s:\n", prefix, label))
		out.WriteString(fmt.Sprintf("  %s\n", input.View()))
	}

	if m.Error != "" {
		out.WriteString("\n" + m.styleError.Render("  "+m.Error) + "\n")
	}

	out.WriteString("\n")
	out.WriteString(m.styleDim.Render("  tab next \u00b7 enter submit \u00b7 esc cancel"))
	return out.String()
}
