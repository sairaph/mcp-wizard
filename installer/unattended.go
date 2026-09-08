package installer

import (
	"context"
	"fmt"
	"io"

	"github.com/sairaph/mcp-wizard/harness"
)

// UnattendedRunner performs the install/uninstall without interaction.
type UnattendedRunner interface {
	Run(ctx context.Context, out, errw io.Writer) int
}

// formatResultStatus renders the per-result status word used by the PrintResults
// family. It has no trailing newline.
func formatResultStatus(r harness.Result, enabling, dryRun bool) string {
	verb := "registered"
	if !enabling {
		verb = "removed"
	}
	status := verb
	switch r.State {
	case harness.Applied:
		status = verb
	case harness.ApplyNoop:
		status = "already " + verb
	case harness.ApplyConflict:
		status = "conflict"
	case harness.ApplySkipped:
		status = "skipped: " + r.Reason
	case harness.ApplyFailed:
		status = "failed: " + r.Reason
	}
	if dryRun && (r.State == harness.Applied || r.State == harness.ApplyNoop) {
		if enabling {
			status = "would register"
		} else {
			status = "would remove"
		}
	}
	return status
}

// PrintResults prints a summary of apply results.
func PrintResults(w io.Writer, results []harness.Result, enabling, dryRun bool) {
	for _, r := range results {
		fmt.Fprintf(w, "  %-22s %s\n", r.Name, formatResultStatus(r, enabling, dryRun))
	}
}

// PrintResultsWithScope prints a summary of apply results with scope context.
// For project scope it reports the project directory and appends the resolved
// configuration file path to each result line. For global scope it matches
// PrintResults output exactly.
func PrintResultsWithScope(w io.Writer, results []harness.Result, scope harness.Scope, enabling, dryRun bool) {
	project := scope.IsProject()
	if project && scope.Dir != "" {
		fmt.Fprintf(w, "  Project scope: %s\n", scope.Dir)
	}
	for _, r := range results {
		line := fmt.Sprintf("  %-22s %s", r.Name, formatResultStatus(r, enabling, dryRun))
		if project && r.Path != "" {
			line += "  " + r.Path
		}
		fmt.Fprintln(w, line)
	}
}

// PrintChanges prints a dry-run plan: one line per change saying what would
// happen. Changes the plan could not evaluate show their state and reason.
func PrintChanges(w io.Writer, changes []harness.Change, scope harness.Scope) {
	if scope.IsProject() && scope.Dir != "" {
		fmt.Fprintf(w, "  Project scope: %s\n", scope.Dir)
	}
	if len(changes) == 0 {
		fmt.Fprintln(w, "  Nothing to change.")
		return
	}
	for _, c := range changes {
		name := c.Name
		if name == "" {
			name = string(c.HarnessID)
		}
		var what string
		switch {
		case c.State == harness.ApplyNoop:
			what = "already registered"
		case c.Action != "":
			what = "would " + c.Action
		case c.Reason != "":
			what = string(c.State) + ": " + c.Reason
		default:
			what = string(c.State)
		}
		line := fmt.Sprintf("  %-22s %s", name, what)
		if c.Path != "" {
			line += "  " + c.Path
		}
		fmt.Fprintln(w, line)
	}
}

// PrintReloadHints prints instructions for restarting affected clients.
func PrintReloadHints(w io.Writer, results []harness.Result, byID map[harness.ID]harness.Harness) {
	anyHints := false
	for _, r := range results {
		if r.State == harness.Applied {
			anyHints = true
			break
		}
	}
	if !anyHints {
		return
	}

	fmt.Fprintf(w, "\n  Restart the affected clients so they pick up the change:\n")
	for _, r := range results {
		if r.State != harness.Applied {
			continue
		}
		h, ok := byID[r.HarnessID]
		if !ok {
			continue
		}
		if h.ReloadHint != "" {
			fmt.Fprintf(w, "    %-22s %s\n", h.Name, h.ReloadHint)
		}
	}
}

// PrintNoClients prints a message when no clients were detected.
func PrintNoClients(w io.Writer, name string, remove bool) {
	past := "configured"
	verb := "configure"
	if remove {
		past = "uninstalled"
		verb = "uninstall"
	}
	fmt.Fprintf(w, "  No AI clients were detected, so nothing was %s.\n", past)
	fmt.Fprintf(w, "  Install an AI client and re-run `%s %s`.\n", name, verb)
}

// PrintPathHint prints a PATH warning if the binary is not on PATH.
func PrintPathHint(w io.Writer, installDir string) {
	if installDir == "" {
		return
	}
	fmt.Fprintf(w, "\n  Add this to your shell profile:\n")
	fmt.Fprintf(w, "    export PATH=\"%s:$PATH\"\n", installDir)
	fmt.Fprintf(w, "  Or open a new terminal so the command is on your PATH.\n")
}
