package form_test

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/form"
)

func TestNew(t *testing.T) {
	fields := []form.Field{
		{Label: "Username"},
		{Label: "Password", Secret: true},
		{Label: "Email", Value: "a@b.com"},
	}
	m := form.New("Login", fields)
	if m.Title != "Login" {
		t.Errorf("expected title 'Login', got %q", m.Title)
	}
	if len(m.Inputs) != 3 {
		t.Fatalf("expected 3 inputs, got %d", len(m.Inputs))
	}
	if !m.Inputs[0].Focused() {
		t.Error("expected first input focused")
	}
	if m.Inputs[1].EchoMode != textinput.EchoPassword {
		t.Error("expected secret field to use password echo mode")
	}
	if m.Inputs[2].Value() != "a@b.com" {
		t.Errorf("expected pre-filled value 'a@b.com', got %q", m.Inputs[2].Value())
	}
}

func TestTabCyclesFocus(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	})

	if m.Focused != 0 {
		t.Fatalf("expected focused 0, got %d", m.Focused)
	}

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Focused != 1 {
		t.Errorf("expected focused 1 after tab, got %d", m.Focused)
	}

	cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Focused != 2 {
		t.Errorf("expected focused 2 after second tab, got %d", m.Focused)
	}

	// wrap around
	cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Focused != 0 {
		t.Errorf("expected focused 0 after wrap, got %d", m.Focused)
	}
}

func TestShiftTabCyclesBackwards(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	})
	m.Focused = 2
	m.Inputs[2].Focus()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("shift+tab")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Focused != 1 {
		t.Errorf("expected focused 1, got %d", m.Focused)
	}
}

func TestUpCyclesBackwards(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	})
	m.Focused = 2
	m.Inputs[2].Focus()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Focused != 1 {
		t.Errorf("expected focused 1, got %d", m.Focused)
	}
}

func TestDownCyclesForward(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})
	if cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if m.Focused != 1 {
		t.Errorf("expected focused 1, got %d", m.Focused)
	}
}

func TestEnterWithValidationError(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "Name", Validate: func(v string) error {
			if v == "" {
				return errors.New("name is required")
			}
			return nil
		}},
		{Label: "Age"},
	})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd (validation should prevent submit), got %v", cmd)
	}
	if m.Submitted {
		t.Fatal("expected form not submitted after validation failure")
	}
	if m.Error != "name is required" {
		t.Errorf("expected error 'name is required', got %q", m.Error)
	}
	if m.Focused != 0 {
		t.Errorf("expected focus on first field after validation error, got %d", m.Focused)
	}
}

func TestEnterWithoutValidationSubmits(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "Name"},
		{Label: "Age"},
	})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd on submit")
	}
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "form" {
		t.Fatalf("expected source 'form', got %q", am.Source)
	}
	if am.Value != "submitted" {
		t.Fatalf("expected value 'submitted', got %q", am.Value)
	}
	if !m.Submitted {
		t.Fatal("expected form to be submitted")
	}
}

func TestEscCancels(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "Name"},
	})

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

func TestValues(t *testing.T) {
	m := form.New("Test", []form.Field{
		{Label: "Name", Value: "Alice"},
		{Label: "Role", Value: "admin"},
	})
	m.Inputs[0].SetValue("Bob")

	vals := m.Values()
	if len(vals) != 2 {
		t.Fatalf("expected 2 values, got %d", len(vals))
	}
	if vals["Name"] != "Bob" {
		t.Errorf("expected 'Bob', got %q", vals["Name"])
	}
	if vals["Role"] != "admin" {
		t.Errorf("expected 'admin', got %q", vals["Role"])
	}
}

func TestSubmittedIgnoresInput(t *testing.T) {
	m := form.New("Test", []form.Field{{Label: "A"}})
	m.Submitted = true

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("expected nil cmd after submit, got %v", cmd)
	}
}
