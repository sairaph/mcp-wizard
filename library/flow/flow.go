package flow

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Flow runs a sequence of steps. It owns the single tea.Model.
// Create it with New[T], then call Model() to get the tea.Model
// for use with tea.NewProgram.
type Flow[T any] struct {
	steps   []Step[T]
	current int
	state   *T
}

// New creates a Flow with the given steps and initial state.
func New[T any](steps []Step[T], state *T) *Flow[T] {
	return &Flow[T]{steps: steps, state: state}
}

// Steps returns the current step list.
func (f *Flow[T]) Steps() []Step[T] { return f.steps }

// State returns the shared state.
func (f *Flow[T]) State() *T { return f.state }

// Current returns the current step index.
func (f *Flow[T]) Current() int { return f.current }

// Advance moves to the next step. Returns false if already at the end.
func (f *Flow[T]) Advance() bool {
	if f.current >= len(f.steps)-1 {
		return false
	}
	f.current++
	return true
}

// Retreat moves to the previous step. Returns false if already at the start.
func (f *Flow[T]) Retreat() bool {
	if f.current <= 0 {
		return false
	}
	f.current--
	return true
}

// JumpTo moves to the step with the given ID. Returns false if not found.
func (f *Flow[T]) JumpTo(id string) bool {
	for i, step := range f.steps {
		if step.ID() == id {
			f.current = i
			return true
		}
	}
	return false
}

// Model returns a tea.Model that drives the Flow.
// Use it with tea.NewProgram.
func (f *Flow[T]) Model() tea.Model {
	return &flowModel[T]{flow: f}
}

// ExitCode returns the exit code after the program finishes:
//
//	0 — normal completion or Quit before Settled
//	1 — Fail (state.Failure is non-nil)
func (f *Flow[T]) ExitCode() int {
	if f.state == nil {
		return 0
	}
	var base *BaseState
	if b, ok := any(f.state).(interface{ GetBaseState() *BaseState }); ok {
		base = b.GetBaseState()
	}
	if base != nil && base.Failure != nil {
		return 1
	}
	return 0
}

// GetBaseState is the interface consumer state structs implement
// by embedding *BaseState.
func (b *BaseState) GetBaseState() *BaseState { return b }

// flowModel is the Bubble Tea model that wraps the Flow.
type flowModel[T any] struct {
	flow *Flow[T]
	step Step[T]
	cmd  tea.Cmd
}

func (m *flowModel[T]) Init() tea.Cmd {
	m.step = m.flow.steps[m.flow.current]
	return m.step.Init(m.flow.state)
}

func (m *flowModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wm, ok := msg.(tea.WindowSizeMsg); ok {
		var base *BaseState
		if b, ok := any(m.flow.state).(interface{ GetBaseState() *BaseState }); ok {
			base = b.GetBaseState()
		}
		if base != nil {
			base.Width = wm.Width
			base.Height = wm.Height
		}
		return m, nil
	}

	if _, ok := msg.(tea.QuitMsg); ok {
		return m, tea.Quit
	}

	directive, cmd := m.step.Update(msg, m.flow.state)

	switch directive {
	case Next:
		if !m.flow.Advance() {
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		return m, m.step.Init(m.flow.state)

	case Back:
		if !m.flow.Retreat() {
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		return m, m.step.Init(m.flow.state)

	case Skip:
		if !m.flow.Advance() {
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		return m, m.step.Init(m.flow.state)

	case Jump:
		nextStep := ""
		if b, ok := any(m.flow.state).(interface{ GetBaseState() *BaseState }); ok {
			nextStep = b.GetBaseState().NextStep
		}
		if !m.flow.JumpTo(nextStep) {
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		return m, m.step.Init(m.flow.state)

	case Quit:
		return m, tea.Quit

	case Fail:
		return m, tea.Quit

	default:
		return m, cmd
	}
}

func (m *flowModel[T]) View() string {
	if m.step == nil {
		return ""
	}
	return m.step.View(m.flow.state)
}
