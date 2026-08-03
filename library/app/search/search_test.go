package search_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/search"
)

func TestNew(t *testing.T) {
	m := search.New("Find")
	if m.Title != "Find" {
		t.Errorf("expected title 'Find', got %q", m.Title)
	}
	if m.Input.Focused() != true {
		t.Error("expected input focused")
	}
	if m.Width != 80 {
		t.Errorf("expected default width 80, got %d", m.Width)
	}
	if m.Height != 24 {
		t.Errorf("expected default height 24, got %d", m.Height)
	}
}

func TestSetResults(t *testing.T) {
	m := search.New("Test")
	m.Searching = true

	results := []search.Result{
		{Label: "foo", Detail: "first"},
		{Label: "bar", Detail: "second"},
	}
	m.SetResults(results)

	if m.Searching {
		t.Error("expected Searching to be false after SetResults")
	}
	if len(m.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(m.Results))
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", m.Cursor)
	}
}

func TestCursorMovement(t *testing.T) {
	m := search.New("Test")
	m.SetResults([]search.Result{
		{Label: "a"},
		{Label: "b"},
		{Label: "c"},
	})

	tests := []struct {
		key  string
		want int
	}{
		{"down", 1},
		{"down", 2},
		{"up", 1},
		{"up", 0},
	}

	for _, tt := range tests {
		cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
		if cmd != nil {
			t.Fatalf("unexpected cmd on %q: %v", tt.key, cmd)
		}
		if m.Cursor != tt.want {
			t.Errorf("key %q: expected cursor %d, got %d", tt.key, tt.want, m.Cursor)
		}
	}
}

func TestCursorBounds(t *testing.T) {
	m := search.New("Test")
	m.SetResults([]search.Result{{Label: "only"}})

	// should not go below 0
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor 0 at top bound, got %d", m.Cursor)
	}

	// should not go above len-1
	cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor 0 at bottom bound, got %d", m.Cursor)
	}
}

func TestEnterTriggersSearch(t *testing.T) {
	m := search.New("Test")
	m.Input.SetValue("hello")

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd on enter")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "search" {
		t.Fatalf("expected source 'search', got %q", am.Source)
	}
	if am.Value != "query" {
		t.Fatalf("expected value 'query', got %q", am.Value)
	}
	if am.Data != "hello" {
		t.Fatalf("expected data 'hello', got %v", am.Data)
	}
	if !m.Searching {
		t.Error("expected Searching to be true after enter")
	}
	if m.Query != "hello" {
		t.Errorf("expected Query 'hello', got %q", m.Query)
	}
}

func TestEnterWhileSearchingDoesNothing(t *testing.T) {
	m := search.New("Test")
	m.Searching = true

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("expected nil cmd when already searching, got %v", cmd)
	}
}

func TestEscCancels(t *testing.T) {
	m := search.New("Test")

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cmd on cancel")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Value != "cancelled" {
		t.Fatalf("expected value 'cancelled', got %q", am.Value)
	}
}

func TestWindowSize(t *testing.T) {
	m := search.New("Test")
	cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Width != 100 {
		t.Errorf("expected Width 100, got %d", m.Width)
	}
	if m.Height != 40 {
		t.Errorf("expected Height 40, got %d", m.Height)
	}
}
