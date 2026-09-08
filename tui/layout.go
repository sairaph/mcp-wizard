package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// Section renders the wizard chrome: the theme title, an optional section
// title, and the content.
func Section(theme Theme, title, content string) string {
	s := theme.Styles()
	head := "\n" + s.Title.Render(theme.Indent+theme.Copy.Title)
	if title == "" {
		return head + "\n\n" + content
	}
	return head + "\n\n" + theme.Indent + title + "\n\n" + content
}

// Footer renders a dim footer line, typically built with Hints.
func Footer(theme Theme, hints string) string {
	s := theme.Styles()
	return "\n" + s.Footer.Render(theme.Indent+hints)
}

// Hint is a key and its label for a footer.
type Hint struct {
	Key   string
	Label string
}

// Hints joins key hints with a middle dot separator.
func Hints(theme Theme, actions ...Hint) string {
	parts := make([]string, len(actions))
	for i, a := range actions {
		parts[i] = a.Key + " " + a.Label
	}
	return strings.Join(parts, " \u00b7 ")
}

// ShortPath replaces the home directory prefix of path with "~".
func ShortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	trimmed := strings.TrimPrefix(path, home+string(filepath.Separator))
	if trimmed != path {
		return "~/" + trimmed
	}
	return path
}

// Wrap splits text into lines no longer than width runes, preserving
// existing newlines. A non-positive width means 76.
func Wrap(text string, width int) []string {
	if width <= 0 {
		width = 76
	}
	if text == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			lines = append(lines, "")
			continue
		}
		runes := []rune(line)
		for len(runes) > width {
			lines = append(lines, string(runes[:width]))
			runes = runes[width:]
		}
		if len(runes) > 0 {
			lines = append(lines, string(runes))
		}
	}
	return lines
}

// KeyValue renders an aligned "label value" line.
func KeyValue(label, value string) string {
	return fmt.Sprintf("  %-22s %s", label, value)
}

// IsInteractive reports whether both stdin and stdout are terminals.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
