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

// ServerSpec is the library-owned server definition.
type ServerSpec struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// Harness is a detection result augmented with the Configured flag.
type Harness struct {
	ID          ID             `json:"id"`
	Name        string         `json:"name"`
	State       DetectionState `json:"state"`
	Evidence    []string       `json:"evidence,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	ConfigPath  string         `json:"configPath,omitempty"`
	ConfigError string         `json:"configError,omitempty"`
	ReloadHint  string         `json:"reloadHint"`
	Configured  bool           `json:"configured"`
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
