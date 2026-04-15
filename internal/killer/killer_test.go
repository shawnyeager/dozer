package killer

import (
	"os"
	"syscall"
	"testing"
)

func TestKillRefusesInvalidPIDs(t *testing.T) {
	for _, pid := range []int32{0, -1, 1} {
		if err := Kill(pid, syscall.SIGTERM); err == nil {
			t.Errorf("Kill(%d) should have failed", pid)
		}
	}
}

// Signal 0 is a permission probe: it checks whether we *could* signal the
// process without actually delivering anything. Sending it to ourselves should
// always succeed and proves the call plumbing works end-to-end.
func TestKillSelfWithSignalZero(t *testing.T) {
	pid := int32(os.Getpid())
	if err := Kill(pid, syscall.Signal(0)); err != nil {
		t.Fatalf("signal 0 to self should succeed: %v", err)
	}
}
