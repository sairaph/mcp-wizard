//go:build !windows

package daemon

import "syscall"

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
