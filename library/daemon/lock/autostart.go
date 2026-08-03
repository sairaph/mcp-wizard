package lock

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EnsureRunning starts the daemon if it is not already running.
// executable is the path to the binary. args are passed to the daemon subcommand.
// Returns true if the daemon is running (was already running or was started).
func EnsureRunning(executable string, args []string, lockFile string) bool {
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

	// Poll for the daemon to acquire the lock.
	for i := 0; i < 10; i++ {
		if IsRunning(lockFile) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return IsRunning(lockFile)
}
