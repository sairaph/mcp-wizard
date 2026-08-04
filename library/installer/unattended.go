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

// PrintResults prints a summary of apply results.
func PrintResults(w io.Writer, results []harness.Result, enabling, dryRun bool) {
	for _, r := range results {
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
		fmt.Fprintf(w, "  %-22s %s\n", r.Name, status)
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
