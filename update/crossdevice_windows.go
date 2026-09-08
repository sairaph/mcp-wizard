//go:build windows

package update

import (
	"errors"
	"syscall"
)

// errorNotSameDevice is ERROR_NOT_SAME_DEVICE, which MoveFileEx returns when
// source and target are on different volumes. syscall.EXDEV on Windows is a
// synthetic errno that the OS never produces, so it cannot be used here.
const errorNotSameDevice syscall.Errno = 17

// isCrossDevice reports whether a rename failed because source and target
// are on different volumes.
func isCrossDevice(err error) bool {
	return errors.Is(err, errorNotSameDevice)
}
