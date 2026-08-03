package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
)

// compile-time interface check
var _ app.Screen = (*mockScreen)(nil)

type mockScreen struct {
	id string
}

func (m *mockScreen) ID() string                        { return m.id }
func (m *mockScreen) Init() tea.Cmd                     { return nil }
func (m *mockScreen) Update(tea.Msg) (tea.Cmd, error)   { return nil, nil }
func (m *mockScreen) View(int, int) string               { return "" }
func (m *mockScreen) Focus() tea.Cmd                     { return nil }
func (m *mockScreen) Blur() tea.Cmd                      { return nil }

func TestPushPop(t *testing.T) {
	s1 := &mockScreen{id: "one"}
	s2 := &mockScreen{id: "two"}

	// Use Push and Pop commands
	pushCmd := app.Push(s1)
	msg := pushCmd()
	if _, ok := msg.(tea.Msg); !ok {
		t.Fatal("Push did not return a tea.Msg")
	}

	popCmd := app.Pop()
	msg2 := popCmd()
	if _, ok := msg2.(tea.Msg); !ok {
		t.Fatal("Pop did not return a tea.Msg")
	}

	_ = s2 // used in stack tests below
}

func TestReplace(t *testing.T) {
	s1 := &mockScreen{id: "one"}
	s2 := &mockScreen{id: "two"}

	replaceCmd := app.Replace(s1)
	msg := replaceCmd()
	if _, ok := msg.(tea.Msg); !ok {
		t.Fatal("Replace did not return a tea.Msg")
	}

	replaceCmd2 := app.Replace(s2)
	msg2 := replaceCmd2()
	if _, ok := msg2.(tea.Msg); !ok {
		t.Fatal("Replace did not return a tea.Msg")
	}
}

func TestBaseModelDefaults(t *testing.T) {
	m := app.BaseModel{}
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
	if m.Quit {
		t.Error("expected Quit=false")
	}
}

func TestScreenInterfaceCompileTime(t *testing.T) {
	//nolint:gosimple
	var s app.Screen = &mockScreen{id: "check"}
	if s.ID() != "check" {
		t.Fatalf("unexpected id: %s", s.ID())
	}
}

func TestErrorf(t *testing.T) {
	errCmd := app.Errorf("something went wrong: %s", "bad")
	msg := errCmd()
	if msg == nil {
		t.Fatal("Errorf returned nil message")
	}
}
