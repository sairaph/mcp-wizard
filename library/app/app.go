package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Init handles the common init for AppModel. Call this from the
// app's Init method.
func (m *AppModel) Init() tea.Cmd {
	return nil
}

// HandleGlobalKeys processes keys that work across all screens.
// Returns true if the key was handled (caller should return immediately).
func (m *AppModel) HandleGlobalKeys(msg tea.Msg) bool {
	if m == nil {
		return false
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quit = true
			return true
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return false // let the screen handle it too
	}
	return false
}
