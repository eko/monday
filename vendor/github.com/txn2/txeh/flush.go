package txeh

import (
	"fmt"
	"os/exec"
	"strings"
)

// execCommandFunc is a variable that wraps exec.Command for testability.
var execCommandFunc = exec.Command

// ExecCommandFunc returns the current exec command function.
// This is intended for test code that needs to save and restore the original.
func ExecCommandFunc() func(string, ...string) *exec.Cmd {
	return execCommandFunc
}

// SetExecCommandFunc replaces the exec command function.
// This is intended for test code that needs to mock command execution.
func SetExecCommandFunc(fn func(string, ...string) *exec.Cmd) {
	execCommandFunc = fn
}

// FlushError represents a DNS cache flush failure with platform context.
type FlushError struct {
	Platform string
	Command  string
	Err      error
}

// Error returns a human-readable error message.
func (e *FlushError) Error() string {
	return fmt.Sprintf("flush DNS cache on %s (%s): %s", e.Platform, e.Command, e.Err)
}

// Unwrap returns the underlying error.
func (e *FlushError) Unwrap() error {
	return e.Err
}

// FlushDNSCache flushes the operating system's DNS cache.
//
// Platform commands (all are OS-provided utilities, not installable dependencies):
//   - macOS: dscacheutil -flushcache + killall -HUP mDNSResponder (ships with macOS)
//   - Linux: resolvectl flush-caches (systemd 239+) or systemd-resolve --flush-caches (older systemd).
//     Returns ErrNoResolver if neither binary is found, which typically means the system
//     does not cache DNS locally and hosts file changes take effect immediately.
//   - Windows: ipconfig /flushdns (ships with Windows)
//
// Returns a *FlushError on failure, or nil on unsupported platforms
// where no resolver is detected.
func FlushDNSCache() error {
	return flushDNSCachePlatform()
}

// joinArgs joins command arguments for error display.
func joinArgs(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}
