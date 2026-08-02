package installer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sairaph/mcp-wizard/harness"
	"github.com/sairaph/mcp-wizard/installer"
)

func TestPrintResults(t *testing.T) {
	tests := []struct {
		name     string
		results  []harness.Result
		enabling bool
		dryRun   bool
		want     []string
		notWant  []string
	}{
		{
			name: "applied",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.Applied},
			},
			enabling: true,
			want:     []string{"Claude Code", "registered"},
		},
		{
			name: "noop",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.ApplyNoop},
			},
			enabling: true,
			want:     []string{"Claude Code", "already registered"},
		},
		{
			name: "conflict",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.ApplyConflict},
			},
			enabling: true,
			want:     []string{"Claude Code", "conflict"},
		},
		{
			name: "skipped",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.ApplySkipped, Reason: "not needed"},
			},
			enabling: true,
			want:     []string{"Claude Code", "skipped: not needed"},
		},
		{
			name: "failed",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.ApplyFailed, Reason: "permission denied"},
			},
			enabling: true,
			want:     []string{"Claude Code", "failed: permission denied"},
		},
		{
			name: "dry run",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.Applied},
			},
			enabling: true,
			dryRun:   true,
			want:     []string{"Claude Code", "would register"},
			notWant:  []string{"applied"},
		},
		{
			name: "uninstall",
			results: []harness.Result{
				{Name: "Claude Code", State: harness.Applied},
			},
			enabling: false,
			want:     []string{"Claude Code", "removed"},
			notWant:  []string{"registered"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			installer.PrintResults(&buf, tt.results, tt.enabling, tt.dryRun)
			out := buf.String()
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, out)
				}
			}
			for _, s := range tt.notWant {
				if strings.Contains(out, s) {
					t.Errorf("expected output NOT to contain %q, got:\n%s", s, out)
				}
			}
		})
	}
}

func TestPrintReloadHints(t *testing.T) {
	t.Run("filters applied results", func(t *testing.T) {
		byID := map[harness.ID]harness.Harness{
			"claude": {ID: "claude", Name: "Claude Code", ReloadHint: "restart Claude"},
			"cursor": {ID: "cursor", Name: "Cursor", ReloadHint: ""},
		}
		results := []harness.Result{
			{HarnessID: "claude", Name: "Claude Code", State: harness.Applied},
			{HarnessID: "cursor", Name: "Cursor", State: harness.ApplyNoop},
		}

		var buf bytes.Buffer
		installer.PrintReloadHints(&buf, results, byID)
		out := buf.String()

		if !strings.Contains(out, "Claude Code") {
			t.Errorf("expected Claude Code hint, got:\n%s", out)
		}
		if !strings.Contains(out, "restart Claude") {
			t.Errorf("expected restart hint, got:\n%s", out)
		}
		if strings.Contains(out, "Cursor") {
			t.Errorf("did not expect Cursor hint, got:\n%s", out)
		}
	})

	t.Run("no output when no applied results", func(t *testing.T) {
		results := []harness.Result{
			{HarnessID: "claude", Name: "Claude Code", State: harness.ApplyNoop},
		}
		var buf bytes.Buffer
		installer.PrintReloadHints(&buf, results, nil)
		if buf.Len() > 0 {
			t.Errorf("expected no output, got:\n%s", buf.String())
		}
	})
}

func TestPrintNoClients(t *testing.T) {
	t.Run("configure", func(t *testing.T) {
		var buf bytes.Buffer
		installer.PrintNoClients(&buf, "myapp", false)
		out := buf.String()
		if !strings.Contains(out, "No AI clients") {
			t.Errorf("expected No AI clients message, got:\n%s", out)
		}
		if !strings.Contains(out, "myapp configure") {
			t.Errorf("expected re-run hint, got:\n%s", out)
		}
	})

	t.Run("uninstall", func(t *testing.T) {
		var buf bytes.Buffer
		installer.PrintNoClients(&buf, "myapp", true)
		out := buf.String()
		if !strings.Contains(out, "uninstalled") {
			t.Errorf("expected uninstalled, got:\n%s", out)
		}
		if !strings.Contains(out, "myapp uninstall") {
			t.Errorf("expected re-run hint, got:\n%s", out)
		}
	})
}

func TestPrintPathHint(t *testing.T) {
	t.Run("non-empty dir", func(t *testing.T) {
		var buf bytes.Buffer
		installer.PrintPathHint(&buf, "/usr/local/bin")
		out := buf.String()
		if !strings.Contains(out, `export PATH="/usr/local/bin:$PATH"`) {
			t.Errorf("expected PATH export, got:\n%s", out)
		}
	})

	t.Run("empty dir", func(t *testing.T) {
		var buf bytes.Buffer
		installer.PrintPathHint(&buf, "")
		if buf.Len() > 0 {
			t.Errorf("expected no output for empty dir, got:\n%s", buf.String())
		}
	})
}

func TestHarnessStepID(t *testing.T) {
	step := installer.HarnessStep[int](nil, func(*int) *installer.HarnessState { return nil }, installer.HarnessStepOptions{})
	if got := step.ID(); got != "harnesses" {
		t.Errorf("ID() = %q, want %q", got, "harnesses")
	}
}

func TestHarnessStepTitle(t *testing.T) {
	step := installer.HarnessStep[int](nil, func(*int) *installer.HarnessState { return nil }, installer.HarnessStepOptions{})
	title := step.Title(new(int))
	if !strings.Contains(title, "AI clients") {
		t.Errorf("Title should mention AI clients, got: %q", title)
	}
}

func TestLoginStepID(t *testing.T) {
	cfg := installer.LoginConfig{ID: "my-login", Label: "My Login"}
	step := installer.LoginStep[int](cfg, func(*int) *installer.LoginState { return nil })
	if got := step.ID(); got != "my-login" {
		t.Errorf("ID() = %q, want %q", got, "my-login")
	}
}

func TestLoginStepTitle(t *testing.T) {
	cfg := installer.LoginConfig{ID: "my-login", Label: "My Login"}
	step := installer.LoginStep[int](cfg, func(*int) *installer.LoginState { return nil })
	title := step.Title(new(int))
	if !strings.Contains(title, "My Login") {
		t.Errorf("Title should contain label, got: %q", title)
	}
}

func TestVisibleIndices(t *testing.T) {
	hs := []harness.Harness{
		{ID: "a", Name: "A", State: harness.Detected},
		{ID: "b", Name: "B", State: harness.NotDetected},
		{ID: "c", Name: "C", State: harness.Detected, Configured: true},
		{ID: "d", Name: "D", State: harness.Unavailable},
	}

	t.Run("showAll=false", func(t *testing.T) {
		indices := installer.VisibleIndices(hs, false)
		want := []int{0, 2}
		if !equalIntSlices(indices, want) {
			t.Errorf("VisibleIndices(false) = %v, want %v", indices, want)
		}
	})

	t.Run("showAll=true", func(t *testing.T) {
		indices := installer.VisibleIndices(hs, true)
		want := []int{0, 1, 2, 3}
		if !equalIntSlices(indices, want) {
			t.Errorf("VisibleIndices(true) = %v, want %v", indices, want)
		}
	})
}

func TestFirstSelectable(t *testing.T) {
	hs := []harness.Harness{
		{ID: "a", Name: "A", State: harness.Unavailable},
		{ID: "b", Name: "B", State: harness.Detected},
		{ID: "c", Name: "C", State: harness.Detected},
	}

	idx := installer.FirstSelectable(hs)
	if idx != 1 {
		t.Errorf("FirstSelectable() = %d, want %d", idx, 1)
	}
}

func TestMoveCursor(t *testing.T) {
	hs := []harness.Harness{
		{ID: "a", Name: "A", State: harness.Detected},
		{ID: "b", Name: "B", State: harness.NotDetected},
		{ID: "c", Name: "C", State: harness.Detected},
	}

	t.Run("move down", func(t *testing.T) {
		st := &installer.HarnessState{Detections: hs, Cursor: 0, ShowAll: false}
		installer.MoveCursor(st, hs, 1)
		if st.Cursor != 2 {
			t.Errorf("after move down, Cursor = %d, want %d", st.Cursor, 2)
		}
	})

	t.Run("wrap around", func(t *testing.T) {
		st := &installer.HarnessState{Detections: hs, Cursor: 2, ShowAll: false}
		installer.MoveCursor(st, hs, 1)
		if st.Cursor != 0 {
			t.Errorf("after wrap, Cursor = %d, want %d", st.Cursor, 0)
		}
	})

	t.Run("move up wraps", func(t *testing.T) {
		st := &installer.HarnessState{Detections: hs, Cursor: 0, ShowAll: false}
		installer.MoveCursor(st, hs, -1)
		if st.Cursor != 2 {
			t.Errorf("after wrap up, Cursor = %d, want %d", st.Cursor, 2)
		}
	})
}

func TestToggleHarness(t *testing.T) {
	hs := []harness.Harness{
		{ID: "a", Name: "A", State: harness.Detected},
		{ID: "b", Name: "B", State: harness.NotDetected},
	}

	st := &installer.HarnessState{
		Detections: hs,
		Selected:   map[harness.ID]bool{"a": false},
		Cursor:     0,
	}

	installer.ToggleHarness(st)
	if !st.Selected["a"] {
		t.Errorf("expected harness a to be selected after toggle")
	}

	installer.ToggleHarness(st)
	if st.Selected["a"] {
		t.Errorf("expected harness a to be unselected after second toggle")
	}
}

func TestToggleAll(t *testing.T) {
	hs := []harness.Harness{
		{ID: "a", Name: "A", State: harness.Detected},
		{ID: "b", Name: "B", State: harness.Detected},
		{ID: "c", Name: "C", State: harness.NotDetected},
	}

	st := &installer.HarnessState{
		Detections: hs,
		Selected:   map[harness.ID]bool{"a": false, "b": false, "c": false},
	}

	installer.ToggleAll(st)
	if !st.Selected["a"] || !st.Selected["b"] {
		t.Errorf("expected selectable harnesses to be selected after toggleAll")
	}
	if st.Selected["c"] {
		t.Errorf("expected non-selectable harness c to remain unselected")
	}

	installer.ToggleAll(st)
	if st.Selected["a"] || st.Selected["b"] {
		t.Errorf("expected selectable harnesses to be unselected after second toggleAll")
	}
}

func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
