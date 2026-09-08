//go:build !windows

package update

import (
	"errors"
	"syscall"
)

// isCrossDevice reports whether a rename failed because source and target
// are on different filesystems.
func isCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}
