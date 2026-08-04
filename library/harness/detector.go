package harness

import (
	"context"
	"sort"

	detectharness "github.com/sairaph/detect-harness"
)

// Detector manages harness detection and registration for one MCP server.
type Detector struct {
	name      string
	installer *detectharness.Installer
}

// New creates a Detector from a ServerSpec.
func New(spec ServerSpec) (*Detector, error) {
	server := detectharness.StdioServer{
		Name:    spec.Name,
		Command: spec.Command,
		Args:    append([]string(nil), spec.Args...),
		Env:     cloneEnv(spec.Env),
	}
	inst, err := detectharness.New(server)
	if err != nil {
		return nil, err
	}
	return &Detector{name: spec.Name, installer: inst}, nil
}

// Name returns the server name.
func (d *Detector) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// Detect probes all harnesses and returns Harness values with the Configured
// flag set (computed by running a Plan and checking for ChangeNoop).
func (d *Detector) Detect(ctx context.Context) []Harness {
	if d == nil || d.installer == nil {
		return nil
	}
	raw := d.installer.Detect(ctx)
	results := make([]Harness, 0, len(raw))
	for _, r := range raw {
		results = append(results, convertDetection(r))
	}

	// Gather IDs of already-configured harnesses by running a Plan.
	configured := make(map[ID]bool)
	var ids []detectharness.ID
	for _, r := range raw {
		ids = append(ids, r.ID)
	}
	if len(ids) > 0 {
		plan := d.installer.Plan(ctx, ids, detectharness.Present, detectharness.PlanOptions{ConflictPolicy: detectharness.ConflictReplace})
		if plan == nil {
			return results
		}
		for _, ch := range plan.Changes() {
			if ch.State == detectharness.ChangeNoop {
				configured[ID(ch.HarnessID)] = true
			}
		}
	}

	for i, r := range raw {
		results[i].Configured = configured[ID(r.ID)]
	}

	// Sort: detected first, then by name.
	sort.Slice(results, func(i, j int) bool {
		ia := results[i].State == Detected || results[i].Configured
		ja := results[j].State == Detected || results[j].Configured
		if ia != ja {
			return ia
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// DetectIn probes all harnesses for the supplied scope and returns Harness
// values with the Configured flag set, computed by running a scope-aware Plan.
// A zero-value scope is global scope and matches Detect exactly.
func (d *Detector) DetectIn(ctx context.Context, scope Scope) []Harness {
	if d == nil || d.installer == nil {
		return nil
	}
	dhScope := toDetectHarnessScope(scope)
	raw, err := detectharness.DetectHarnesses(ctx, detectharness.DetectOptions{Scope: dhScope})
	if err != nil {
		return nil
	}
	results := make([]Harness, 0, len(raw))
	for _, r := range raw {
		results = append(results, convertDetection(r))
	}

	// Gather IDs of already-configured harnesses by running a scope-aware Plan.
	configured := make(map[ID]bool)
	var ids []detectharness.ID
	for _, r := range raw {
		ids = append(ids, r.ID)
	}
	if len(ids) > 0 {
		plan := d.installer.Plan(ctx, ids, detectharness.Present, detectharness.PlanOptions{ConflictPolicy: detectharness.ConflictReplace, Scope: dhScope})
		if plan == nil {
			return results
		}
		for _, ch := range plan.Changes() {
			if ch.State == detectharness.ChangeNoop {
				configured[ID(ch.HarnessID)] = true
			}
		}
	}

	for i, r := range raw {
		results[i].Configured = configured[ID(r.ID)]
	}

	// Sort: detected first, then by name.
	sort.Slice(results, func(i, j int) bool {
		ia := results[i].State == Detected || results[i].Configured
		ja := results[j].State == Detected || results[j].Configured
		if ia != ja {
			return ia
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// Apply registers or unregisters the server in the given harnesses.
func (d *Detector) Apply(ctx context.Context, ids []ID, desired DesiredState, policy ConflictPolicy) []Result {
	if d == nil || d.installer == nil {
		return nil
	}
	dhIDs := make([]detectharness.ID, len(ids))
	for i, id := range ids {
		dhIDs[i] = detectharness.ID(id)
	}
	dhPolicy := detectharness.ConflictError
	if policy == ConflictReplace {
		dhPolicy = detectharness.ConflictReplace
	}
	dhDesired := detectharness.Present
	if desired == Absent {
		dhDesired = detectharness.Absent
	}

	plan := d.installer.Plan(ctx, dhIDs, dhDesired, detectharness.PlanOptions{ConflictPolicy: dhPolicy})
	if plan == nil {
		return nil
	}
	rawResults := d.installer.Apply(ctx, plan)

	results := make([]Result, len(rawResults))
	for i, r := range rawResults {
		results[i] = Result{
			HarnessID: ID(r.HarnessID),
			Name:      r.Name,
			Path:      r.Path,
			Desired:   desired,
			State:     ApplyState(r.State),
			Action:    r.Action,
			Reason:    r.Reason,
		}
	}
	return results
}

// ApplyIn registers or unregisters the server in the given harnesses for the
// supplied scope. A zero-value scope is global scope and matches Apply exactly.
func (d *Detector) ApplyIn(ctx context.Context, scope Scope, ids []ID, desired DesiredState, policy ConflictPolicy) []Result {
	if d == nil || d.installer == nil {
		return nil
	}
	dhScope := toDetectHarnessScope(scope)
	dhIDs := make([]detectharness.ID, len(ids))
	for i, id := range ids {
		dhIDs[i] = detectharness.ID(id)
	}
	dhPolicy := detectharness.ConflictError
	if policy == ConflictReplace {
		dhPolicy = detectharness.ConflictReplace
	}
	dhDesired := detectharness.Present
	if desired == Absent {
		dhDesired = detectharness.Absent
	}

	plan := d.installer.Plan(ctx, dhIDs, dhDesired, detectharness.PlanOptions{ConflictPolicy: dhPolicy, Scope: dhScope})
	if plan == nil {
		return nil
	}
	rawResults := d.installer.Apply(ctx, plan)

	results := make([]Result, len(rawResults))
	for i, r := range rawResults {
		results[i] = Result{
			HarnessID: ID(r.HarnessID),
			Name:      r.Name,
			Path:      r.Path,
			Desired:   desired,
			State:     ApplyState(r.State),
			Action:    r.Action,
			Reason:    r.Reason,
			ScopeMode: ScopeMode(r.Scope),
			ScopeDir:  r.ScopeDir,
		}
	}
	return results
}

// PlanResults computes changes without writing, for dry-run preview.
func (d *Detector) PlanResults(ctx context.Context, ids []ID, desired DesiredState, policy ConflictPolicy) ([]Change, error) {
	if d == nil || d.installer == nil {
		return nil, nil
	}
	dhIDs := make([]detectharness.ID, len(ids))
	for i, id := range ids {
		dhIDs[i] = detectharness.ID(id)
	}
	dhPolicy := detectharness.ConflictError
	if policy == ConflictReplace {
		dhPolicy = detectharness.ConflictReplace
	}
	dhDesired := detectharness.Present
	if desired == Absent {
		dhDesired = detectharness.Absent
	}

	plan := d.installer.Plan(ctx, dhIDs, dhDesired, detectharness.PlanOptions{ConflictPolicy: dhPolicy})
	if plan == nil {
		return nil, nil
	}
	changes := plan.Changes()

	result := make([]Change, len(changes))
	for i, ch := range changes {
		result[i] = Change{
			HarnessID: ID(ch.HarnessID),
			Name:      ch.Name,
			Path:      ch.Path,
			Desired:   desired,
			State:     ApplyState(ch.State),
			Action:    ch.Action,
			Reason:    ch.Reason,
		}
	}
	return result, nil
}

// PlanResultsIn computes changes for the supplied scope without writing, for
// dry-run preview. A zero-value scope is global scope and matches PlanResults.
func (d *Detector) PlanResultsIn(ctx context.Context, scope Scope, ids []ID, desired DesiredState, policy ConflictPolicy) ([]Change, error) {
	if d == nil || d.installer == nil {
		return nil, nil
	}
	dhScope := toDetectHarnessScope(scope)
	dhIDs := make([]detectharness.ID, len(ids))
	for i, id := range ids {
		dhIDs[i] = detectharness.ID(id)
	}
	dhPolicy := detectharness.ConflictError
	if policy == ConflictReplace {
		dhPolicy = detectharness.ConflictReplace
	}
	dhDesired := detectharness.Present
	if desired == Absent {
		dhDesired = detectharness.Absent
	}

	plan := d.installer.Plan(ctx, dhIDs, dhDesired, detectharness.PlanOptions{ConflictPolicy: dhPolicy, Scope: dhScope})
	if plan == nil {
		return nil, nil
	}
	changes := plan.Changes()

	result := make([]Change, len(changes))
	for i, ch := range changes {
		result[i] = Change{
			HarnessID: ID(ch.HarnessID),
			Name:      ch.Name,
			Path:      ch.Path,
			Desired:   desired,
			State:     ApplyState(ch.State),
			Action:    ch.Action,
			Reason:    ch.Reason,
			ScopeMode: ScopeMode(ch.Scope),
			ScopeDir:  ch.ScopeDir,
		}
	}
	return result, nil
}

// Supported returns the built-in harness catalog, including per-harness project
// scope metadata when available.
func Supported() []SupportedHarness {
	raw := detectharness.Supported()
	result := make([]SupportedHarness, len(raw))
	for i, h := range raw {
		result[i] = SupportedHarness{
			ID:         string(h.ID),
			Name:       h.Name,
			ReloadHint: h.ReloadHint,
			Project:    convertProjectScope(h.Project),
		}
	}
	return result
}

// convertDetection converts a detect-harness Detection to a library Harness.
func convertDetection(d detectharness.Detection) Harness {
	return Harness{
		ID:          ID(d.ID),
		Name:        d.Name,
		State:       DetectionState(d.State),
		Evidence:    append([]string(nil), d.Evidence...),
		Reason:      d.Reason,
		ConfigPath:  d.ConfigPath,
		ConfigError: d.ConfigError,
		ReloadHint:  d.ReloadHint,
		Project:     convertProjectScope(d.Project),
		ScopeMode:   ScopeMode(d.Scope),
		ScopeDir:    d.ScopeDir,
	}
}

// toDetectHarnessScope converts a library Scope to its detect-harness counterpart.
func toDetectHarnessScope(scope Scope) detectharness.Scope {
	return detectharness.Scope{Mode: detectharness.ScopeMode(scope.Mode), Dir: scope.Dir}
}

// convertProjectScope converts a detect-harness ProjectScope pointer to a
// library ProjectScopeInfo pointer, returning nil for nil input.
func convertProjectScope(p *detectharness.ProjectScope) *ProjectScopeInfo {
	if p == nil {
		return nil
	}
	return &ProjectScopeInfo{
		Path:       p.Path,
		ReloadHint: p.ReloadHint,
		Shareable:  p.Shareable,
		TrustGate:  p.TrustGate,
	}
}

func cloneEnv(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string, len(env))
	for k, v := range env {
		result[k] = v
	}
	return result
}
