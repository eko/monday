//go:build !darwin && !linux && !windows

package txeh

import (
	"errors"
	"runtime"
)

// flushDNSCachePlatform returns an error on unsupported platforms.
func flushDNSCachePlatform() error {
	return &FlushError{
		Platform: runtime.GOOS,
		Command:  "",
		Err:      errors.New("unsupported platform"),
	}
}
