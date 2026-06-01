//go:build windows

package txeh

import "runtime"

// flushDNSCachePlatform flushes the DNS cache on Windows.
//
// Command: ipconfig /flushdns
// Source: https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/ipconfig
// Present in all supported Windows versions (XP through current).
// Ships with Windows. No external dependencies.
func flushDNSCachePlatform() error {
	cmd := execCommandFunc("ipconfig", "/flushdns") // #nosec G204 -- hardcoded command, not user input
	if err := cmd.Run(); err != nil {
		return &FlushError{
			Platform: runtime.GOOS,
			Command:  joinArgs("ipconfig", []string{"/flushdns"}),
			Err:      err,
		}
	}
	return nil
}
