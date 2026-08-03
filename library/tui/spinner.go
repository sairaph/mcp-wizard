package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var spinFrames = []string{"-", "\\", "|", "/"}

type spinMsg struct{}

func Spinner() tea.Cmd {
	return tea.Tick(110*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

func SpinFrame(frame int) string {
	if frame < 0 {
		frame = 0
	}
	return spinFrames[frame%len(spinFrames)]
}

func IsSpinMsg(msg tea.Msg) bool {
	_, ok := msg.(spinMsg)
	return ok
}
