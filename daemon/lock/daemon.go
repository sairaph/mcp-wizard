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

	"github.com/gofrs/flock"
)

// ErrAlreadyRunning is returned when the daemon lock is held by another process.
var ErrAlreadyRunning = errors.New("daemon is already running")

// ErrNotRunning is returned by Stop when no process holds the daemon lock.
var ErrNotRunning = errors.New("daemon is not running")

// Options controls daemon behaviour.
type Options struct {
	// LockFile is the path to the file lock (e.g. ~/.app/lock).
	LockFile string

	// PIDFile is the path to the PID file (e.g. ~/.app/daemon.pid).
	PIDFile string

	// LogFile is the path EnsureRunning points the daemon's stdout/stderr
	// at (appended). Empty means discard.
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

// Close removes the PID file and releases the daemon lock, in that order so
// a successor that grabs the lock cannot have its fresh PID file deleted.
func (inst *Instance) Close() {
	if inst.opts.PIDFile != "" {
		os.Remove(inst.opts.PIDFile)
	}
	if inst.locked {
		inst.lock.Unlock()
		inst.locked = false
	}
}

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
		return false // lock was free - daemon is not running
	}
	return true // lock is held - daemon is running
}

// Stop asks a running daemon to shut down using the PID in the PID file. On
// Unix it sends SIGINT so the daemon can exit gracefully; on Windows, where
// processes cannot receive SIGINT, it terminates the process.
//
// When lockFile is given, Stop first checks that the lock is held; if not,
// the PID file is stale (the daemon crashed), it is removed and ErrNotRunning
// is returned, so an unrelated process that reused the PID is never signalled.
func Stop(pidFile, lockFile string) error {
	if pidFile == "" {
		return fmt.Errorf("daemon: pidFile is required to stop the daemon")
	}
	if lockFile != "" && !IsRunning(lockFile) {
		os.Remove(pidFile)
		return ErrNotRunning
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
	if err := stopProcess(proc); err != nil {
		return fmt.Errorf("daemon: stop process %d: %w", pid, err)
	}
	return nil
}
