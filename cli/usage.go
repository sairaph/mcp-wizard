package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Spec describes one subcommand for the Usage output.
type Spec struct {
	Name        string
	Description string
}

// Usage writes help text listing the available subcommands. The binary name
// is taken from os.Args[0].
func Usage(w io.Writer, specs []Spec) {
	if w == nil {
		return
	}
	binary := "<binary>"
	if len(os.Args) > 0 && os.Args[0] != "" {
		binary = filepath.Base(os.Args[0])
	}
	maxLen := 0
	for _, spec := range specs {
		if l := len(spec.Name); l > maxLen {
			maxLen = l
		}
	}

	fmt.Fprintf(w, "Usage: %s <command> [flags]\n\nCommands:\n", binary)
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
	fmt.Fprintf(w, "  --scope        Configuration scope: project (default: global)\n")
	fmt.Fprintf(w, "  --dir          Project directory for --scope project and add\n")
	fmt.Fprintf(w, "\nServer flags:\n")
	fmt.Fprintf(w, "  --remote       Bridge stdio to a remote Streamable HTTP endpoint\n")
	fmt.Fprintf(w, "\nUpdate flags:\n")
	fmt.Fprintf(w, "  --from         Swap in a pre-downloaded binary instead of downloading\n")
}
