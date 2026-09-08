package menu_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/menu"
)

func TestInit(t *testing.T) {
	items := []menu.Item{{Label: "a", Action: "a"}, {Label: "b", Action: "b"}}
	built := false
	m := menu.New("test", func() []menu.Item {
		built = true
		return items
	})

	cmd := m.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if !built {
		t.Fatal("expected buildFn to be called")
	}
}

func TestCursorMovement(t *testing.T) {
	m := menu.New("test", func() []menu.Item {
		return []menu.Item{
			{Label: "a", Action: "a"},
			{Label: "b", Action: "b"},
			{Label: "c", Action: "c"},
		}
	})
	m.Init()
	if m.Cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", m.Cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Cursor != 0 {
		t.Fatalf("expected cursor 0 after up, got %d", m.Cursor)
	}
}

func TestEnterReturnsAction(t *testing.T) {
	m := menu.New("test", func() []menu.Item {
		return []menu.Item{
			{Label: "a", Action: "alpha"},
			{Label: "b", Action: "beta"},
		}
	})
	m.Init()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "menu" {
		t.Fatalf("expected source 'menu', got %q", am.Source)
	}
	if am.Value != "select" {
		t.Fatalf("expected value 'select', got %q", am.Value)
	}
	if am.Data != "alpha" {
		t.Fatalf("expected data 'alpha', got %v", am.Data)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	am = msg.(app.ActionMsg)
	if am.Data != "beta" {
		t.Fatalf("expected data 'beta', got %v", am.Data)
	}
}

func TestViewRenders(t *testing.T) {
	m := menu.New("Test Menu", func() []menu.Item {
		return []menu.Item{
			{Label: "Option 1", Action: "opt1"},
			{Label: "Option 2", Action: "opt2"},
		}
	})
	m.Init()

	v := m.View()
	if v == "" {
		t.Fatal("expected non-empty view")
	}
	if !contains(v, "Test Menu") {
		t.Fatal("expected title in view")
	}
	if !contains(v, "Option 1") {
		t.Fatal("expected Option 1 in view")
	}
	if !contains(v, "Option 2") {
		t.Fatal("expected Option 2 in view")
	}
	if !contains(v, "\u2191\u2193") {
		t.Fatal("expected help text in view")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
