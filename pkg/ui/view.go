package ui

import (
	"fmt"
	"strings"
	"sync"
)

// maxBufferLines is the maximum number of lines kept in memory per view
const maxBufferLines = 5000

type View interface {
	GetName() string
	Write(str string)
	Writef(str string, args ...interface{})
}

// view is a thread-safe buffered view: it accumulates lines that the terminal
// UI renders, or directly prints to stdout when the terminal UI is disabled
type view struct {
	name  string
	title string

	// stdout indicates the view writes directly to standard output (UI disabled)
	stdout bool

	mux     sync.Mutex
	lines   []string
	partial string
	version uint64

	// notify nudges the terminal UI that new content is available
	notify func()
}

// NewView returns a new buffered view instance, rendered by the terminal UI
func NewView(name, title string) *view {
	return &view{
		name:  name,
		title: title,
	}
}

// NewEmptyView returns a view that prints directly to standard output,
// used when the terminal UI is disabled
func NewEmptyView(name string) *view {
	return &view{
		name:   name,
		stdout: true,
	}
}

// GetName returns the name of the view
func (v *view) GetName() string {
	return v.name
}

// GetTitle returns the title of the view
func (v *view) GetTitle() string {
	return v.title
}

// Write allows to write a string to the view
func (v *view) Write(str string) {
	if v.stdout {
		fmt.Print(str)
		return
	}

	v.mux.Lock()

	content := v.partial + str
	newLines := strings.Split(content, "\n")

	// The last element is either empty (content ended with a newline) or an
	// incomplete line to be continued on the next write
	v.partial = newLines[len(newLines)-1]
	v.lines = append(v.lines, newLines[:len(newLines)-1]...)

	if overflow := len(v.lines) - maxBufferLines; overflow > 0 {
		v.lines = append([]string(nil), v.lines[overflow:]...)
	}

	v.version++
	notify := v.notify

	v.mux.Unlock()

	if notify != nil {
		notify()
	}
}

// Writef allows to write a string to the view with some given arguments
func (v *view) Writef(str string, args ...interface{}) {
	v.Write(fmt.Sprintf(str, args...))
}

// setNotify registers the callback nudging the terminal UI on new content
func (v *view) setNotify(notify func()) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.notify = notify
}

// snapshot returns the current lines (including a trailing partial line when
// present) along with a version incremented on each write
func (v *view) snapshot() ([]string, uint64) {
	v.mux.Lock()
	defer v.mux.Unlock()

	lines := v.lines
	if v.partial != "" {
		lines = append(lines[:len(lines):len(lines)], v.partial)
	}

	return lines, v.version
}
