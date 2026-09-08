package app

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Init handles the common init for AppModel. Call this from the
// app's Init method.
func (m *AppModel) Init() tea.Cmd {
	return nil
}

// HandleGlobalKeys processes messages that apply across all screens.
//
// It returns handled=true when the message was consumed and the caller
// should return immediately with cmd. For ctrl+c it sets Quit and returns
// tea.Quit; nothing else in the framework exits the program, so callers
// must return the command they are given:
//
//	if handled, cmd := m.HandleGlobalKeys(msg); handled {
//	    return m, cmd
//	}
//
// WindowSizeMsg updates Width/Height but is not consumed, so screens can
// react to it too.
func (m *AppModel) HandleGlobalKeys(msg tea.Msg) (handled bool, cmd tea.Cmd) {
	if m == nil {
		return false, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quit = true
			return true, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return false, nil
	}
	return false, nil
}

// ScrollIntoView adjusts vp's YOffset so that the span of lines starting at
// line (span lines long) is visible. Components whose cursor moves through
// content taller than the viewport call this after every cursor change so the
// cursor never leaves the visible window.
func ScrollIntoView(vp *viewport.Model, line, span int) {
	if vp == nil || vp.Height <= 0 {
		return
	}
	if span < 1 {
		span = 1
	}
	if line < 0 {
		line = 0
	}
	if line < vp.YOffset {
		vp.SetYOffset(line)
		return
	}
	bottom := line + span - 1
	if bottom >= vp.YOffset+vp.Height {
		vp.SetYOffset(bottom - vp.Height + 1)
	}
}
