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

// Detect probes all harnesses in global scope and returns Harness values with
// the Configured flag set. It is DetectIn with a zero-value Scope.
func (d *Detector) Detect(ctx context.Context) []Harness {
	return d.DetectIn(ctx, Scope{})
}

// DetectIn probes all harnesses for the supplied scope and returns Harness
// values with the Configured and Installed flags set: Configured by running a
// scope-aware Plan and checking for a no-op change, Installed from global
// detection. Results are sorted with relevant harnesses (installed,
// configured, or with a config present) first, then by name.
func (d *Detector) DetectIn(ctx context.Context, scope Scope) []Harness {
	if d == nil || d.installer == nil {
		return nil
	}
	dhScope := toDetectHarnessScope(scope)
	raw, err := d.installer.DetectScoped(ctx, dhScope)
	if err != nil {
		return nil
	}
	results := make([]Harness, 0, len(raw))
	ids := make([]detectharness.ID, 0, len(raw))
	for _, r := range raw {
		results = append(results, convertDetection(r))
		ids = append(ids, r.ID)
	}

	// Installed comes from global detection: in project scope the scoped
	// State only says whether the project file exists.
	if scope.IsProject() {
		installed := make(map[ID]bool)
		for _, g := range d.installer.Detect(ctx) {
			if g.State == detectharness.Detected {
				installed[ID(g.ID)] = true
			}
		}
		for i := range results {
			results[i].Installed = installed[results[i].ID]
		}
	} else {
		for i := range results {
			results[i].Installed = results[i].State == Detected
		}
	}

	// A harness is configured when planning "present" for it is a no-op.
	if len(ids) > 0 {
		plan := d.installer.Plan(ctx, ids, detectharness.Present, detectharness.PlanOptions{ConflictPolicy: detectharness.ConflictReplace, Scope: dhScope})
		configured := make(map[ID]bool)
		if plan != nil {
			for _, ch := range plan.Changes() {
				if ch.State == detectharness.ChangeNoop {
					configured[ID(ch.HarnessID)] = true
				}
			}
		}
		for i := range results {
			results[i].Configured = configured[results[i].ID]
		}
	}

	sort.Slice(results, func(i, j int) bool {
		ia := results[i].Relevant()
		ja := results[j].Relevant()
		if ia != ja {
			return ia
		}
		return results[i].Name < results[j].Name
	})
	return results
}

// Apply registers or unregisters the server in the given harnesses in global
// scope. It is ApplyIn with a zero-value Scope.
func (d *Detector) Apply(ctx context.Context, ids []ID, desired DesiredState, policy ConflictPolicy) []Result {
	return d.ApplyIn(ctx, Scope{}, ids, desired, policy)
}

// ApplyIn registers or unregisters the server in the given harnesses for the
// supplied scope.
func (d *Detector) ApplyIn(ctx context.Context, scope Scope, ids []ID, desired DesiredState, policy ConflictPolicy) []Result {
	if d == nil || d.installer == nil {
		return nil
	}
	plan := d.installer.Plan(ctx, toDetectHarnessIDs(ids), toDetectHarnessDesired(desired), detectharness.PlanOptions{ConflictPolicy: toDetectHarnessPolicy(policy), Scope: toDetectHarnessScope(scope)})
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

// PlanResults computes global-scope changes without writing, for dry-run
// preview. It is PlanResultsIn with a zero-value Scope.
func (d *Detector) PlanResults(ctx context.Context, ids []ID, desired DesiredState, policy ConflictPolicy) ([]Change, error) {
	return d.PlanResultsIn(ctx, Scope{}, ids, desired, policy)
}

// PlanResultsIn computes changes for the supplied scope without writing, for
// dry-run preview.
func (d *Detector) PlanResultsIn(ctx context.Context, scope Scope, ids []ID, desired DesiredState, policy ConflictPolicy) ([]Change, error) {
	if d == nil || d.installer == nil {
		return nil, nil
	}
	plan := d.installer.Plan(ctx, toDetectHarnessIDs(ids), toDetectHarnessDesired(desired), detectharness.PlanOptions{ConflictPolicy: toDetectHarnessPolicy(policy), Scope: toDetectHarnessScope(scope)})
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

func toDetectHarnessScope(scope Scope) detectharness.Scope {
	return detectharness.Scope{Mode: detectharness.ScopeMode(scope.Mode), Dir: scope.Dir}
}

func toDetectHarnessIDs(ids []ID) []detectharness.ID {
	out := make([]detectharness.ID, len(ids))
	for i, id := range ids {
		out[i] = detectharness.ID(id)
	}
	return out
}

func toDetectHarnessPolicy(policy ConflictPolicy) detectharness.ConflictPolicy {
	if policy == ConflictReplace {
		return detectharness.ConflictReplace
	}
	return detectharness.ConflictError
}

func toDetectHarnessDesired(desired DesiredState) detectharness.DesiredState {
	if desired == Absent {
		return detectharness.Absent
	}
	return detectharness.Present
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
