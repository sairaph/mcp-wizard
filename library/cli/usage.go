package cli

import (
	"fmt"
	"io"
	"strings"
)

// Spec describes one subcommand for the Usage output.
type Spec struct {
	Name        string
	Description string
}

// Usage writes help text listing the available subcommands.
func Usage(w io.Writer, specs []Spec) {
	if w == nil {
		return
	}
	maxLen := 0
	for _, spec := range specs {
		if l := len(spec.Name); l > maxLen {
			maxLen = l
		}
	}

	fmt.Fprintf(w, "Usage: <binary> <command> [flags]\n\nCommands:\n")
	for _, spec := range specs {
		padding := strings.Repeat(" ", maxLen-len(spec.Name)+2)
		fmt.Fprintf(w, "  %s%s%s\n", spec.Name, padding, spec.Description)
	}
	fmt.Fprintf(w, "\nFlags:\n")
	fmt.Fprintf(w, "  --help, -h     Show this help\n")
	fmt.Fprintf(w, "  --version, -v  Show version\n")
	fmt.Fprintf(w, "\nInstall/configure flags:\n")
	fmt.Fprintf(w, "  --all          Register with all known clients\n")
	fmt.Fprintf(w, "  --yes          Non-interactive mode\n")
	fmt.Fprintf(w, "  --dry-run      Show what would change without writing\n")
	fmt.Fprintf(w, "  --name         Server name in client configs\n")
	fmt.Fprintf(w, "  --email        Email for authentication\n")
	fmt.Fprintf(w, "  --token        API token for authentication\n")
	fmt.Fprintf(w, "  --clients      Comma-separated client IDs\n")
}
