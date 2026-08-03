package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
)

func TestActionReturnsActionMsg(t *testing.T) {
	cmd := app.Action("menu", "select", "boards")
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Source != "menu" {
		t.Errorf("expected Source=menu, got %q", am.Source)
	}
	if am.Value != "select" {
		t.Errorf("expected Value=select, got %q", am.Value)
	}
	if am.Data != "boards" {
		t.Errorf("expected Data=boards, got %v", am.Data)
	}
}

func TestActionWithData(t *testing.T) {
	cmd := app.Action("list", "select", 42)
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Data != 42 {
		t.Errorf("expected Data=42, got %v", am.Data)
	}
}

func TestActionWithoutData(t *testing.T) {
	cmd := app.Action("confirm", "confirmed")
	msg := cmd()
	am, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatalf("expected app.ActionMsg, got %T", msg)
	}
	if am.Data != nil {
		t.Errorf("expected Data=nil, got %v", am.Data)
	}
}

func TestActionWithExtraData(t *testing.T) {
	cmd := app.Action("table", "select", 7, "extra")
	msg := cmd()
	actionMsg, ok := msg.(app.ActionMsg)
	if !ok {
		t.Fatal("expected ActionMsg")
	}
	_ = actionMsg
	if actionMsg.Data != 7 {
		t.Errorf("expected Data=7, got %v", actionMsg.Data)
	}
}

func TestAppModelDefaults(t *testing.T) {
	m := app.AppModel{}
	if m.Step != app.StepMenu {
		t.Errorf("expected Step=%d (StepMenu), got %d", app.StepMenu, m.Step)
	}
	if m.Width != 0 {
		t.Errorf("expected Width=0, got %d", m.Width)
	}
	if m.Height != 0 {
		t.Errorf("expected Height=0, got %d", m.Height)
	}
	if m.Status != "" {
		t.Errorf("expected Status=\"\", got %q", m.Status)
	}
	if m.Failure != "" {
		t.Errorf("expected Failure=\"\", got %q", m.Failure)
	}
}

func TestHandleGlobalKeysCtrlC(t *testing.T) {
	m := app.AppModel{}
	handled := m.HandleGlobalKeys(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled {
		t.Fatal("expected ctrl+c to be handled")
	}
	if !m.Quit {
		t.Fatal("expected Quit=true after ctrl+c")
	}
}

func TestHandleGlobalKeysWindowSize(t *testing.T) {
	m := app.AppModel{}
	handled := m.HandleGlobalKeys(tea.WindowSizeMsg{Width: 120, Height: 40})
	if handled {
		t.Fatal("expected WindowSizeMsg not to be handled (returns false)")
	}
	if m.Width != 120 {
		t.Errorf("expected Width=120, got %d", m.Width)
	}
	if m.Height != 40 {
		t.Errorf("expected Height=40, got %d", m.Height)
	}
}

func TestStepConstants(t *testing.T) {
	if app.StepMenu != 0 {
		t.Errorf("expected StepMenu=0, got %d", app.StepMenu)
	}
	if app.StepCustom != 1 {
		t.Errorf("expected StepCustom=1, got %d", app.StepCustom)
	}
}
