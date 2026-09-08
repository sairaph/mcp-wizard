package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var spinFrames = []string{"-", "\\", "|", "/"}

type spinMsg struct{}

// Spinner returns a tick command that drives spinner animation; forward the
// resulting message to IsSpinMsg and re-issue Spinner while spinning.
func Spinner() tea.Cmd {
	return tea.Tick(110*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

// SpinFrame returns the spinner glyph for the given frame counter.
func SpinFrame(frame int) string {
	if frame < 0 {
		frame = 0
	}
	return spinFrames[frame%len(spinFrames)]
}

// IsSpinMsg reports whether msg is a spinner tick.
func IsSpinMsg(msg tea.Msg) bool {
	_, ok := msg.(spinMsg)
	return ok
}
