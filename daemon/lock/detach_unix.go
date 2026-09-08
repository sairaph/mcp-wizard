//go:build !windows

package lock

import (
	"os"
	"syscall"
)

// detachAttr starts the daemon in its own session so it outlives the
// caller's terminal and is not reached by its job-control signals.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}

// stopProcess asks the process to shut down gracefully.
func stopProcess(proc *os.Process) error {
	return proc.Signal(os.Interrupt)
}
