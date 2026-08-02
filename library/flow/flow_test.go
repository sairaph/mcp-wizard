package flow_test

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
)

// compileTimeCheck ensures Step[mockState] is satisfied.
var _ flow.Step[mockState] = (*mockStep[mockState])(nil)

type mockState struct {
	*flow.BaseState
	Value string
}

type mockStep[T any] struct {
	id       string
	title    string
	updateFn func(msg tea.Msg, state *T) (flow.Directive, tea.Cmd)
}

func (m *mockStep[T]) ID() string                                          { return m.id }
func (m *mockStep[T]) Title(state *T) string                               { return m.title }
func (m *mockStep[T]) Hints(state *T) []struct{ Key, Label string }        { return nil }
func (m *mockStep[T]) Init(state *T) tea.Cmd                               { return nil }
func (m *mockStep[T]) Update(msg tea.Msg, state *T) (flow.Directive, tea.Cmd) {
	if m.updateFn != nil {
		return m.updateFn(msg, state)
	}
	return flow.Continue, nil
}
func (m *mockStep[T]) View(state *T) string                                { return "" }

func TestNewFlowEmptySteps(t *testing.T) {
	f := flow.New([]flow.Step[int]{}, nil)
	m := f.Model()
	if m == nil {
		t.Fatal("Model() returned nil")
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected a quit command from Init on empty steps")
	}
}

func TestAdvance(t *testing.T) {
	steps := []flow.Step[int]{
		&mockStep[int]{id: "a"},
		&mockStep[int]{id: "b"},
		&mockStep[int]{id: "c"},
	}
	f := flow.New(steps, nil)

	if got := f.Current(); got != 0 {
		t.Fatalf("expected current=0, got %d", got)
	}

	if !f.Advance() {
		t.Fatal("expected Advance to return true")
	}
	if got := f.Current(); got != 1 {
		t.Fatalf("expected current=1, got %d", got)
	}

	if !f.Advance() {
		t.Fatal("expected Advance to return true")
	}
	if got := f.Current(); got != 2 {
		t.Fatalf("expected current=2, got %d", got)
	}

	if f.Advance() {
		t.Fatal("expected Advance to return false at end")
	}
	if got := f.Current(); got != 2 {
		t.Fatalf("expected current=2, got %d", got)
	}
}

func TestRetreat(t *testing.T) {
	steps := []flow.Step[int]{
		&mockStep[int]{id: "a"},
		&mockStep[int]{id: "b"},
	}
	f := flow.New(steps, nil)
	f.Advance()

	if !f.Retreat() {
		t.Fatal("expected Retreat to return true")
	}
	if got := f.Current(); got != 0 {
		t.Fatalf("expected current=0, got %d", got)
	}

	if f.Retreat() {
		t.Fatal("expected Retreat to return false at start")
	}
	if got := f.Current(); got != 0 {
		t.Fatalf("expected current=0, got %d", got)
	}
}

func TestJumpToFound(t *testing.T) {
	steps := []flow.Step[int]{
		&mockStep[int]{id: "first"},
		&mockStep[int]{id: "second"},
		&mockStep[int]{id: "third"},
	}
	f := flow.New(steps, nil)

	if !f.JumpTo("second") {
		t.Fatal("expected JumpTo to return true")
	}
	if got := f.Current(); got != 1 {
		t.Fatalf("expected current=1, got %d", got)
	}
}

func TestJumpToNotFound(t *testing.T) {
	steps := []flow.Step[int]{
		&mockStep[int]{id: "a"},
		&mockStep[int]{id: "b"},
	}
	f := flow.New(steps, nil)

	if f.JumpTo("nonexistent") {
		t.Fatal("expected JumpTo to return false")
	}
	if got := f.Current(); got != 0 {
		t.Fatalf("expected current=0, got %d", got)
	}
}

func TestExitCodeNoState(t *testing.T) {
	f := flow.New([]flow.Step[int]{}, nil)
	if got := f.ExitCode(); got != 0 {
		t.Fatalf("expected exit code 0, got %d", got)
	}
}

func TestExitCodeNoFailure(t *testing.T) {
	state := &mockState{BaseState: &flow.BaseState{}}
	steps := []flow.Step[mockState]{
		&mockStep[mockState]{id: "a"},
	}
	f := flow.New(steps, state)
	if got := f.ExitCode(); got != 0 {
		t.Fatalf("expected exit code 0, got %d", got)
	}
}

func TestExitCodeWithFailure(t *testing.T) {
	state := &mockState{BaseState: &flow.BaseState{Failure: errors.New("boom")}}
	steps := []flow.Step[mockState]{
		&mockStep[mockState]{id: "a"},
	}
	f := flow.New(steps, state)
	if got := f.ExitCode(); got != 1 {
		t.Fatalf("expected exit code 1, got %d", got)
	}
}

func TestBaseStateGetBaseState(t *testing.T) {
	bs := &flow.BaseState{Message: "hello"}
	if got := bs.GetBaseState(); got != bs {
		t.Fatal("GetBaseState did not return itself")
	}
}

func TestFlowModel_Update(t *testing.T) {
	called := false
	step := &mockStep[mockState]{}
	step.updateFn = func(msg tea.Msg, state *mockState) (flow.Directive, tea.Cmd) {
		called = true
		return flow.Quit, nil
	}
	f := flow.New[mockState]([]flow.Step[mockState]{step}, &mockState{})
	m := f.Model()
	m.Init()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes})
	if !called {
		t.Fatal("step.Update was not called")
	}
	_ = cmd
}
