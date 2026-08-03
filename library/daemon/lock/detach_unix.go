//go:build !windows

package lock

import "syscall"

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
