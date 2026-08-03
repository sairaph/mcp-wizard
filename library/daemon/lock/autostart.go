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

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: open devnull: %v\n", err)
		return false
	}
	defer devNull.Close()
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: failed to start: %v\n", err)
		return false
	}

	// Poll for the daemon to acquire the lock.
	for i := 0; i < 20; i++ {
		if IsRunning(lockFile) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return IsRunning(lockFile)
}
