package tui

import (
	"fmt"

	"github.com/sairaph/mcp-wizard/render"
)

func StepIndicator(current, total int, names []string) string {
	if total <= 0 || len(names) == 0 || current >= total || current < 0 {
		return ""
	}
	name := ""
	if current >= 0 && current < len(names) {
		name = names[current]
	}
	return fmt.Sprintf("Step %d of %d \u2014 %s", current+1, total, name)
}

func ProgressBar(done, total, width int) string {
	bar := render.ProgressBar(done, total, width)
	if bar == "" {
		return ""
	}
	return "  " + bar
}
