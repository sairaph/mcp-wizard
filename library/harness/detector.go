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

	// Gather IDs of already-configured harnesses by running a Plan.
	configured := make(map[ID]bool)
	var ids []detectharness.ID
	for _, r := range raw {
		ids = append(ids, r.ID)
	}
	if len(ids) > 0 {
		plan := d.installer.Plan(ctx, ids, detectharness.Present, detectharness.PlanOptions{ConflictPolicy: detectharness.ConflictReplace})
		for _, ch := range plan.Changes() {
			if ch.State == detectharness.ChangeNoop {
				configured[ID(ch.HarnessID)] = true
			}
		}
	}

	for _, r := range raw {
		h := convertDetection(r)
		h.Configured = configured[ID(r.ID)]
		results = append(results, h)
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

// Supported returns the built-in harness catalog.
func Supported() []struct{ ID, Name, ReloadHint string } {
	raw := detectharness.Supported()
	result := make([]struct{ ID, Name, ReloadHint string }, len(raw))
	for i, h := range raw {
		result[i] = struct{ ID, Name, ReloadHint string }{ID: string(h.ID), Name: h.Name, ReloadHint: h.ReloadHint}
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
