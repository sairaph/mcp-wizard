package installer

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
	"github.com/sairaph/mcp-wizard/harness"
	"github.com/sairaph/mcp-wizard/tui"
)

// ApplyStepOptions controls the registration step.
type ApplyStepOptions struct {
	// Scope selects where registration happens. Use the same scope as the
	// HarnessStep that collected the selection.
	Scope harness.Scope
	// DryRun computes the plan and shows it instead of writing anything.
	DryRun bool
	// ConflictPolicy decides what happens to a same-name entry that differs.
	// The zero value is harness.ConflictReplace.
	ConflictPolicy harness.ConflictPolicy
}

// ResultsState is embedded in consumer state for the apply step.
type ResultsState struct {
	Results []harness.Result // per-client outcome (not set in dry-run)
	Changes []harness.Change // planned changes (dry-run only)
	Done    bool
}

// ApplyStep returns a flow.Step that registers the server with the harnesses
// selected in a HarnessStep, shows the outcome with reload hints, and marks
// the flow's BaseState as Settled. It is the final step of an install wizard:
// the wizard would otherwise collect a selection and discard it.
//
// harnessFn returns the HarnessState holding the selection; resultsFn
// returns where to store the outcome.
func ApplyStep[T any](ctx context.Context, detector *harness.Detector, harnessFn func(*T) *HarnessState, resultsFn func(*T) *ResultsState, opts ApplyStepOptions) flow.Step[T] {
	if detector == nil {
		panic("installer: ApplyStep requires a non-nil detector")
	}
	if opts.ConflictPolicy == "" {
		opts.ConflictPolicy = harness.ConflictReplace
	}
	return &applyStep[T]{
		ctx:       ctx,
		detector:  detector,
		harnessFn: harnessFn,
		resultsFn: resultsFn,
		opts:      opts,
	}
}

type applyStep[T any] struct {
	ctx       context.Context
	detector  *harness.Detector
	harnessFn func(*T) *HarnessState
	resultsFn func(*T) *ResultsState
	opts      ApplyStepOptions
}

type appliedMsg struct {
	results []harness.Result
	changes []harness.Change
	err     error
}

func (s *applyStep[T]) ID() string { return "apply" }

func (s *applyStep[T]) Title(state *T) string {
	if s.opts.DryRun {
		return "Planned changes - nothing has been written"
	}
	return "Registration"
}

func (s *applyStep[T]) Hints(state *T) []struct{ Key, Label string } {
	rs := s.results(state)
	if rs == nil || !rs.Done {
		if s.opts.DryRun {
			return []struct{ Key, Label string }{{Key: "q", Label: "cancel"}}
		}
		return nil // the write is in flight and cannot be cancelled
	}
	return []struct{ Key, Label string }{{Key: "enter", Label: "finish"}}
}

func (s *applyStep[T]) results(state *T) *ResultsState {
	if s.resultsFn == nil {
		return nil
	}
	return s.resultsFn(state)
}

// selectedIDs returns the harness IDs chosen in the HarnessStep, in
// detection order.
func (s *applyStep[T]) selectedIDs(state *T) []harness.ID {
	if s.harnessFn == nil {
		return nil
	}
	hs := s.harnessFn(state)
	if hs == nil {
		return nil
	}
	var ids []harness.ID
	for _, h := range hs.Detections {
		if hs.Selected[h.ID] {
			ids = append(ids, h.ID)
		}
	}
	return ids
}

func (s *applyStep[T]) Init(state *T) tea.Cmd {
	rs := s.results(state)
	if rs == nil {
		return nil
	}
	rs.Results = nil
	rs.Changes = nil
	rs.Done = false

	ids := s.selectedIDs(state)
	if len(ids) == 0 {
		return func() tea.Msg { return appliedMsg{} }
	}
	scope, policy, dryRun := s.opts.Scope, s.opts.ConflictPolicy, s.opts.DryRun
	return tea.Batch(tui.Spinner(), func() tea.Msg {
		if dryRun {
			changes, err := s.detector.PlanResultsIn(s.ctx, scope, ids, harness.Present, policy)
			return appliedMsg{changes: changes, err: err}
		}
		return appliedMsg{results: s.detector.ApplyIn(s.ctx, scope, ids, harness.Present, policy)}
	})
}

func (s *applyStep[T]) Update(msg tea.Msg, state *T) (flow.Directive, tea.Cmd) {
	rs := s.results(state)
	if rs == nil {
		return flow.Fail, nil
	}

	switch m := msg.(type) {
	case appliedMsg:
		rs.Done = true
		rs.Results = m.results
		rs.Changes = m.changes
		if base := baseStateOf(state); base != nil {
			if m.err != nil {
				base.Failure = m.err
				return flow.Fail, nil
			}
			// The work is complete; cancelling from here on is a normal exit.
			base.Settled = true
			for _, r := range m.results {
				if r.State == harness.ApplyFailed {
					base.Failure = fmt.Errorf("registration failed for %s: %s", r.Name, r.Reason)
					break
				}
			}
		}
		return flow.Continue, nil

	case tea.KeyMsg:
		switch m.String() {
		case "ctrl+c":
			return flow.Quit, nil
		case "q":
			// While a real write is in flight, quitting would report a
			// clean exit for a registration that may still complete.
			if rs.Done || s.opts.DryRun {
				return flow.Quit, nil
			}
		case "enter":
			if rs.Done {
				return flow.Next, nil
			}
		}
	}

	if tui.IsSpinMsg(msg) && !rs.Done {
		if base := baseStateOf(state); base != nil {
			base.Spinner.Frame++
		}
		return flow.Continue, tui.Spinner()
	}
	return flow.Continue, nil
}

func (s *applyStep[T]) View(state *T) string {
	rs := s.results(state)
	if rs == nil {
		return ""
	}
	theme := tui.DefaultTheme
	var b strings.Builder

	if !rs.Done {
		frame := 0
		if base := baseStateOf(state); base != nil {
			frame = base.Spinner.Frame
		}
		verb := "Registering with"
		if s.opts.DryRun {
			verb = "Planning changes for"
		}
		fmt.Fprintf(&b, "  %s %s the selected clients...\n", tui.SpinFrame(frame), verb)
		return tui.Section(theme, s.Title(state), b.String())
	}

	var hs *HarnessState
	if s.harnessFn != nil {
		hs = s.harnessFn(state)
	}
	switch {
	case s.opts.DryRun:
		PrintChanges(&b, rs.Changes, s.opts.Scope)
	case len(rs.Results) == 0:
		b.WriteString("  No clients were selected, so nothing was configured.\n")
	default:
		PrintResultsWithScope(&b, rs.Results, s.opts.Scope, true, false)
		if hs != nil {
			byID := make(map[harness.ID]harness.Harness, len(hs.Detections))
			for _, h := range hs.Detections {
				byID[h.ID] = h
			}
			PrintReloadHints(&b, rs.Results, byID)
		}
	}

	b.WriteString("\n" + tui.Footer(theme, tui.Hints(theme, tui.Hint{Key: "enter", Label: "finish"})))
	return tui.Section(theme, s.Title(state), b.String())
}

// baseStateOf returns the embedded *flow.BaseState when the consumer state
// exposes one, mirroring how the Flow runner finds it.
func baseStateOf[T any](state *T) *flow.BaseState {
	if state == nil {
		return nil
	}
	if b, ok := any(state).(interface{ GetBaseState() *flow.BaseState }); ok {
		return b.GetBaseState()
	}
	return nil
}
