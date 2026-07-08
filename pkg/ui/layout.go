package ui

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

// Layout holds the terminal UI views and the Bubble Tea program rendering them
type Layout struct {
	uiEnabled bool
	project   string
	version   string

	logsView     *view
	forwardsView *view
	proxyView    *view

	forwardStatuses *Statuses
	proxyStatuses   *ProxyStatuses

	program atomic.Pointer[tea.Program]
	dirty   atomic.Bool
}

// NewLayout returns a new layout instance
func NewLayout(uiEnabled bool) *Layout {
	return &Layout{
		uiEnabled: uiEnabled,
	}
}

// Init initializes the layout views
func (l *Layout) Init() {
	if !l.uiEnabled {
		l.logsView = NewEmptyView("logs")
		l.forwardsView = NewEmptyView("forwards")
		l.proxyView = NewEmptyView("proxy")
		l.forwardStatuses = NewStatuses(l.forwardsView)
		l.proxyStatuses = NewProxyStatuses()

		return
	}

	l.logsView = NewView("logs", "Logs")

	// Forward and proxy events share a single activity view, displayed below
	// the combined "Forwards & Proxy" table
	activityView := NewView("activity", "Forwards & Proxy")
	l.forwardsView = activityView
	l.proxyView = activityView

	for _, v := range []*view{l.logsView, activityView} {
		v.setNotify(l.notify)
	}

	l.forwardStatuses = NewStatuses(nil)
	l.forwardStatuses.setNotify(l.notify)

	l.proxyStatuses = NewProxyStatuses()
	l.proxyStatuses.setNotify(l.notify)
}

// SetProject sets the project name displayed in the terminal UI header
func (l *Layout) SetProject(name string) {
	l.project = name
}

// SetVersion sets the application version displayed in the terminal UI header
func (l *Layout) SetVersion(version string) {
	l.version = version
}

// IsUIEnabled returns true when the terminal UI is enabled
func (l *Layout) IsUIEnabled() bool {
	return l.uiEnabled
}

// GetLogsView returns the logs view structure
func (l *Layout) GetLogsView() *view {
	return l.logsView
}

// GetForwardsView returns the forward view structure
func (l *Layout) GetForwardsView() *view {
	return l.forwardsView
}

// GetProxyView returns the proxy view structure
func (l *Layout) GetProxyView() *view {
	return l.proxyView
}

// GetForwardStatuses returns the forward statuses registry displayed in the
// terminal UI forwards pane
func (l *Layout) GetForwardStatuses() *Statuses {
	return l.forwardStatuses
}

// GetProxyStatuses returns the proxy statuses registry displayed in the
// terminal UI proxy pane
func (l *Layout) GetProxyStatuses() *ProxyStatuses {
	return l.proxyStatuses
}

// Run starts the terminal UI and blocks until the user quits, it is a no-op
// when the terminal UI is disabled
func (l *Layout) Run() error {
	if !l.uiEnabled {
		return nil
	}

	model := newModel(
		l.project,
		l.version,
		[]*view{l.logsView, l.forwardsView},
		l.forwardStatuses,
		l.proxyStatuses,
		&l.dirty,
	)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	l.program.Store(program)
	defer l.program.Store(nil)

	_, err := program.Run()

	return err
}

// notify nudges the running program that new view content is available, taking
// care of not flooding it when a lot of lines are written at once
func (l *Layout) notify() {
	program := l.program.Load()
	if program == nil {
		return
	}

	if l.dirty.CompareAndSwap(false, true) {
		go program.Send(refreshMsg{})
	}
}
