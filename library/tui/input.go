package tui

import (
	"strings"
)

func TextInput(value, placeholder string, masked bool) string {
	display := value
	if display == "" {
		display = placeholder
	}
	if masked && display != placeholder {
		display = strings.Repeat("*", len([]rune(display)))
	}
	return "  " + display + "_"
}
