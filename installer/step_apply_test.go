package installer_test

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
	"github.com/sairaph/mcp-wizard/harness"
	"github.com/sairaph/mcp-wizard/installer"
)

type applyTestState struct {
	flow.BaseState
	Harness installer.HarnessState
	Results installer.ResultsState
}

func newApplyStep(t *testing.T, opts installer.ApplyStepOptions) (flow.Step[applyTestState], *applyTestState) {
	t.Helper()
	detector, err := harness.New(harness.ServerSpec{Name: "apply-test", Command: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	step := installer.ApplyStep(context.Background(), detector,
		func(s *applyTestState) *installer.HarnessState { return &s.Harness },
		func(s *applyTestState) *installer.ResultsState { return &s.Results },
		opts)
	return step, &applyTestState{}
}

// runCmd executes a tea.Cmd and returns the messages it produces, expanding
// batches. Spinner ticks are skipped.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, runCmd(c)...)
		}
		return out
	}
	if _, isTick := msg.(interface{ Tick() }); isTick {
		return nil
	}
	return []tea.Msg{msg}
}

func TestApplyStepPanicsWithoutDetector(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for nil detector")
		}
	}()
	installer.ApplyStep[applyTestState](context.Background(), nil, nil, nil, installer.ApplyStepOptions{})
}

func TestApplyStepNothingSelectedFinishesImmediately(t *testing.T) {
	step, state := newApplyStep(t, installer.ApplyStepOptions{})
	state.Harness.Detections = []harness.Harness{{ID: "x", Name: "X", State: harness.Detected}}
	state.Harness.Selected = map[harness.ID]bool{}

	if step.ID() != "apply" {
		t.Fatalf("ID = %q", step.ID())
	}
	cmd := step.Init(state)
	if cmd == nil {
		t.Fatal("Init must return a command that delivers the empty result")
	}
	for _, msg := range runCmd(cmd) {
		step.Update(msg, state)
	}
	if !state.Results.Done {
		t.Fatal("step should be done after the empty result")
	}
	if !state.Settled {
		t.Fatal("finishing the apply step must mark the flow Settled")
	}
	v := step.View(state)
	if !strings.Contains(v, "No clients were selected") {
		t.Fatalf("view should explain that nothing happened, got:\n%s", v)
	}
	d, _ := step.Update(tea.KeyMsg{Type: tea.KeyEnter}, state)
	if d != flow.Next {
		t.Fatalf("enter after Done must advance, got %v", d)
	}
}

func TestApplyStepBeforeDoneShowsSpinnerAndIgnoresEnter(t *testing.T) {
	// Unknown ID inside a temp project scope: even if the apply command ran,
	// nothing on the machine could be written.
	step, state := newApplyStep(t, installer.ApplyStepOptions{Scope: harness.ProjectScopeDir(t.TempDir())})
	state.Harness.Detections = []harness.Harness{{ID: "no-such-client", Name: "Nope", State: harness.Detected}}
	state.Harness.Selected = map[harness.ID]bool{"no-such-client": true}
	_ = step.Init(state)

	if v := step.View(state); !strings.Contains(v, "Registering") {
		t.Fatalf("expected in-progress view, got:\n%s", v)
	}
	if d, _ := step.Update(tea.KeyMsg{Type: tea.KeyEnter}, state); d != flow.Continue {
		t.Fatalf("enter before Done must not advance, got %v", d)
	}
	if d, _ := step.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, state); d != flow.Continue {
		t.Fatalf("q while a write is in flight must be ignored, got %v", d)
	}
	if d, _ := step.Update(tea.KeyMsg{Type: tea.KeyCtrlC}, state); d != flow.Quit {
		t.Fatalf("ctrl+c must still quit, got %v", d)
	}
	if hints := step.Hints(state); len(hints) != 0 {
		t.Fatalf("no hints while the write is in flight, got %v", hints)
	}
}

func TestApplyStepDryRunPlansWithoutWriting(t *testing.T) {
	step, state := newApplyStep(t, installer.ApplyStepOptions{DryRun: true, Scope: harness.ProjectScopeDir(t.TempDir())})
	// Select an ID detect-harness does not know so planning cannot touch
	// the real machine; the plan reports it as unavailable.
	state.Harness.Detections = []harness.Harness{{ID: "no-such-client", Name: "Nope", State: harness.Detected}}
	state.Harness.Selected = map[harness.ID]bool{"no-such-client": true}

	for _, msg := range runCmd(step.Init(state)) {
		step.Update(msg, state)
	}
	if !state.Results.Done {
		t.Fatal("dry run should complete")
	}
	if len(state.Results.Results) != 0 {
		t.Fatalf("dry run must not apply, got results %v", state.Results.Results)
	}
	if len(state.Results.Changes) != 1 {
		t.Fatalf("expected one planned change, got %v", state.Results.Changes)
	}
	if !strings.Contains(step.Title(state), "nothing has been written") {
		t.Fatalf("dry-run title = %q", step.Title(state))
	}
	// detect-harness does not know the ID, so the change carries no Name;
	// the view falls back to the ID and shows the state and reason.
	if v := step.View(state); !strings.Contains(v, "no-such-client") || !strings.Contains(v, "unavailable") {
		t.Fatalf("dry-run view should list the client and why, got:\n%s", v)
	}
}

func TestApplyStepWithoutBaseStateStillCompletes(t *testing.T) {
	type plain struct {
		Harness installer.HarnessState
		Results installer.ResultsState
	}
	detector, err := harness.New(harness.ServerSpec{Name: "apply-test", Command: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	step := installer.ApplyStep(context.Background(), detector,
		func(s *plain) *installer.HarnessState { return &s.Harness },
		func(s *plain) *installer.ResultsState { return &s.Results },
		installer.ApplyStepOptions{})
	state := &plain{}
	for _, msg := range runCmd(step.Init(state)) {
		step.Update(msg, state)
	}
	if !state.Results.Done {
		t.Fatal("step must complete for state types without BaseState")
	}
}
