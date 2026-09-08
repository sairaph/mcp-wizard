package lock

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EnsureRunning starts the daemon if it is not already running.
//
// executable is the path to the binary and args are passed to it (typically
// the daemon subcommand). opts.LockFile is polled to confirm the child
// acquired the lock; opts.LogFile, when set, receives the child's stdout and
// stderr (appended), otherwise they are discarded.
//
// Returns true if the daemon is running (was already running or was started).
func EnsureRunning(executable string, args []string, opts Options) bool {
	if IsRunning(opts.LockFile) {
		return true
	}

	cmd := exec.Command(executable, args...)
	cmd.SysProcAttr = detachAttr()
	cmd.Dir = filepath.Dir(executable)

	output, err := openLog(opts.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: open log: %v\n", err)
		return false
	}
	defer output.Close()
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: failed to start: %v\n", err)
		return false
	}
	// The child is detached; release our handle so it is not reported as a
	// zombie when it exits after we do.
	go cmd.Wait()

	// Poll for the daemon to acquire the lock.
	for i := 0; i < 20; i++ {
		if IsRunning(opts.LockFile) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return IsRunning(opts.LockFile)
}

// openLog opens the daemon log for appending, creating its directory. An
// empty path opens the null device.
func openLog(path string) (*os.File, error) {
	if path == "" {
		return os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
}
