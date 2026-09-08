package tui

import (
	"strings"
)

// TextInput renders a one-line text field, masking the value when masked
// is true and showing placeholder when the value is empty.
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
