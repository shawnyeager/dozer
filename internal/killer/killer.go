package killer

import (
	"errors"
	"fmt"
	"syscall"
)

// Kill sends sig to pid. Returns a wrapped error that preserves syscall errno
// values like EPERM and ESRCH so callers can errors.Is against them.
func Kill(pid int32, sig syscall.Signal) error {
	if pid <= 1 {
		return errors.New("refusing to signal invalid pid")
	}
	if err := syscall.Kill(int(pid), sig); err != nil {
		return fmt.Errorf("kill pid %d: %w", pid, err)
	}
	return nil
}
