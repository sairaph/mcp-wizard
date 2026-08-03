package confirm_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/confirm"
)

// compile-time interface check
var _ app.Screen = (*confirm.Model)(nil)

func TestID(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")
	if c.ID() != "confirm" {
		t.Fatalf("expected ID 'confirm', got %q", c.ID())
	}
}

func TestInit(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")
	cmd := c.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
}

func TestCursorMovement(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd, err := c.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err != nil {
		t.Fatal("expected no error")
	}

	cmd, err = c.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err != nil {
		t.Fatal("expected no error")
	}

	cmd, err = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err != nil {
		t.Fatal("expected no error")
	}

	cmd, err = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err != nil {
		t.Fatal("expected no error")
	}
}

func TestEnterConfirmReturnsError(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd, err := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "confirm:confirmed" {
		t.Fatalf("expected 'confirm:confirmed', got %q", err.Error())
	}
}

func TestEnterCancelReturnsError(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	_, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})

	cmd, err := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "confirm:cancelled" {
		t.Fatalf("expected 'confirm:cancelled', got %q", err.Error())
	}
}

func TestEscReturnsError(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd, err := c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "confirm:cancelled" {
		t.Fatalf("expected 'confirm:cancelled', got %q", err.Error())
	}
}

func TestQReturnsError(t *testing.T) {
	c := confirm.New("title", "detail", "Yes", "No")

	cmd, err := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "confirm:cancelled" {
		t.Fatalf("expected 'confirm:cancelled', got %q", err.Error())
	}
}

func TestViewRenders(t *testing.T) {
	c := confirm.New("Confirm Title", "Are you sure?", "Yes", "No")

	v := c.View(80, 24)
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
	if !contains(v, "↑↓") {
		t.Fatal("expected help text in view")
	}
}

func TestViewWithoutDetail(t *testing.T) {
	c := confirm.New("Confirm Title", "", "Y", "N")

	v := c.View(80, 24)
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
