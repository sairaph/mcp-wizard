package render

import (
	"fmt"
	"strings"
)

// Fence renders content inside a tilde-fenced block, growing the fence
// past any content tildes. language is emitted as the info-string.
func Fence(content, language string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	longest := 2
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		count := 0
		for _, ch := range trimmed {
			if ch == '~' {
				count++
			} else {
				break
			}
		}
		if count > longest {
			longest = count
		}
	}

	fence := strings.Repeat("~", longest+1)
	var out strings.Builder
	if language != "" {
		out.WriteString(fence + language + "\n")
	} else {
		out.WriteString(fence + "\n")
	}
	out.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		out.WriteString("\n")
	}
	out.WriteString(fence + "\n")
	return out.String()
}

// Screen renders terminal screen lines as plain text inside a fence.
func Screen(lines []string) string {
	if len(lines) == 0 {
		return Fence("(the screen is blank)", "text")
	}
	return Fence(strings.Join(lines, "\n"), "text")
}

// ProgressBar renders a text progress bar as a pure string (no lipgloss).
// Returns "[####----]" for example, or "" if total is 0.
func ProgressBar(done, total, width int) string {
	if total <= 0 || width <= 2 {
		return ""
	}
	if done > total {
		done = total
	}
	filled := done * (width - 2) / total
	if filled > width-2 {
		filled = width - 2
	}
	return fmt.Sprintf("[%s%s]", strings.Repeat("#", filled), strings.Repeat("-", width-2-filled))
}
