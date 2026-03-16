//go:build darwin

package txeh

import "runtime"

// flushDNSCachePlatform flushes the DNS cache on macOS.
//
// Commands:
//   - dscacheutil -flushcache: flushes the Directory Service cache.
//     Source: man dscacheutil (local man page, -flushcache flag).
//     Present since macOS 10.5 Leopard (2007).
//   - killall -HUP mDNSResponder: restarts the mDNS responder to clear its cache.
//     Standard Apple convention, widely documented.
//
// Both commands together are the standard recommendation from macOS 10.15 Catalina (2019)
// through current releases. On 10.12-10.14, only the killall was strictly needed, but
// running dscacheutil there is harmless (flushes the Directory Service cache, a no-op
// if DNS isn't cached there).
//
// Both binaries ship with macOS. No external dependencies.
//
// The killall step is non-fatal since mDNSResponder may not be running.
func flushDNSCachePlatform() error {
	cmd := execCommandFunc("dscacheutil", "-flushcache") // #nosec G204 -- hardcoded command, not user input
	if err := cmd.Run(); err != nil {
		return &FlushError{
			Platform: runtime.GOOS,
			Command:  joinArgs("dscacheutil", []string{"-flushcache"}),
			Err:      err,
		}
	}

	// Attempt to restart mDNSResponder. This may fail if the process
	// is not running, which is not an error condition.
	cmd = execCommandFunc("killall", "-HUP", "mDNSResponder") // #nosec G204 -- hardcoded command, not user input
	_ = cmd.Run()

	return nil
}
