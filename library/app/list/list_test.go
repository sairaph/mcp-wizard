package list_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app/list"
)

func TestNew(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	if m.Title != "test" {
		t.Errorf("expected title 'test', got %q", m.Title)
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.Cursor)
	}
	if m.Page != 0 {
		t.Errorf("expected page 0, got %d", m.Page)
	}
	if m.Total != 10 {
		t.Errorf("expected total 10, got %d", m.Total)
	}
	if m.PerPage != 5 {
		t.Errorf("expected perPage 5, got %d", m.PerPage)
	}
}

func TestSetItems(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	items := []list.Item{
		{Label: "one", Active: true},
		{Label: "two"},
		{Label: "three", Active: true},
	}
	m.SetItems(items)
	if len(m.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(m.Items))
	}
	if m.Cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", m.Cursor)
	}
}

func TestSetItemsClampsCursor(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	m.Cursor = 5
	m.SetItems([]list.Item{{Label: "only"}})
	if m.Cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.Cursor)
	}
}

func TestSelected(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	if got := m.Selected(); got != -1 {
		t.Errorf("expected -1 for empty, got %d", got)
	}

	m.SetItems([]list.Item{{Label: "a"}, {Label: "b"}})
	if got := m.Selected(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}

	m.Cursor = 1
	if got := m.Selected(); got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestCursorMovement(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	m.SetItems([]list.Item{
		{Label: "a"},
		{Label: "b"},
		{Label: "c"},
	})

	tests := []struct {
		key    string
		want   int
	}{
		{"down", 1},
		{"j", 2},
		{"up", 1},
		{"k", 0},
		{"home", 0},
		{"g", 0},
		{"end", 2},
		{"G", 2},
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
	m := list.New("test", nil, 0, 10, 5)
	m.SetItems([]list.Item{{Label: "a"}})

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

func TestEnterReturnsSelectError(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	m.SetItems([]list.Item{{Label: "a"}, {Label: "b"}})

	_, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err == nil {
		t.Fatal("expected select error on enter")
	}
	if err.Error() != "list:select:0" {
		t.Errorf("unexpected error: %v", err)
	}

	m.Cursor = 1
	_, err = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err == nil {
		t.Fatal("expected select error on enter")
	}
	if err.Error() != "list:select:1" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnterOnEmptyNoError(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	_, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err != nil {
		t.Errorf("expected no error on enter with empty list, got %v", err)
	}
}

func TestViewLoading(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	m.Loading = true
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view when loading")
	}
}

func TestViewEmpty(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view when empty")
	}
}

func TestViewWithItems(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
	m.SetItems([]list.Item{
		{Label: "alpha", Active: true},
		{Label: "beta"},
	})
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view with items")
	}
}

func TestWindowSize(t *testing.T) {
	m := list.New("test", nil, 0, 10, 5)
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
