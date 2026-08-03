package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// EnsureRunning starts the daemon if it is not already running.
// executable is the path to the binary. args are passed to the daemon subcommand.
// Returns true if the daemon is running (was already running or was started).
func EnsureRunning(ctx context.Context, executable string, args []string, lockFile string) bool {
	if IsRunning(lockFile) {
		return true
	}

	cmd := exec.Command(executable, args...)
	cmd.SysProcAttr = detachAttr()
	cmd.Dir = filepath.Dir(executable)

	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: failed to start: %v\n", err)
		return false
	}

	return IsRunning(lockFile)
}
