// Package doctor provides health checks for MCP server installations.
package doctor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sairaph/mcp-wizard/update"
)

// Status is the result of one health check.
type Status string

const (
	OK   Status = "ok"
	Warn Status = "warn"
	Fail Status = "fail"
)

// Result is the outcome of one health check.
type Result struct {
	Name   string
	Status Status
	Detail string
}

// Check is one diagnostic probe.
type Check interface {
	Name() string
	Run(ctx context.Context) Result
}

// Runner executes checks and formats results.
type Runner struct {
	checks []Check
}

// New creates a Runner with the given checks.
func New(checks ...Check) *Runner {
	return &Runner{checks: checks}
}

// Run executes all checks and writes formatted output to w.
// Returns 0 if no Fail results, 1 otherwise.
func (r *Runner) Run(ctx context.Context, w io.Writer) int {
	if r == nil || w == nil {
		return 1
	}
	exitCode := 0
	for _, check := range r.checks {
		if err := ctx.Err(); err != nil {
			fmt.Fprintf(w, "  [%s] %s\n", "fail", "cancelled")
			return 1
		}
		result := check.Run(ctx)
		var statusStr string
		switch result.Status {
		case OK:
			statusStr = fmt.Sprintf("[%s]", OK)
		case Warn:
			statusStr = fmt.Sprintf("[%s]", Warn)
		case Fail:
			statusStr = fmt.Sprintf("[%s]", Fail)
			exitCode = 1
		}
		fmt.Fprintf(w, "  %-6s %s\n", statusStr, result.Name)
		if result.Detail != "" {
			fmt.Fprintf(w, "         %s\n", result.Detail)
		}
	}
	return exitCode
}

// Add appends checks to the runner.
func (r *Runner) Add(checks ...Check) {
	r.checks = append(r.checks, checks...)
}

// --- Built-in checks ---

// ExecutableCheck verifies the binary is resolvable and executable.
type ExecutableCheck struct {
	Executable string // path to check (empty = use os.Executable)
}

func (c ExecutableCheck) Name() string { return "Executable" }

func (c ExecutableCheck) Run(_ context.Context) Result {
	path := c.Executable
	if path == "" {
		var err error
		path, err = os.Executable()
		if err != nil {
			return Result{Name: c.Name(), Status: Fail, Detail: fmt.Sprintf("cannot determine executable: %v", err)}
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return Result{Name: c.Name(), Status: Fail, Detail: fmt.Sprintf("cannot stat %s: %v", path, err)}
	}
	if info.Mode()&0111 == 0 {
		return Result{Name: c.Name(), Status: Fail, Detail: fmt.Sprintf("%s is not executable", path)}
	}
	return Result{Name: c.Name(), Status: OK, Detail: path}
}

// PathCheck verifies the install directory is on $PATH.
type PathCheck struct {
	Dir string // expected install directory
}

func (c PathCheck) Name() string { return "PATH" }

func (c PathCheck) Run(_ context.Context) Result {
	if c.Dir == "" {
		return Result{Name: c.Name(), Status: Warn, Detail: "no install directory configured"}
	}
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if filepath.Clean(dir) == filepath.Clean(c.Dir) {
			return Result{Name: c.Name(), Status: OK, Detail: c.Dir}
		}
	}
	return Result{Name: c.Name(), Status: Warn, Detail: fmt.Sprintf("%s is not on PATH", c.Dir)}
}

// ConfigExistsCheck verifies a configuration file exists.
type ConfigExistsCheck struct {
	Path string
}

func (c ConfigExistsCheck) Name() string { return "Config" }

func (c ConfigExistsCheck) Run(_ context.Context) Result {
	if c.Path == "" {
		return Result{Name: c.Name(), Status: Warn, Detail: "no config path configured"}
	}
	info, err := os.Stat(c.Path)
	if err != nil {
		return Result{Name: c.Name(), Status: Fail, Detail: fmt.Sprintf("%s: %v", c.Path, err)}
	}
	return Result{Name: c.Name(), Status: OK, Detail: fmt.Sprintf("%s (%d bytes)", c.Path, info.Size())}
}

// UpdateCheck verifies an update is available.
type UpdateCheck struct {
	Opts update.Options
}

func (c UpdateCheck) Name() string { return "Update" }

func (c UpdateCheck) Run(ctx context.Context) Result {
	if c.Opts.Owner == "" || c.Opts.Repo == "" || c.Opts.CurrentVersion == "" {
		return Result{Name: c.Name(), Status: Warn, Detail: "update check not configured"}
	}
	latest, available, err := update.Check(ctx, c.Opts)
	if err != nil {
		return Result{Name: c.Name(), Status: Warn, Detail: fmt.Sprintf("check failed: %v", err)}
	}
	if !available {
		if latest != "" {
			return Result{Name: c.Name(), Status: OK, Detail: fmt.Sprintf("up to date (%s)", c.Opts.CurrentVersion)}
		}
		return Result{Name: c.Name(), Status: Warn, Detail: "could not check for updates"}
	}
	return Result{Name: c.Name(), Status: Warn, Detail: fmt.Sprintf("update available: %s (current: %s)", latest, c.Opts.CurrentVersion)}
}
