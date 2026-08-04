// Package harness wraps detect-harness at the library boundary with
// library-owned types. Consumers never import detect-harness directly.
package harness

import (
	"os"
	"path/filepath"
)

// ID is a stable identifier for a supported AI harness.
type ID string

// DesiredState describes whether a server should be present or absent.
type DesiredState string

const (
	Present DesiredState = "present"
	Absent  DesiredState = "absent"
)

// ConflictPolicy controls what happens when a same-name entry doesn't match.
type ConflictPolicy string

const (
	ConflictError   ConflictPolicy = "error"
	ConflictReplace ConflictPolicy = "replace"
)

// ScopeMode selects the configuration scope.
type ScopeMode string

const (
	// ScopeGlobal is the zero-value scope: system/user configuration. It
	// matches existing global behavior exactly.
	ScopeGlobal ScopeMode = ""
	// ScopeProject targets a directory-local (per-project) configuration.
	ScopeProject ScopeMode = "project"
)

// Scope selects where configuration is detected and applied. The zero value is
// global scope and preserves existing behavior.
type Scope struct {
	Mode ScopeMode
	// Dir is the project directory. It is required when Mode is ScopeProject
	// and ignored otherwise. Absolute paths are recommended; relative paths are
	// resolved against the process working directory.
	Dir string
}

// ProjectScopeDir returns a Scope that targets directory-local configuration in
// dir.
func ProjectScopeDir(dir string) Scope { return Scope{Mode: ScopeProject, Dir: dir} }

// IsProject reports whether the scope targets a per-project configuration.
func (s Scope) IsProject() bool { return s.Mode == ScopeProject }

// DetectionState distinguishes absence from an environment that couldn't be inspected.
type DetectionState string

const (
	Detected    DetectionState = "present"
	NotDetected DetectionState = "absent"
	Unavailable DetectionState = "unavailable"
)

// ApplyState is the outcome of applying one change.
type ApplyState string

const (
	Applied       ApplyState = "applied"
	ApplyNoop     ApplyState = "noop"
	ApplySkipped  ApplyState = "skipped"
	ApplyConflict ApplyState = "conflict"
	ApplyFailed   ApplyState = "failed"
)

// ProjectScopeInfo describes how a harness supports directory-local MCP
// configuration. It is informational metadata for consumers building install
// UX; the harness package never creates files unless asked to plan or apply.
type ProjectScopeInfo struct {
	Path       string `json:"path,omitempty"`
	ReloadHint string `json:"reloadHint,omitempty"`
	// Shareable indicates the file is intended to be committed to version control.
	Shareable bool `json:"shareable,omitempty"`
	// TrustGate indicates the harness gates project servers behind a trust or
	// approval dialog before they are loaded.
	TrustGate bool `json:"trustGate,omitempty"`
}

// SupportedHarness is one entry in the built-in harness catalog.
type SupportedHarness struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	ReloadHint string              `json:"reloadHint"`
	// Project describes project-scoped (directory-local) configuration support.
	// It is nil for harnesses that only support a global configuration.
	Project *ProjectScopeInfo `json:"project,omitempty"`
}

// ServerSpec is the library-owned server definition.
type ServerSpec struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// Harness is a detection result augmented with the Configured flag.
type Harness struct {
	ID          ID                 `json:"id"`
	Name        string             `json:"name"`
	State       DetectionState     `json:"state"`
	Evidence    []string           `json:"evidence,omitempty"`
	Reason      string             `json:"reason,omitempty"`
	ConfigPath  string             `json:"configPath,omitempty"`
	ConfigError string             `json:"configError,omitempty"`
	ReloadHint  string             `json:"reloadHint"`
	Configured  bool               `json:"configured"`
	// Project describes project-scoped (directory-local) configuration support.
	// It is nil for harnesses that only support a global configuration.
	Project   *ProjectScopeInfo `json:"project,omitempty"`
	ScopeMode ScopeMode         `json:"scopeMode,omitempty"`
	ScopeDir  string            `json:"scopeDir,omitempty"`
}

// Selectable reports whether a harness can be registered.
func (h Harness) Selectable() bool {
	return h.State == Detected || h.Configured
}

// StatusText returns a human-readable status line.
func (h Harness) StatusText() string {
	switch h.State {
	case Detected:
		if h.Configured {
			return "configured"
		}
		return "detected"
	case NotDetected:
		return "not installed"
	case Unavailable:
		return h.Reason
	default:
		return string(h.State)
	}
}

// Result reports the outcome for one harness.
type Result struct {
	HarnessID ID           `json:"harnessId"`
	Name      string       `json:"name"`
	Path      string       `json:"path,omitempty"`
	Desired   DesiredState `json:"desired"`
	State     ApplyState   `json:"state"`
	Action    string       `json:"action,omitempty"`
	Reason    string       `json:"reason,omitempty"`
	ScopeMode ScopeMode    `json:"scopeMode,omitempty"`
	ScopeDir  string       `json:"scopeDir,omitempty"`
}

// Change is a serializable description of one planned change.
type Change struct {
	HarnessID ID           `json:"harnessId"`
	Name      string       `json:"name"`
	Path      string       `json:"path,omitempty"`
	Desired   DesiredState `json:"desired"`
	State     ApplyState   `json:"state"`
	Action    string       `json:"action,omitempty"`
	Reason    string       `json:"reason,omitempty"`
	ScopeMode ScopeMode    `json:"scopeMode,omitempty"`
	ScopeDir  string       `json:"scopeDir,omitempty"`
}

// ResolveExecutable returns the absolute, symlink-resolved path to the
// calling binary.
func ResolveExecutable() (string, error) {
	exec, err := os.Executable()
	if err != nil {
		return "", err
	}
	exec, err = filepath.Abs(exec)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exec)
	if err != nil {
		return "", err
	}
	return resolved, nil
}
