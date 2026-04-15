package proc

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
)

// Comm returns the short command name from /proc/<pid>/comm, or "" if unreadable.
func Comm(pid int32) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Cmdline returns the full argv of the process, space-joined, or "" if unreadable.
func Cmdline(pid int32) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	return strings.Join(parts, " ")
}

// Username resolves a UID to a username, falling back to the numeric id.
func Username(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return strconv.FormatUint(uint64(uid), 10)
	}
	return u.Username
}

// Exists reports whether /proc/<pid> still exists (process alive).
func Exists(pid int32) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}
