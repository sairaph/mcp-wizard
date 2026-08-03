package confirm_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/confirm"
)

func TestInit(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")
	cmd := c.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
}

func TestCursorMovement(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd := c.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}

	cmd = c.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}

	cmd = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}

	cmd = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
}

func TestEnterConfirmReturnsAction(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "confirm" {
		t.Fatalf("expected source 'confirm', got %q", am.Source)
	}
	if am.Value != "confirmed" {
		t.Fatalf("expected value 'confirmed', got %q", am.Value)
	}
}

func TestEnterCancelReturnsAction(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	c.Update(tea.KeyMsg{Type: tea.KeyDown})

	cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "confirm" {
		t.Fatalf("expected source 'confirm', got %q", am.Source)
	}
	if am.Value != "cancelled" {
		t.Fatalf("expected value 'cancelled', got %q", am.Value)
	}
}

func TestEscReturnsAction(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd := c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cmd")
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

func TestQReturnsAction(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected cmd")
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

func TestViewRenders(t *testing.T) {
	c := confirm.New("Confirm Title", "Are you sure?", "Yes", "No")

	v := c.View()
	if v == "" {
		t.Fatal("expected non-empty view")
	}
	if !contains(v, "Confirm Title") {
		t.Fatal("expected title in view")
	}
	if !contains(v, "Are you sure?") {
		t.Fatal("expected detail in view")
	}
	if !contains(v, "Yes") {
		t.Fatal("expected Yes in view")
	}
	if !contains(v, "No") {
		t.Fatal("expected No in view")
	}
	if !contains(v, "\u2191\u2193") {
		t.Fatal("expected help text in view")
	}
}

func TestViewWithoutDetail(t *testing.T) {
	c := confirm.New("Confirm Title", "", "Y", "N")

	v := c.View()
	if !contains(v, "Y") {
		t.Fatal("expected Y in view")
	}
	if !contains(v, "N") {
		t.Fatal("expected N in view")
	}
}

func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
