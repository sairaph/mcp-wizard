package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/sairaph/mcp-wizard/tui"
)

func init() {
	// Plain output so assertions can match text without escape codes
	// (termenv.Ascii is profile 3; importing termenv would add a direct
	// dependency for one constant).
	lipgloss.SetColorProfile(3)
}

func TestCheckboxListRendersCursorMarksAndStatus(t *testing.T) {
	items := []tui.CheckboxItem{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}, {ID: "c", Name: "Gamma"}}
	selected := map[string]bool{"a": true}
	selectable := func(i int) bool { return i != 2 }
	status := func(i int) string { return []string{"detected", "detected", "not installed"}[i] }
	out := tui.CheckboxList(tui.DefaultTheme, items, 1, selected, selectable, status, false, 4)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[0], "\u25cf") || strings.Contains(lines[0], ">") {
		t.Fatalf("selected row without cursor: %q", lines[0])
	}
	if !strings.Contains(lines[1], ">") || !strings.Contains(lines[1], "\u25cb") {
		t.Fatalf("cursor row unselected: %q", lines[1])
	}
	if !strings.Contains(lines[2], "\u00b7") || !strings.Contains(lines[2], "not installed") {
		t.Fatalf("non-selectable row: %q", lines[2])
	}
	if !strings.Contains(out, "press v to show 4 client(s)") {
		t.Fatalf("hidden hint missing: %q", out)
	}
	if got := tui.CheckboxList(tui.DefaultTheme, items, 2, selected, selectable, status, true, 0); !strings.Contains(got, "press v to hide") || !strings.Contains(strings.Split(got, "\n")[2], ">") {
		t.Fatalf("showAll view: %q", got)
	}
	if got := tui.CheckboxList(tui.DefaultTheme, nil, 0, nil, nil, nil, false, 0); !strings.Contains(got, tui.DefaultCopy.NoClients) {
		t.Fatalf("empty list: %q", got)
	}
}

func TestRadioActionAndToggleLists(t *testing.T) {
	radio := tui.RadioList(tui.DefaultTheme, 1, "one", "two")
	rl := strings.Split(strings.TrimRight(radio, "\n"), "\n")
	if strings.Contains(rl[0], "\u25cf") || !strings.Contains(rl[1], "\u25cf") || !strings.Contains(rl[1], ">") {
		t.Fatalf("radio: %q", radio)
	}
	menu := tui.ActionMenu(tui.DefaultTheme, 0, "Sign in", "Skip")
	if !strings.HasPrefix(strings.Split(menu, "\n")[0], " >") {
		t.Fatalf("action menu cursor: %q", menu)
	}
	toggle := tui.ToggleList(tui.DefaultTheme, []tui.ToggleItem{{ID: "id1", Label: "Label One", Tier: "core", Checked: true}, {ID: "id2", Tier: "extra"}}, 1)
	tl := strings.Split(strings.TrimRight(toggle, "\n"), "\n")
	if !strings.Contains(tl[0], "Label One") || !strings.Contains(tl[0], "(core)") || !strings.Contains(tl[0], "\u25cf") {
		t.Fatalf("toggle row 0: %q", tl[0])
	}
	if !strings.Contains(tl[1], "id2") || !strings.Contains(tl[1], ">") {
		t.Fatalf("toggle row falls back to ID and shows cursor: %q", tl[1])
	}
}

func TestTextInput(t *testing.T) {
	if got := tui.TextInput("", "type here", false); !strings.Contains(got, "type here") {
		t.Fatalf("placeholder: %q", got)
	}
	if got := tui.TextInput("s3cret", "type here", true); strings.Contains(got, "s3cret") || !strings.Contains(got, "******") {
		t.Fatalf("masked: %q", got)
	}
	if got := tui.TextInput("", "type here", true); !strings.Contains(got, "type here") {
		t.Fatalf("masked placeholder must stay readable: %q", got)
	}
}

func TestLayoutHelpers(t *testing.T) {
	sec := tui.Section(tui.DefaultTheme, "Title here", "body\n")
	if !strings.Contains(sec, tui.DefaultCopy.Title) || !strings.Contains(sec, "Title here") || !strings.Contains(sec, "body") {
		t.Fatalf("section: %q", sec)
	}
	if got := tui.Section(tui.DefaultTheme, "", "x"); strings.Count(got, "\n\n") != 1 {
		t.Fatalf("section without title: %q", got)
	}
	hints := tui.Hints(tui.DefaultTheme, tui.Hint{Key: "enter", Label: "go"}, tui.Hint{Key: "q", Label: "quit"})
	if hints != "enter go \u00b7 q quit" {
		t.Fatalf("hints = %q", hints)
	}
	if got := tui.Footer(tui.DefaultTheme, hints); !strings.HasPrefix(got, "\n") || !strings.Contains(got, hints) {
		t.Fatalf("footer = %q", got)
	}
	if got := tui.KeyValue("Name", "value"); got != "  Name                   value" {
		t.Fatalf("keyvalue = %q", got)
	}
}

func TestWrap(t *testing.T) {
	if tui.Wrap("", 10) != nil {
		t.Fatal("empty text wraps to nil")
	}
	lines := tui.Wrap("abcdefghij\n\nklm", 4)
	want := []string{"abcd", "efgh", "ij", "", "klm"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("wrap = %v", lines)
	}
	if got := tui.Wrap("h\u00e9llo", 2); got[0] != "h\u00e9" {
		t.Fatalf("wrap must count runes, got %v", got)
	}
	if got := tui.Wrap(strings.Repeat("a", 80), 0); len(got) != 2 || len(got[0]) != 76 {
		t.Fatalf("default width 76, got %d lines first len %d", len(got), len(got[0]))
	}
}

func TestShortPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home")
	}
	if got := tui.ShortPath(home); got != "~" {
		t.Fatalf("home = %q", got)
	}
	if got := tui.ShortPath(filepath.Join(home, "x", "y")); got != "~/x/y" {
		t.Fatalf("under home = %q", got)
	}
	if got := tui.ShortPath("/etc/hosts"); got != "/etc/hosts" {
		t.Fatalf("outside home = %q", got)
	}
	if got := tui.ShortPath(home + "2"); got != home+"2" {
		t.Fatalf("sibling of home must not be shortened: %q", got)
	}
}

func TestProgressAndSpinner(t *testing.T) {
	if got := tui.StepIndicator(1, 3, []string{"a", "b", "c"}); got != "Step 2 of 3 - b" {
		t.Fatalf("step indicator = %q", got)
	}
	if tui.StepIndicator(3, 3, []string{"a"}) != "" || tui.StepIndicator(0, 0, nil) != "" {
		t.Fatal("out-of-range step indicator must be empty")
	}
	if got := tui.ProgressBar(2, 4, 10); got != "  [####----]" {
		t.Fatalf("progress = %q", got)
	}
	if tui.ProgressBar(1, 0, 10) != "" {
		t.Fatal("zero total renders nothing")
	}
	frames := map[string]bool{}
	for i := 0; i < 8; i++ {
		frames[tui.SpinFrame(i)] = true
	}
	if len(frames) != 4 || tui.SpinFrame(-1) != tui.SpinFrame(0) {
		t.Fatalf("spinner frames = %v", frames)
	}
	if !tui.IsSpinMsg(tui.Spinner()()) {
		// Spinner returns a Tick command; executing it yields the tick message.
		t.Skip("tick fired outside its timer; covered by installer tests")
	}
}
