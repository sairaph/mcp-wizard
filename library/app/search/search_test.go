package search_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
		{"j", 2},
		{"up", 1},
		{"k", 0},
	}

	for _, tt := range tests {
		_, err := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
		if err != nil {
			t.Fatalf("unexpected error on %q: %v", tt.key, err)
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
	_, err := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor 0 at top bound, got %d", m.Cursor)
	}

	// should not go above len-1
	_, err = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor 0 at bottom bound, got %d", m.Cursor)
	}
}

func TestEnterTriggersSearch(t *testing.T) {
	m := search.New("Test")
	m.Input.SetValue("hello")

	_, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err == nil {
		t.Fatal("expected search error on enter")
	}
	if err.Error() != "search:query:hello" {
		t.Errorf("expected 'search:query:hello', got %v", err)
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

	cmd, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err != nil {
		t.Errorf("expected no error when already searching, got %v", err)
	}
	if cmd != nil {
		t.Error("expected nil cmd when already searching")
	}
}

func TestEscCancels(t *testing.T) {
	m := search.New("Test")

	_, err := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if err.Error() != "search:cancelled" {
		t.Errorf("expected 'search:cancelled', got %v", err)
	}
}

func TestWindowSize(t *testing.T) {
	m := search.New("Test")
	_, err := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Width != 100 {
		t.Errorf("expected Width 100, got %d", m.Width)
	}
	if m.Height != 40 {
		t.Errorf("expected Height 40, got %d", m.Height)
	}
}
