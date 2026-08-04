// Package installer provides pre-built flow.Step constructors for
// MCP server install wizards and unattended-mode helpers.
package installer

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/flow"
	"github.com/sairaph/mcp-wizard/harness"
	"github.com/sairaph/mcp-wizard/tui"
)

// HarnessStepOptions controls the harness selection step.
type HarnessStepOptions struct {
	AllDetected     bool
	ConflictPolicy  harness.ConflictPolicy
	ShowUnavailable bool
	// Scope selects where detection and registration happen. The zero value is
	// global scope and preserves existing behavior. Set to harness.ProjectScopeDir(dir)
	// to detect and register in a per-project configuration.
	Scope harness.Scope
}

// HarnessState is embedded in consumer state for harness selection.
type HarnessState struct {
	Detections []harness.Harness
	Selected   map[harness.ID]bool
	Cursor     int
	ShowAll    bool
	// Scope is the scope used for detection, recorded so downstream steps know
	// whether registration targets a project or global configuration.
	Scope harness.Scope
}

// HarnessStep returns a flow.Step that detects harnesses and lets the user
// select which ones to register with.
func HarnessStep[T any](ctx context.Context, detector *harness.Detector, stateFn func(*T) *HarnessState, opts HarnessStepOptions) flow.Step[T] {
	if detector == nil {
		panic("installer: HarnessStep requires a non-nil detector")
	}
	return &harnessStep[T]{
		ctx:      ctx,
		detector: detector,
		stateFn:  stateFn,
		opts:     opts,
	}
}

type harnessStep[T any] struct {
	ctx      context.Context
	detector *harness.Detector
	stateFn  func(*T) *HarnessState
	opts     HarnessStepOptions
	ready    bool
}

func (s *harnessStep[T]) ID() string { return "harnesses" }

func (s *harnessStep[T]) Title(state *T) string {
	if s.opts.Scope.IsProject() {
		return fmt.Sprintf("AI clients - register in this project (%s), which should be able to use this server?", s.opts.Scope.Dir)
	}
	return "AI clients - which should be able to use this server?"
}

func (s *harnessStep[T]) Hints(state *T) []struct{ Key, Label string } {
	return []struct{ Key, Label string }{
		{Key: "\u2191\u2193", Label: "move"},
		{Key: "space", Label: "toggle"},
		{Key: "a", Label: "all/none"},
		{Key: "v", Label: "show all"},
		{Key: "enter", Label: "continue"},
		{Key: "q", Label: "cancel"},
	}
}

type detectedMsg struct {
	harnesses []harness.Harness
}

func (s *harnessStep[T]) Init(state *T) tea.Cmd {
	if s.stateFn == nil {
		return nil
	}
	hState := s.stateFn(state)
	if hState == nil {
		return nil
	}
	hState.Scope = s.opts.Scope
	return tea.Batch(
		tui.Spinner(),
		func() tea.Msg {
			if s.opts.Scope.IsProject() {
				return detectedMsg{harnesses: s.detector.DetectIn(s.ctx, s.opts.Scope)}
			}
			return detectedMsg{harnesses: s.detector.Detect(s.ctx)}
		},
	)
}

func (s *harnessStep[T]) Update(msg tea.Msg, state *T) (flow.Directive, tea.Cmd) {
	if s.stateFn == nil {
		return flow.Fail, nil
	}
	hState := s.stateFn(state)
	if hState == nil {
		return flow.Fail, nil
	}

	switch m := msg.(type) {
	case detectedMsg:
		hState.Detections = m.harnesses
		hState.Selected = make(map[harness.ID]bool)
		for _, h := range m.harnesses {
			if h.Configured || (s.opts.AllDetected && h.Selectable()) {
				hState.Selected[h.ID] = true
			}
		}
		hState.Cursor = FirstSelectable(m.harnesses)
		if hState.Cursor < 0 {
			indices := VisibleIndices(m.harnesses, false)
			if len(indices) > 0 {
				hState.Cursor = indices[0]
			}
		}
		s.ready = true
		return flow.Continue, nil

	case tea.KeyMsg:
		if !s.ready {
			if m.String() == "q" || m.String() == "ctrl+c" {
				return flow.Quit, nil
			}
			return flow.Continue, nil
		}

		switch m.String() {
		case "q", "ctrl+c":
			return flow.Quit, nil
		case "up", "k":
			MoveCursor(hState, hState.Detections, -1)
		case "down", "j":
			MoveCursor(hState, hState.Detections, 1)
		case " ":
			ToggleHarness(hState)
		case "a":
			ToggleAll(hState)
		case "v":
			hState.ShowAll = !hState.ShowAll
			if !hState.ShowAll {
				indices := VisibleIndices(hState.Detections, false)
				if len(indices) > 0 {
					found := false
					for _, idx := range indices {
						if idx == hState.Cursor {
							found = true
							break
						}
					}
					if !found {
						hState.Cursor = indices[0]
					}
				}
			}
		case "enter":
			count := 0
			for _, v := range hState.Selected {
				if v {
					count++
				}
			}
			if count == 0 {
				return flow.Continue, nil
			}
			return flow.Next, nil
		case "esc":
			return flow.Back, nil
		}
	}

	if tui.IsSpinMsg(msg) && !s.ready {
		return flow.Continue, tui.Spinner()
	}

	return flow.Continue, nil
}

func (s *harnessStep[T]) View(state *T) string {
	if s.stateFn == nil {
		return ""
	}
	hState := s.stateFn(state)
	if hState == nil {
		return ""
	}

	if !s.ready {
		return tui.Section(tui.DefaultTheme, "", "  Initialising, looking for AI clients...\n")
	}

	indices := VisibleIndices(hState.Detections, hState.ShowAll)
	items := make([]tui.CheckboxItem, len(indices))
	for i, idx := range indices {
		items[i] = tui.CheckboxItem{ID: string(hState.Detections[idx].ID), Name: hState.Detections[idx].Name}
	}

	selected := make(map[string]bool)
	for id := range hState.Selected {
		selected[string(id)] = true
	}

	selectable := func(i int) bool {
		if i < 0 || i >= len(indices) {
			return false
		}
		return hState.Detections[indices[i]].Selectable()
	}

	statusFn := func(i int) string {
		if i < 0 || i >= len(indices) {
			return ""
		}
		return hState.Detections[indices[i]].StatusText()
	}

	hidden := len(hState.Detections) - len(indices)

	cursorPos := 0
	for i, idx := range indices {
		if idx == hState.Cursor {
			cursorPos = i
			break
		}
	}

	content := tui.CheckboxList(tui.DefaultTheme, items, cursorPos, selected, selectable, statusFn, hState.ShowAll, hidden)
	content += "\n" + tui.Footer(tui.DefaultTheme, tui.Hints(tui.DefaultTheme,
		tui.Hint{Key: "\u2191\u2193", Label: "move"},
		tui.Hint{Key: "space", Label: "toggle"},
		tui.Hint{Key: "a", Label: "all/none"},
		tui.Hint{Key: "v", Label: "show all"},
		tui.Hint{Key: "enter", Label: "continue"},
		tui.Hint{Key: "q", Label: "cancel"},
	))

	return tui.Section(tui.DefaultTheme, s.Title(state), content)
}

// --- helpers ---

func VisibleIndices(harnesses []harness.Harness, showAll bool) []int {
	var indices []int
	for i, h := range harnesses {
		if h.State == harness.Detected || h.Configured || showAll {
			indices = append(indices, i)
		}
	}
	return indices
}

func FirstSelectable(harnesses []harness.Harness) int {
	for i, h := range harnesses {
		if h.Selectable() {
			return i
		}
	}
	return -1
}

func MoveCursor(state *HarnessState, harnesses []harness.Harness, dir int) {
	indices := VisibleIndices(harnesses, state.ShowAll)
	if len(indices) == 0 {
		return
	}
	pos := 0
	for i, idx := range indices {
		if idx == state.Cursor {
			pos = i
			break
		}
	}
	pos += dir
	if pos < 0 {
		pos = len(indices) - 1
	}
	if pos >= len(indices) {
		pos = 0
	}
	state.Cursor = indices[pos]
}

func ToggleHarness(state *HarnessState) {
	if state == nil || state.Cursor < 0 || state.Cursor >= len(state.Detections) {
		return
	}
	if state.Selected == nil {
		state.Selected = make(map[harness.ID]bool)
	}
	h := state.Detections[state.Cursor]
	if !h.Selectable() {
		return
	}
	state.Selected[h.ID] = !state.Selected[h.ID]
}

func ToggleAll(state *HarnessState) {
	if state == nil {
		return
	}
	if state.Selected == nil {
		state.Selected = make(map[harness.ID]bool)
	}
	anyUnselected := false
	for _, h := range state.Detections {
		if h.Selectable() && !state.Selected[h.ID] {
			anyUnselected = true
			break
		}
	}
	for _, h := range state.Detections {
		if h.Selectable() {
			state.Selected[h.ID] = anyUnselected
		}
	}
}
