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
	return strings.ReplaceAll(path, home+string(filepath.Separator),
		"~"+string(filepath.Separator))
}

func Wrap(text string, width int) []string {
	if text == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		for len(line) > width {
			lines = append(lines, line[:width])
			line = line[width:]
		}
		lines = append(lines, line)
	}
	return lines
}

func KeyValue(label, value string) string {
	return fmt.Sprintf("  %-22s %s", label, value)
}

func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
