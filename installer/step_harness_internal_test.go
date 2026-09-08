package installer

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/harness"
)

// TestHarnessStepInit_projectDispatchUsesDetectIn drives the command returned
// by Init for a project scope and asserts that the detection sub-command emits a
// detectedMsg, confirming Init dispatches to Detector.DetectIn for project
// scopes rather than the global Detect.
func TestHarnessStepInit_projectDispatchUsesDetectIn(t *testing.T) {
	detector, err := harness.New(harness.ServerSpec{Name: "test", Command: "echo"})
	if err != nil {
		t.Fatal(err)
	}
	var st HarnessState
	step := HarnessStep[int](context.Background(), detector,
		func(*int) *HarnessState { return &st },
		HarnessStepOptions{Scope: harness.ProjectScopeDir(t.TempDir())})
	cmd := step.Init(new(int))
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Init")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg from Init, got %T", msg)
	}
	for _, c := range batch {
		if c == nil {
			continue
		}
		if dm, ok := c().(detectedMsg); ok {
			if len(dm.harnesses) == 0 {
				t.Error("expected non-empty project-scoped detections")
			}
			return
		}
	}
	t.Fatal("no detectedMsg produced for project scope")
}
