package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ProxyState represents the state of a proxied hostname
type ProxyState string

const (
	// ProxyStateWaiting means the hostname is registered but not routed yet
	ProxyStateWaiting ProxyState = "waiting"
	// ProxyStateMapped means the hostname resolves locally but no port is proxied
	ProxyStateMapped ProxyState = "mapped"
	// ProxyStateListening means connections are accepted and forwarded
	ProxyStateListening ProxyState = "listening"
	// ProxyStateError means the proxy could not listen for this hostname
	ProxyStateError ProxyState = "error"
)

// ProxyStatus holds the live routing and traffic information of a proxied hostname
type ProxyStatus struct {
	// Name is the forward or application name this proxy entry belongs to
	Name       string
	Hostname   string
	LocalIP    string
	LocalPort  string
	TargetHost string
	TargetPort string
	State      ProxyState
	Message    string
	UpdatedAt  time.Time

	// ActiveConnections is the number of currently open client connections
	ActiveConnections int
	// TotalConnections is the number of client connections handled since startup
	TotalConnections int
	// Errors is the number of connections that could not reach their target
	Errors int
	// BytesReceived counts bytes flowing from the target to local clients
	BytesReceived int64
	// BytesSent counts bytes flowing from local clients to the target
	BytesSent int64
}

// ProxyStatuses is a thread-safe registry holding the live state and traffic
// metrics of each proxied hostname, rendered by the terminal UI proxy pane
type ProxyStatuses struct {
	mux     sync.Mutex
	order   []string
	items   map[string]*ProxyStatus
	version uint64

	notify func()
}

// NewProxyStatuses returns a new proxy statuses registry
func NewProxyStatuses() *ProxyStatuses {
	return &ProxyStatuses{
		items: make(map[string]*ProxyStatus),
	}
}

func proxyStatusKey(hostname, localPort string) string {
	return fmt.Sprintf("%s_%s", hostname, localPort)
}

// Register declares a proxied hostname and its routing information, attached
// to the given forward or application name
func (p *ProxyStatuses) Register(name, hostname, localIP, localPort, targetHost, targetPort string) {
	p.mux.Lock()

	key := proxyStatusKey(hostname, localPort)

	if status, ok := p.items[key]; ok {
		status.Name = name
		status.LocalIP = localIP
		status.TargetHost = targetHost
		status.TargetPort = targetPort
		p.mux.Unlock()
		p.notifyChange()

		return
	}

	state := ProxyStateWaiting
	if localPort == "" {
		// Hostname-only mapping (e.g. a local application): nothing to proxify
		state = ProxyStateMapped
	}

	p.order = append(p.order, key)
	p.items[key] = &ProxyStatus{
		Name:       name,
		Hostname:   hostname,
		LocalIP:    localIP,
		LocalPort:  localPort,
		TargetHost: targetHost,
		TargetPort: targetPort,
		State:      state,
		UpdatedAt:  time.Now(),
	}

	sort.Slice(p.order, func(i, j int) bool {
		first, second := p.items[p.order[i]], p.items[p.order[j]]

		if !strings.EqualFold(first.Hostname, second.Hostname) {
			return strings.ToLower(first.Hostname) < strings.ToLower(second.Hostname)
		}

		return first.LocalPort < second.LocalPort
	})

	p.version++

	p.mux.Unlock()
	p.notifyChange()
}

// SetState updates the state of a proxied hostname
func (p *ProxyStatuses) SetState(hostname, localPort string, state ProxyState, message string) {
	p.mux.Lock()

	status, ok := p.items[proxyStatusKey(hostname, localPort)]
	if !ok {
		p.mux.Unlock()
		return
	}

	status.State = state
	status.Message = message
	status.UpdatedAt = time.Now()
	p.version++

	p.mux.Unlock()
	p.notifyChange()
}

// ConnectionOpened records a new client connection on a proxied hostname
func (p *ProxyStatuses) ConnectionOpened(hostname, localPort string) {
	p.mux.Lock()

	if status, ok := p.items[proxyStatusKey(hostname, localPort)]; ok {
		status.ActiveConnections++
		status.TotalConnections++
		p.version++
	}

	p.mux.Unlock()
	p.notifyChange()
}

// ConnectionClosed records the end of a client connection
func (p *ProxyStatuses) ConnectionClosed(hostname, localPort string) {
	p.mux.Lock()

	if status, ok := p.items[proxyStatusKey(hostname, localPort)]; ok {
		if status.ActiveConnections > 0 {
			status.ActiveConnections--
		}
		p.version++
	}

	p.mux.Unlock()
	p.notifyChange()
}

// AddTraffic records bytes transferred through a proxied hostname while the
// connection is still open, so long-lived connections show live traffic
func (p *ProxyStatuses) AddTraffic(hostname, localPort string, received, sent int64) {
	p.mux.Lock()

	if status, ok := p.items[proxyStatusKey(hostname, localPort)]; ok {
		status.BytesReceived += received
		status.BytesSent += sent
		p.version++
	}

	p.mux.Unlock()
	p.notifyChange()
}

// ConnectionFailed records a client connection that could not reach its target
func (p *ProxyStatuses) ConnectionFailed(hostname, localPort, message string) {
	p.mux.Lock()

	if status, ok := p.items[proxyStatusKey(hostname, localPort)]; ok {
		status.Errors++
		status.Message = message
		status.UpdatedAt = time.Now()
		p.version++
	}

	p.mux.Unlock()
	p.notifyChange()
}

// setNotify registers the callback nudging the terminal UI on change
func (p *ProxyStatuses) setNotify(notify func()) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.notify = notify
}

func (p *ProxyStatuses) notifyChange() {
	p.mux.Lock()
	notify := p.notify
	p.mux.Unlock()

	if notify != nil {
		notify()
	}
}

// Snapshot returns a copy of the current proxy statuses, in registration order
func (p *ProxyStatuses) Snapshot() []ProxyStatus {
	statuses, _ := p.snapshot()
	return statuses
}

// snapshot returns a copy of the current proxy statuses, in registration order
func (p *ProxyStatuses) snapshot() ([]ProxyStatus, uint64) {
	p.mux.Lock()
	defer p.mux.Unlock()

	statuses := make([]ProxyStatus, 0, len(p.order))
	for _, key := range p.order {
		statuses = append(statuses, *p.items[key])
	}

	return statuses, p.version
}

// humanBytes renders a compact human-readable byte count such as "12.3MB"
func humanBytes(count int64) string {
	const unit = 1024

	if count < unit {
		return fmt.Sprintf("%dB", count)
	}

	div, exp := int64(unit), 0
	for n := count / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB", float64(count)/float64(div), "KMGTPE"[exp])
}
