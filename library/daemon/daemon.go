// Package daemon provides lifecycle management for background daemon processes.
// It handles lock acquisition, PID tracking, autostart, and graceful shutdown.
package daemon

import (
	"context"
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
	cancel context.CancelFunc
	done   chan struct{}
}

// Open acquires the daemon lock and starts the instance.
// Returns ErrAlreadyRunning if the lock is held by another process.
func Open(ctx context.Context, opts Options) (*Instance, error) {
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

	ctx, cancel := context.WithCancel(ctx)
	inst := &Instance{
		opts:   opts,
		lock:   lock,
		locked: true,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	if opts.PIDFile != "" {
		if err := os.WriteFile(opts.PIDFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644); err != nil {
			inst.Close()
			return nil, fmt.Errorf("write PID file: %w", err)
		}
	}

	return inst, nil
}

// Serve runs the daemon loop. The function should block until ctx is done.
// Returns the error from serve or nil on clean shutdown.
func (inst *Instance) Serve(ctx context.Context, serve func(context.Context) error) error {
	defer close(inst.done)
	return serve(ctx)
}

// Close releases the daemon lock and cleans up.
func (inst *Instance) Close() {
	if inst.cancel != nil {
		inst.cancel()
	}
	if inst.locked {
		inst.lock.Unlock()
		inst.locked = false
	}
	if inst.opts.PIDFile != "" {
		os.Remove(inst.opts.PIDFile)
	}
}

// Done returns a channel that is closed when Serve returns.
func (inst *Instance) Done() <-chan struct{} {
	return inst.done
}

// StaleAfter is the duration after which a lock file is considered stale.
const StaleAfter = 5 * time.Minute

// IsRunning reports whether a daemon is running by checking the lock file.
// A stale lock (older than StaleAfter) is considered not running.
func IsRunning(lockFile string) bool {
	info, err := os.Stat(lockFile)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) <= StaleAfter
}

// Stop stops a running daemon by sending a signal to the PID in the PID file.
// Returns nil if the daemon was stopped, or an error if it couldn't be stopped.
func Stop(pidFile, lockFile string) error {
	if pidFile != "" {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil && pid > 0 {
				proc, err := os.FindProcess(pid)
				if err == nil {
					if err := proc.Signal(os.Interrupt); err == nil {
						return nil
					}
				}
			}
		}
	}
	if lockFile != "" {
		os.Remove(lockFile)
	}
	return fmt.Errorf("daemon: could not stop process")
}
