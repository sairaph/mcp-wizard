package menu_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/menu"
)

// compile-time interface check
var _ app.Screen = (*menu.Model)(nil)

func TestID(t *testing.T) {
	m := menu.New("test", nil)
	if m.ID() != "menu" {
		t.Fatalf("expected ID 'menu', got %q", m.ID())
	}
}

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

func TestFocusRebuildsItems(t *testing.T) {
	callCount := 0
	m := menu.New("test", func() []menu.Item {
		callCount++
		return []menu.Item{{Label: "x", Action: "x"}}
	})

	m.Focus()
	if callCount != 1 {
		t.Fatalf("expected buildFn called once, got %d", callCount)
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

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestEnterReturnsError(t *testing.T) {
	m := menu.New("test", func() []menu.Item {
		return []menu.Item{
			{Label: "a", Action: "alpha"},
			{Label: "b", Action: "beta"},
		}
	})
	m.Init()

	cmd, err := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "menu:select:alpha" {
		t.Fatalf("expected 'menu:select:alpha', got %q", err.Error())
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd, err = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err.Error() != "menu:select:beta" {
		t.Fatalf("expected 'menu:select:beta', got %q", err.Error())
	}
}

func TestQReturnsError(t *testing.T) {
	m := menu.New("test", func() []menu.Item {
		return []menu.Item{{Label: "a", Action: "a"}}
	})
	m.Init()

	cmd, err := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "menu:quit" {
		t.Fatalf("expected 'menu:quit', got %q", err.Error())
	}
}

func TestCtrlCReturnsError(t *testing.T) {
	m := menu.New("test", func() []menu.Item {
		return []menu.Item{{Label: "a", Action: "a"}}
	})
	m.Init()

	cmd, err := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "menu:quit" {
		t.Fatalf("expected 'menu:quit', got %q", err.Error())
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

	v := m.View(80, 24)
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
	if !contains(v, "↑↓") {
		t.Fatal("expected help text in view")
	}
}

func TestViewWithoutHelpWhenShort(t *testing.T) {
	m := menu.New("T", func() []menu.Item {
		return []menu.Item{{Label: "Only", Action: "o"}}
	})
	m.Init()

	v := m.View(80, 1)
	if contains(v, "↑↓") {
		t.Fatal("expected no help text when height is too small")
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
