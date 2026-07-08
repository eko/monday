package helper

import (
	"runtime/debug"

	"github.com/eko/monday/pkg/ui"
)

// RecoverAndLog recovers from a panic in a background goroutine and reports it
// to the given view instead of crashing the whole application. It must be
// deferred at the beginning of the goroutine
func RecoverAndLog(
	view ui.View,
	component string,
) {
	if r := recover(); r != nil {
		view.Writef(
			"❌  Recovered from a panic in %s: %v\n%s\n",
			component,
			r,
			string(debug.Stack()),
		)
	}
}
