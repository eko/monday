//go:build linux

package txeh

import (
	"errors"
	"os/exec"
	"runtime"
)

// ErrNoResolver is returned when no supported DNS resolver is found on the system.
// If your system does not cache DNS locally (e.g., no systemd-resolved, dnsmasq, or
// unbound), hosts file changes take effect immediately without flushing.
var ErrNoResolver = errors.New(
	"systemd-resolved not detected (tried resolvectl, systemd-resolve). " +
		"If your system does not cache DNS locally, hosts file changes take effect immediately without flushing",
)

// flushDNSCachePlatform flushes the DNS cache on Linux.
//
// Commands:
//   - resolvectl flush-caches: systemd-resolved 239+ (2018).
//     Source: https://man7.org/linux/man-pages/man1/resolvectl.1.html
//   - systemd-resolve --flush-caches: pre-239 name for the same operation.
//
// Tries resolvectl first, then falls back to systemd-resolve.
// Returns ErrNoResolver (wrapped in FlushError) if neither is available.
//
// Other resolvers (dnsmasq, unbound, nscd) are intentionally not supported here
// because flushing them reliably depends on per-site configuration (socket paths,
// service names, etc.) that we can't safely assume.
func flushDNSCachePlatform() error {
	// Try resolvectl first (systemd 239+).
	if _, err := exec.LookPath("resolvectl"); err == nil {
		cmd := execCommandFunc("resolvectl", "flush-caches") // #nosec G204 -- hardcoded command, not user input
		if err := cmd.Run(); err != nil {
			return &FlushError{
				Platform: runtime.GOOS,
				Command:  joinArgs("resolvectl", []string{"flush-caches"}),
				Err:      err,
			}
		}
		return nil
	}

	// Fallback to systemd-resolve (older systemd).
	if _, err := exec.LookPath("systemd-resolve"); err == nil {
		cmd := execCommandFunc("systemd-resolve", "--flush-caches") // #nosec G204 -- hardcoded command, not user input
		if err := cmd.Run(); err != nil {
			return &FlushError{
				Platform: runtime.GOOS,
				Command:  joinArgs("systemd-resolve", []string{"--flush-caches"}),
				Err:      err,
			}
		}
		return nil
	}

	return &FlushError{
		Platform: runtime.GOOS,
		Command:  "",
		Err:      ErrNoResolver,
	}
}
