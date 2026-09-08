//go:build windows

package lock

import (
	"os"
	"syscall"
)

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// stopProcess terminates the process. Windows has no SIGINT delivery for a
// process outside our console, so a hard stop is the only portable option.
func stopProcess(proc *os.Process) error {
	return proc.Kill()
}
