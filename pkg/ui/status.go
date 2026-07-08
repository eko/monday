package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ForwardState represents the connection state of a forwarded application
type ForwardState string

const (
	StateWaiting      ForwardState = "waiting"
	StateConnecting   ForwardState = "connecting"
	StateReady        ForwardState = "ready"
	StateReconnecting ForwardState = "reconnecting"
	StateError        ForwardState = "error"
	StateStopped      ForwardState = "stopped"
)

var stateIcons = map[ForwardState]string{
	StateWaiting:      "○",
	StateConnecting:   "◌",
	StateReady:        "●",
	StateReconnecting: "◍",
	StateError:        "✕",
	StateStopped:      "○",
}

var stateStdoutIcons = map[ForwardState]string{
	StateWaiting:      "⏳",
	StateConnecting:   "🔌",
	StateReady:        "✅",
	StateReconnecting: "🔁",
	StateError:        "❌",
	StateStopped:      "⏹",
}

// ForwardStatus holds the latest known state of a forwarded application
type ForwardStatus struct {
	Name      string
	Type      string
	Ports     string
	State     ForwardState
	Message   string
	UpdatedAt time.Time

	// Reconnects counts how many times the connection has been lost
	Reconnects int

	// History holds periodic state samples, rendered as a mini graph
	History []ForwardState
}

// Statuses is a thread-safe registry holding the latest state of each
// forwarded application, rendered live by the terminal UI
type Statuses struct {
	mux     sync.Mutex
	order   []string
	items   map[string]*ForwardStatus
	version uint64

	// notify nudges the terminal UI that a status changed
	notify func()

	// stdoutView receives state transitions as plain lines when the terminal
	// UI is disabled
	stdoutView View
}

// NewStatuses returns a new forward statuses registry. When stdoutView is
// given, state transitions are printed to it instead of being rendered by the
// terminal UI
func NewStatuses(stdoutView View) *Statuses {
	return &Statuses{
		items:      make(map[string]*ForwardStatus),
		stdoutView: stdoutView,
	}
}

// Register declares a forwarded application, in its initial waiting state
func (s *Statuses) Register(name, forwardType string, ports []string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.items[name]; ok {
		return
	}

	s.order = append(s.order, name)
	sort.Slice(s.order, func(i, j int) bool {
		return strings.ToLower(s.order[i]) < strings.ToLower(s.order[j])
	})

	s.items[name] = &ForwardStatus{
		Name:      name,
		Type:      forwardType,
		Ports:     strings.Join(ports, ","),
		State:     StateWaiting,
		UpdatedAt: time.Now(),
	}
	s.version++
}

// Set updates the state of a forwarded application
func (s *Statuses) Set(name string, state ForwardState, message string) {
	s.mux.Lock()

	status, ok := s.items[name]
	if !ok {
		s.mux.Unlock()
		return
	}

	previousState := status.State

	status.State = state
	status.Message = message
	status.UpdatedAt = time.Now()

	if state == StateReconnecting && previousState != StateReconnecting {
		status.Reconnects++
	}

	s.version++

	notify := s.notify
	stdoutView := s.stdoutView

	s.mux.Unlock()

	if stdoutView != nil && previousState != state {
		stdoutView.Writef("%s  Forward '%s' is now %s: %s\n", stateStdoutIcons[state], name, state, message)
	}

	if notify != nil {
		notify()
	}
}

// setNotify registers the callback nudging the terminal UI on status change
func (s *Statuses) setNotify(notify func()) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.notify = notify
}

// Sample appends the current state of each forward to its history, keeping at
// most maxHistory samples, so it can be rendered as a mini state graph
func (s *Statuses) Sample(maxHistory int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, status := range s.items {
		status.History = append(status.History, status.State)

		if overflow := len(status.History) - maxHistory; overflow > 0 {
			status.History = append([]ForwardState(nil), status.History[overflow:]...)
		}
	}
}

// snapshot returns a copy of the current statuses, in registration order,
// along with a version incremented on each change
func (s *Statuses) snapshot() ([]ForwardStatus, uint64) {
	s.mux.Lock()
	defer s.mux.Unlock()

	statuses := make([]ForwardStatus, 0, len(s.order))
	for _, name := range s.order {
		status := *s.items[name]
		status.History = append([]ForwardState(nil), status.History...)
		statuses = append(statuses, status)
	}

	return statuses, s.version
}

// shortDuration renders a compact human duration such as "12s" or "3m05s"
func shortDuration(d time.Duration) string {
	d = d.Round(time.Second)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}

	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}
