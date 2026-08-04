// Package lock provides file-lock-based daemon lifecycle management.
// It handles lock acquisition, PID tracking, autostart, and graceful shutdown.
package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// ErrAlreadyRunning is returned when the daemon lock is held by another process.
var ErrAlreadyRunning = errors.New("daemon is already running")

// Options controls daemon behaviour.
type Options struct {
	// LockFile is the path to the file lock (e.g. ~/.app/lock).
	LockFile string

	// PIDFile is the path to the PID file (e.g. ~/.app/daemon.pid).
	PIDFile string

	// LogFile is the path for daemon stdout/stderr. Empty means discard.
	LogFile string
}

// Instance is a running daemon instance.
type Instance struct {
	opts   Options
	lock   *flock.Flock
	locked bool
}

// Open acquires the daemon lock and starts the instance.
// Returns ErrAlreadyRunning if the lock is held by another process.
func Open(opts Options) (*Instance, error) {
	if err := os.MkdirAll(filepath.Dir(opts.LockFile), 0700); err != nil {
		return nil, fmt.Errorf("create daemon directory: %w", err)
	}

	lock := flock.New(opts.LockFile)
	locked, err := lock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquire daemon lock: %w", err)
	}
	if !locked {
		return nil, ErrAlreadyRunning
	}

	inst := &Instance{
		opts:   opts,
		lock:   lock,
		locked: true,
	}

	if opts.PIDFile != "" {
		if err := os.WriteFile(opts.PIDFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0600); err != nil {
			inst.Close()
			return nil, fmt.Errorf("write PID file: %w", err)
		}
	}

	return inst, nil
}

// Close releases the daemon lock and cleans up.
func (inst *Instance) Close() {
	if inst.locked {
		inst.lock.Unlock()
		inst.locked = false
	}
	if inst.opts.PIDFile != "" {
		os.Remove(inst.opts.PIDFile)
	}
}

// StaleAfter is the duration after which a lock file is considered stale.
// Used by tests. IsRunning uses flock.TryLock instead of timestamp checks.
const StaleAfter = 5 * time.Minute

// IsRunning reports whether a daemon is running by trying to acquire the lock.
// If the lock cannot be acquired, the daemon is considered running.
func IsRunning(lockFile string) bool {
	lock := flock.New(lockFile)
	locked, err := lock.TryLock()
	if err != nil {
		return false
	}
	if locked {
		lock.Unlock()
		return false // lock was free — daemon is not running
	}
	return true // lock is held — daemon is running
}

// Stop stops a running daemon by sending a signal to the PID in the PID file.
// Returns nil if the daemon was stopped, or an error if it couldn't be stopped.
func Stop(pidFile, lockFile string) error {
	if pidFile == "" {
		return fmt.Errorf("daemon: pidFile is required to stop the daemon")
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("daemon: read pid file: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return fmt.Errorf("daemon: invalid pid in %s: %q", pidFile, strings.TrimSpace(string(data)))
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("daemon: find process %d: %w", pid, err)
	}
	if proc == nil {
		return fmt.Errorf("daemon: process %d not found", pid)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("daemon: signal process %d: %w", pid, err)
	}
	return nil
}
