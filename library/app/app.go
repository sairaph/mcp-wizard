package app

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Options controls the application.
type Options struct {
	Title   string
	Version string
}

// stack is a simple LIFO stack for screen navigation.
type stack struct {
	items []Screen
}

func (s *stack) push(item Screen) {
	s.items = append(s.items, item)
}

func (s *stack) pop() Screen {
	if len(s.items) == 0 {
		return nil
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}

func (s *stack) peek() Screen {
	if len(s.items) == 0 {
		return nil
	}
	return s.items[len(s.items)-1]
}

func (s *stack) empty() bool {
	return len(s.items) == 0
}

// Run starts the application with the given initial screens.
// Returns exit code 0 on success, 1 on failure.
func Run(ctx context.Context, initial []Screen, opts Options) int {
	nav := &stack{}
	for _, screen := range initial {
		nav.push(screen)
	}

	model := &appModel{
		nav:  nav,
		base: &BaseModel{},
	}

	program := tea.NewProgram(model, tea.WithContext(ctx),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout))

	if _, err := program.Run(); err != nil {
		return 1
	}
	if model.base.Failure != "" {
		return 1
	}
	return 0
}

// appModel is the Bubble Tea model that drives the application.
type appModel struct {
	nav  *stack
	base *BaseModel
}

func (m *appModel) Init() tea.Cmd {
	if screen := m.nav.peek(); screen != nil {
		cmd := screen.Init()
		focusCmd := screen.Focus()
		return tea.Batch(cmd, focusCmd)
	}
	return tea.Quit
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.base.Width = msg.Width
		m.base.Height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.base.Quit = true
			return m, tea.Quit
		}
	}

	screen := m.nav.peek()
	if screen == nil {
		return m, tea.Quit
	}

	cmd, err := screen.Update(msg)
	if err != nil {
		m.base.Failure = err.Error()
		return m, tea.Quit
	}

	// Handle navigation messages.
	switch msg := msg.(type) {
	case pushMsg:
		if msg.screen != nil {
			if current := m.nav.peek(); current != nil {
				current.Blur()
			}
			m.nav.push(msg.screen)
			return m, tea.Batch(msg.screen.Init(), msg.screen.Focus())
		}
	case popMsg:
		if current := m.nav.peek(); current != nil {
			current.Blur()
		}
		m.nav.pop()
		if screen := m.nav.peek(); screen != nil {
			return m, screen.Focus()
		}
		return m, tea.Quit
	case replaceMsg:
		if current := m.nav.peek(); current != nil {
			current.Blur()
		}
		m.nav.pop()
		if msg.screen != nil {
			m.nav.push(msg.screen)
			return m, tea.Batch(msg.screen.Init(), msg.screen.Focus())
		}
		return m, tea.Quit
	}

	if m.base.Quit {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *appModel) View() string {
	screen := m.nav.peek()
	if screen == nil {
		return ""
	}
	return screen.View(m.base.Width, m.base.Height)
}

// PushMsg tells the app to push a new screen onto the navigation stack.
// Send this from a screen's Update to navigate forward.
type pushMsg struct {
	screen Screen
}

// Push returns a command that pushes a new screen onto the stack.
func Push(s Screen) tea.Cmd {
	return func() tea.Msg {
		return pushMsg{screen: s}
	}
}

// PopMsg tells the app to pop the current screen and go back.
type popMsg struct{}

// Pop returns a command that pops the current screen.
func Pop() tea.Cmd {
	return func() tea.Msg {
		return popMsg{}
	}
}

// ReplaceMsg tells the app to replace the current screen (no history).
type replaceMsg struct {
	screen Screen
}

// Replace returns a command that replaces the current screen.
func Replace(s Screen) tea.Cmd {
	return func() tea.Msg {
		return replaceMsg{screen: s}
	}
}

// Errorf returns a command that sets the failure state and quits.
func Errorf(format string, args ...any) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{message: fmt.Sprintf(format, args...)}
	}
}

type errorMsg struct {
	message string
}
