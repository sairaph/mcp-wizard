package detail_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app/detail"
)

func TestNew(t *testing.T) {
	m := detail.New("test title", "hello world")
	if m.Title != "test title" {
		t.Errorf("expected title 'test title', got %q", m.Title)
	}
	if m.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", m.Content)
	}
	if m.Width != 80 {
		t.Errorf("expected default Width 80, got %d", m.Width)
	}
	if m.Height != 24 {
		t.Errorf("expected default Height 24, got %d", m.Height)
	}
}

func TestSetContent(t *testing.T) {
	m := detail.New("t", "initial")
	m.SetContent("updated")
	if m.Content != "updated" {
		t.Errorf("expected content 'updated', got %q", m.Content)
	}
}

func TestUpdateWindowSize(t *testing.T) {
	m := detail.New("t", "content")
	cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Width != 120 {
		t.Errorf("expected Width 120, got %d", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("expected Height 50, got %d", m.Height)
	}
}

func TestViewWithTitle(t *testing.T) {
	m := detail.New("my title", "some content\nmore content")
	v := m.View()
	if v == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestViewWithoutTitle(t *testing.T) {
	m := detail.New("", "content only")
	v := m.View()
	if v == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestUpdatePassesThrough(t *testing.T) {
	m := detail.New("t", "c")
	// send a key msg that doesn't match any case — should still work
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		t.Fatalf("unexpected cmd on passthrough: %v", cmd)
	}
}
