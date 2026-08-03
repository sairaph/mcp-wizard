package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

func Section(theme Theme, title, content string) string {
	s := theme.Styles()
	head := "\n" + s.Title.Render(theme.Indent + theme.Copy.Title)
	if title == "" {
		return head + "\n\n" + content
	}
	return head + "\n\n" + theme.Indent + title + "\n\n" + content
}

func Footer(theme Theme, hints string) string {
	s := theme.Styles()
	return "\n" + s.Footer.Render(theme.Indent + hints)
}

type Hint struct {
	Key   string
	Label string
}

func Hints(theme Theme, actions ...Hint) string {
	parts := make([]string, len(actions))
	for i, a := range actions {
		parts[i] = a.Key + " " + a.Label
	}
	return strings.Join(parts, " \u00b7 ")
}

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

func Wrap(text string, width int) []string {
	if width <= 0 {
		width = 76
	}
	if text == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
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

func KeyValue(label, value string) string {
	return fmt.Sprintf("  %-22s %s", label, value)
}

func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
