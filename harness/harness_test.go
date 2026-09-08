package harness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/sairaph/mcp-wizard/harness"
)

func TestResolveExecutable_returnsNonEmpty(t *testing.T) {
	path, err := harness.ResolveExecutable()
	if err != nil {
		t.Fatalf("ResolveExecutable: %v", err)
	}
	if path == "" {
		t.Fatal("ResolveExecutable returned empty string")
	}
}

func TestNew_withValidSpec_returnsDetector(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "my-server",
		Command: "/usr/bin/env",
		Args:    []string{"--help"},
		Env:     map[string]string{"KEY": "value"},
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if d.Name() != "my-server" {
		t.Fatalf("Name() = %q, want %q", d.Name(), "my-server")
	}
}

func TestNew_withEmptyName_returnsError(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "",
		Command: "/usr/bin/env",
	}
	_, err := harness.New(spec)
	if err == nil {
		t.Fatal("New with empty name should return error")
	}
}

func TestHarness_Selectable_trueWhenDetected(t *testing.T) {
	h := harness.Harness{State: harness.Detected}
	if !h.Selectable() {
		t.Fatal("Selectable should be true when State is Detected")
	}
}

func TestHarness_Selectable_trueWhenConfigured(t *testing.T) {
	h := harness.Harness{State: harness.NotDetected, Configured: true}
	if !h.Selectable() {
		t.Fatal("Selectable should be true when Configured is true")
	}
}

func TestHarness_Selectable_falseWhenUnavailableAndNotConfigured(t *testing.T) {
	h := harness.Harness{State: harness.Unavailable, Configured: false}
	if h.Selectable() {
		t.Fatal("Selectable should be false when Unavailable and not Configured")
	}
}

func TestHarness_StatusText(t *testing.T) {
	tests := []struct {
		h    harness.Harness
		want string
	}{
		{harness.Harness{State: harness.Detected, Configured: false}, "detected"},
		{harness.Harness{State: harness.Detected, Configured: true}, "configured"},
		{harness.Harness{State: harness.NotDetected}, "not installed"},
		{harness.Harness{State: harness.Unavailable, Reason: "permission denied"}, "permission denied"},
	}
	for _, tt := range tests {
		got := tt.h.StatusText()
		if got != tt.want {
			t.Errorf("StatusText() = %q, want %q", got, tt.want)
		}
	}
}

func TestSupported_returnsNonEmpty(t *testing.T) {
	supported := harness.Supported()
	if len(supported) == 0 {
		t.Fatal("Supported returned empty list")
	}
	for _, s := range supported {
		if s.ID == "" {
			t.Fatal("Supported entry has empty ID")
		}
		if s.Name == "" {
			t.Fatal("Supported entry has empty Name")
		}
	}
}

func TestDetect_returnsResults(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	results := d.Detect(context.Background())
	if len(results) == 0 {
		t.Fatal("Detect returned empty results")
	}
	for _, r := range results {
		if r.ID == "" {
			t.Error("Detect result has empty ID")
		}
		if r.Name == "" {
			t.Error("Detect result has empty Name")
		}
	}
}

func TestDetect_configuredFlag(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	results := d.Detect(context.Background())
	for _, r := range results {
		if r.State == harness.Detected && !r.Configured {
			t.Logf("harness %s is detected but not configured (expected on first run)", r.Name)
		}
	}
}

func TestApply_withNoIDs(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	results := d.Apply(context.Background(), nil, harness.Present, harness.ConflictError)
	if len(results) != 0 {
		t.Fatalf("Apply with nil ids returned %d results, want 0", len(results))
	}
}

func TestPlanResults_withNoIDs(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	changes, err := d.PlanResults(context.Background(), nil, harness.Present, harness.ConflictError)
	if err != nil {
		t.Fatalf("PlanResults: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("PlanResults with nil ids returned %d changes, want 0", len(changes))
	}
}

func TestDetect_sortedOrder(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	results := d.Detect(context.Background())
	for i := 1; i < len(results); i++ {
		prev := results[i-1]
		cur := results[i]
		prevActive := prev.State == harness.Detected || prev.Configured
		curActive := cur.State == harness.Detected || cur.Configured
		if !prevActive && curActive {
			t.Fatal("Detect should sort detected/configured before others")
		}
	}
}

func TestNew_nilEnv(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	_, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New with nil env: %v", err)
	}
}

func TestResolveExecutable_returnsValidPath(t *testing.T) {
	path, err := harness.ResolveExecutable()
	if err != nil {
		t.Fatalf("ResolveExecutable: %v", err)
	}
	if !strings.HasSuffix(path, ".test") && !strings.Contains(path, "/") {
		t.Fatalf("ResolveExecutable returned unexpected path: %q", path)
	}
}

func TestProjectScopeDir_setsModeAndDir(t *testing.T) {
	s := harness.ProjectScopeDir("/some/dir")
	if s.Mode != harness.ScopeProject {
		t.Fatalf("Mode = %q, want %q", s.Mode, harness.ScopeProject)
	}
	if s.Dir != "/some/dir" {
		t.Fatalf("Dir = %q, want %q", s.Dir, "/some/dir")
	}
}

func TestScope_IsProject(t *testing.T) {
	if !harness.ProjectScopeDir("/tmp").IsProject() {
		t.Fatal("ProjectScopeDir(\"/tmp\").IsProject() = false, want true")
	}
	if (harness.Scope{}).IsProject() {
		t.Fatal("zero-value Scope.IsProject() = true, want false")
	}
	if (harness.Scope{Mode: harness.ScopeGlobal}).IsProject() {
		t.Fatal("ScopeGlobal.IsProject() = true, want false")
	}
}

func TestDetectIn_zeroScopeMatchesExplicitGlobal(t *testing.T) {
	d, err := harness.New(harness.ServerSpec{Name: "t", Command: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	want := d.DetectIn(ctx, harness.Scope{})
	got := d.DetectIn(ctx, harness.Scope{Mode: harness.ScopeGlobal})
	if len(want) != len(got) {
		t.Fatalf("zero scope returned %d results, explicit global %d", len(want), len(got))
	}
	for i := range want {
		if want[i].ID != got[i].ID || want[i].Configured != got[i].Configured || want[i].State != got[i].State {
			t.Fatalf("result %d differs: %+v vs %+v", i, want[i], got[i])
		}
	}
	// Results are ordered selectable-first, then by name.
	seenUnselectable := false
	for i, h := range want {
		if !h.Selectable() {
			seenUnselectable = true
		} else if seenUnselectable {
			t.Fatalf("selectable harness %s at index %d after an unselectable one", h.ID, i)
		}
	}
}
func TestDetectIn_projectScopeDoesNotPanic(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DetectIn with project scope panicked: %v", r)
		}
	}()
	results := d.DetectIn(context.Background(), harness.ProjectScopeDir("/tmp"))
	if results == nil {
		t.Fatal("DetectIn with project scope returned nil slice")
	}
}

func TestSupported_populatesProjectField(t *testing.T) {
	supported := harness.Supported()
	var withProject, withoutProject int
	for _, s := range supported {
		if s.Project != nil {
			withProject++
			if s.Project.Path == "" {
				t.Errorf("harness %s has Project but empty Path", s.ID)
			}
		} else {
			withoutProject++
		}
	}
	if withProject == 0 {
		t.Fatal("no supported harness advertises project scope; expected at least one")
	}
	if withoutProject == 0 {
		t.Fatal("no supported harness lacks project scope; expected at least one to exercise nil conversion")
	}
}

func TestConvertProjectScope_nilPropagatesThroughSupported(t *testing.T) {
	supported := harness.Supported()
	var sawNil bool
	for _, s := range supported {
		if s.Project == nil {
			sawNil = true
			break
		}
	}
	if !sawNil {
		t.Fatal("expected at least one Supported entry with nil Project (convertProjectScope(nil) safety)")
	}
}

func TestConvertProjectScope_nilPropagatesThroughDetect(t *testing.T) {
	spec := harness.ServerSpec{
		Name:    "test-server",
		Command: "/usr/bin/env",
	}
	d, err := harness.New(spec)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	results := d.Detect(context.Background())
	var sawNil bool
	for _, r := range results {
		if r.Project == nil {
			sawNil = true
			break
		}
	}
	if !sawNil {
		t.Fatal("expected at least one Detect result with nil Project (convertProjectScope(nil) safety)")
	}
}

func TestProjectScopeSelectableAndInstalled(t *testing.T) {
	d, err := harness.New(harness.ServerSpec{Name: "t", Command: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	results := d.DetectIn(context.Background(), harness.ProjectScopeDir(dir))
	if len(results) == 0 {
		t.Fatal("no results")
	}
	global := map[harness.ID]bool{}
	for _, g := range d.Detect(context.Background()) {
		global[g.ID] = g.State == harness.Detected
	}
	sawSelectable := false
	for _, h := range results {
		if h.ScopeMode != harness.ScopeProject {
			t.Fatalf("%s: ScopeMode = %q, want project", h.ID, h.ScopeMode)
		}
		if h.Installed != global[h.ID] {
			t.Fatalf("%s: Installed=%v but global detection says %v", h.ID, h.Installed, global[h.ID])
		}
		switch h.State {
		case harness.Unavailable:
			if h.Selectable() {
				t.Fatalf("%s is unavailable in project scope but selectable", h.ID)
			}
		default:
			// A fresh project has no config files, yet the harness must be
			// selectable so the file can be created.
			if !h.Selectable() {
				t.Fatalf("%s: state %q should be selectable in project scope", h.ID, h.State)
			}
			sawSelectable = true
			if h.Configured {
				t.Fatalf("%s reported configured in an empty project", h.ID)
			}
			if h.Installed && h.StatusText() != "installed" {
				t.Fatalf("%s: status %q, want installed", h.ID, h.StatusText())
			}
		}
	}
	if !sawSelectable {
		t.Fatal("expected at least one selectable harness in project scope")
	}
}

func TestGlobalScopeInstalledEqualsDetected(t *testing.T) {
	d, err := harness.New(harness.ServerSpec{Name: "t", Command: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range d.Detect(context.Background()) {
		if h.Installed != (h.State == harness.Detected) {
			t.Fatalf("%s: Installed=%v State=%s", h.ID, h.Installed, h.State)
		}
	}
}
