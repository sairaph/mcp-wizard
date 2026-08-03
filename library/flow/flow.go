package flow

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Flow runs a sequence of steps. It owns the single tea.Model.
// Create it with New[T], then call Model() to get the tea.Model
// for use with tea.NewProgram.
type Flow[T any] struct {
	steps       []Step[T]
	current     int
	state       *T
	flowFailure error
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
//
// The state type T must embed *BaseState for Failure detection;
// otherwise, ExitCode always returns 0.
func (f *Flow[T]) ExitCode() int {
	if f.flowFailure != nil {
		return 1
	}
	if f.state == nil {
		return 0
	}
	if base := f.getBaseState(); base != nil && base.Failure != nil {
		return 1
	}
	return 0
}

func (f *Flow[T]) getBaseState() *BaseState {
	if f.state == nil {
		return nil
	}
	if b, ok := any(f.state).(interface{ GetBaseState() *BaseState }); ok {
		return b.GetBaseState()
	}
	return nil
}

// GetBaseState is the interface consumer state structs implement
// by embedding *BaseState.
func (b *BaseState) GetBaseState() *BaseState { return b }

// flowModel is the Bubble Tea model that wraps the Flow.
type flowModel[T any] struct {
	flow *Flow[T]
	step Step[T]
}

func (m *flowModel[T]) Init() tea.Cmd {
	if len(m.flow.steps) == 0 || m.flow.current < 0 || m.flow.current >= len(m.flow.steps) || m.flow.state == nil {
		return tea.Quit
	}
	m.step = m.flow.steps[m.flow.current]
	if m.step == nil {
		return tea.Quit
	}
	return m.step.Init(m.flow.state)
}

func (m *flowModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.step == nil || m.flow.state == nil {
		return m, tea.Quit
	}
	// Update base state dimensions on resize, then forward to step.
	if wm, ok := msg.(tea.WindowSizeMsg); ok {
		if base := m.flow.getBaseState(); base != nil {
			base.Width = wm.Width
			base.Height = wm.Height
		}
	}

	if _, ok := msg.(tea.QuitMsg); ok {
		directive, cmd := m.step.Update(msg, m.flow.state)
		if directive == Fail {
			if base := m.flow.getBaseState(); base != nil {
				base.Failure = fmt.Errorf("flow: step %q returned Fail on quit", m.step.ID())
			} else {
				m.flow.flowFailure = fmt.Errorf("flow: step %q returned Fail on quit", m.step.ID())
			}
		}
		return m, tea.Batch(cmd, tea.Quit)
	}

	directive, cmd := m.step.Update(msg, m.flow.state)

	switch directive {
	case Next:
		stepCmd := cmd
		if !m.flow.Advance() {
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		if m.step == nil {
			return m, tea.Quit
		}
		initCmd := m.step.Init(m.flow.state)
		return m, tea.Batch(stepCmd, initCmd)

	case Back:
		stepCmd := cmd
		if !m.flow.Retreat() {
			return m, stepCmd // return the step's cmd, don't discard it
		}
		m.step = m.flow.steps[m.flow.current]
		if m.step == nil {
			return m, tea.Quit
		}
		initCmd := m.step.Init(m.flow.state)
		return m, tea.Batch(stepCmd, initCmd)

	// Skip advances to the next step without rendering the current one.
	// This case is reached when a step's Update returns Skip.
	case Skip:
		stepCmd := cmd
		if !m.flow.Advance() {
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		if m.step == nil {
			return m, tea.Quit
		}
		initCmd := m.step.Init(m.flow.state)
		return m, tea.Batch(stepCmd, initCmd)

	case Jump:
		stepCmd := cmd
		base := m.flow.getBaseState()
		nextStep := ""
		if base != nil {
			nextStep = base.NextStep
		}
		if nextStep == "" {
			if base != nil {
				base.Failure = fmt.Errorf("flow: Jump directive used without setting NextStep")
			} else {
				m.flow.flowFailure = fmt.Errorf("flow: Jump directive used without setting NextStep")
			}
			return m, tea.Quit
		}
		if !m.flow.JumpTo(nextStep) {
			if base != nil {
				base.Failure = fmt.Errorf("flow: no step with ID %q", nextStep)
			} else {
				m.flow.flowFailure = fmt.Errorf("flow: no step with ID %q", nextStep)
			}
			return m, tea.Quit
		}
		m.step = m.flow.steps[m.flow.current]
		if m.step == nil {
			return m, tea.Quit
		}
		initCmd := m.step.Init(m.flow.state)
		return m, tea.Batch(stepCmd, initCmd)

	case Quit:
		return m, tea.Quit

	case Fail:
		if base := m.flow.getBaseState(); base != nil {
			if base.Failure == nil {
				base.Failure = fmt.Errorf("flow: step %q returned Fail", m.step.ID())
			}
		} else {
			m.flow.flowFailure = fmt.Errorf("flow: step %q returned Fail", m.step.ID())
		}
		return m, tea.Quit

	default:
		return m, cmd
	}
}

func (m *flowModel[T]) View() string {
	if m.step == nil || m.flow.state == nil {
		return ""
	}
	return m.step.View(m.flow.state)
}
