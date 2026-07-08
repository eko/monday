package proxy

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eko/monday/internal/wait"
	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/ui"
)

const (
	// ProxyPortStart is the first port that will be allocated by the proxy component.
	// Others will be incremented by 1 each time
	ProxyPortStart = "9400"

	// dialTimeout is the maximum duration for a single dial attempt to a forwarded target
	dialTimeout = 5 * time.Second

	// dialRetryTimeout is the total duration during which a client connection waits for
	// its target to become reachable again (e.g. while a pod is being redeployed)
	dialRetryTimeout = 30 * time.Second
)

type Proxy interface {
	Listen() error
	Stop() error
	AddProxyForward(name string, proxyForward *ProxyForward)
}

// StatusNotifier receives the live routing state and traffic metrics of each
// proxied hostname
type StatusNotifier interface {
	Register(
		name string,
		hostname string,
		localIP string,
		localPort string,
		targetHost string,
		targetPort string,
	)
	SetState(
		hostname string,
		localPort string,
		state ui.ProxyState,
		message string,
	)
	ConnectionOpened(
		hostname string,
		localPort string,
	)
	ConnectionClosed(
		hostname string,
		localPort string,
	)
	AddTraffic(
		hostname string,
		localPort string,
		received int64,
		sent int64,
	)
	ConnectionFailed(
		hostname string,
		localPort string,
		message string,
	)
}

// proxy represents the proxy component instance
type proxy struct {
	ProxyForwards      map[string][]*ProxyForward
	hostfile           hostfile.Hostfile
	statuses           StatusNotifier
	listeners          map[string]net.Listener
	listening          atomic.Bool
	addProxyForwardMux sync.Mutex
	listenerMux        sync.Mutex
	latestPort         string
	lastIpByteA        byte
	lastIpByteB        byte
	lastIpByteC        byte
	lastIpByteD        byte
	attributedIPs      map[string]string
	view               ui.View
}

// NewProxy initializes a new proxy component instance
func NewProxy(
	view ui.View,
	statuses StatusNotifier,
	hostfile hostfile.Hostfile,
) *proxy {
	p := &proxy{
		ProxyForwards: make(map[string][]*ProxyForward, 0),
		hostfile:      hostfile,
		statuses:      statuses,
		listeners:     make(map[string]net.Listener),
		latestPort:    ProxyPortStart,
		lastIpByteA:   127,
		lastIpByteB:   0,
		lastIpByteC:   1,
		lastIpByteD:   0,
		attributedIPs: make(map[string]string),
		view:          view,
	}
	p.listening.Store(true)

	return p
}

// Listen opens a TCP proxy for each ProxyForward instance
func (p *proxy) Listen() error {
	for _, pf := range p.snapshotProxyForwards() {
		if pf.LocalPort == "" {
			// In case no local port is specified: don't handle connections
			continue
		}

		key := fmt.Sprintf("%s_%s", pf.Name, pf.LocalPort)

		p.listenerMux.Lock()

		// We already have a listening port
		if _, ok := p.listeners[key]; ok {
			p.listenerMux.Unlock()
			continue
		}

		listener, err := net.Listen("tcp", net.JoinHostPort(pf.LocalIP, pf.LocalPort))
		if err != nil {
			p.listenerMux.Unlock()
			p.statuses.SetState(pf.GetHostname(), pf.LocalPort, ui.ProxyStateError, err.Error())
			p.view.Writef("❌  Could not create proxy listener for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
			continue
		}

		p.listeners[key] = listener
		p.listenerMux.Unlock()

		p.statuses.SetState(pf.GetHostname(), pf.LocalPort, ui.ProxyStateListening, "")
		p.view.Writef("🔌  Proxifying %s locally (%s:%s) <-> forwarding to %s:%s\n", pf.GetHostname(), pf.LocalIP, pf.LocalPort, pf.GetProxyHostname(), pf.ProxyPort)

		go p.handleConnections(pf, listener)
	}

	return nil
}

// snapshotProxyForwards returns a copy of the currently registered proxy
// forwards, so they can be iterated without holding the lock
func (p *proxy) snapshotProxyForwards() []*ProxyForward {
	p.addProxyForwardMux.Lock()
	defer p.addProxyForwardMux.Unlock()

	proxyForwards := make([]*ProxyForward, 0, len(p.ProxyForwards))
	for _, pfs := range p.ProxyForwards {
		proxyForwards = append(proxyForwards, pfs...)
	}

	return proxyForwards
}

// Stop stops all currently active proxy listeners
func (p *proxy) Stop() error {
	p.listening.Store(false)

	p.listenerMux.Lock()
	for name, listener := range p.listeners {
		err := listener.Close()
		if err != nil {
			p.view.Writef("❌  An error has occured while stopping proxy listener '%s': %v\n", name, err)
		}
	}
	p.listenerMux.Unlock()

	for _, pf := range p.snapshotProxyForwards() {
		err := p.hostfile.RemoveHost(pf.GetHostname())
		if err != nil {
			p.view.Writef("❌  An error has occured while trying to remove host from file for application '%s' (ip: %s): %v\n", pf.Name, pf.LocalIP, err)
		}
	}

	return nil
}

// handleConnections accepts clients on the given listener and proxifies calls
// to the forwarded target
func (p *proxy) handleConnections(pf *ProxyForward, listener net.Listener) {
	for {
		client, err := listener.Accept()
		if !p.listening.Load() {
			if client != nil {
				_ = client.Close()
			}
			return
		}
		if err != nil {
			p.view.Writef("❌  Could not accept client connection for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
			return
		}

		go p.proxifyConnection(pf, client)
	}
}

// proxifyConnection pipes a single accepted client connection to its forwarded
// target, waiting for the target to become reachable again if needed (e.g. while
// an application is being redeployed)
func (p *proxy) proxifyConnection(pf *ProxyForward, client net.Conn) {
	defer func() { _ = client.Close() }()

	target, err := p.dialTarget(pf)
	if err != nil {
		p.statuses.ConnectionFailed(pf.GetHostname(), pf.LocalPort, err.Error())
		p.view.Writef("❌  Could not reach target '%s:%s' for '%s': %v\n", pf.GetProxyHostname(), pf.ProxyPort, pf.GetHostname(), err)
		return
	}
	defer func() { _ = target.Close() }()

	p.statuses.ConnectionOpened(pf.GetHostname(), pf.LocalPort)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		// Count bytes while they flow so long-lived connections (databases,
		// message brokers, websockets...) show live traffic
		writer := &countingWriter{
			writer: target,
			count: func(n int64) {
				p.statuses.AddTraffic(pf.GetHostname(), pf.LocalPort, 0, n)
			},
		}

		_, _ = io.Copy(writer, client)
		closeWrite(target)
	}()

	go func() {
		defer wg.Done()

		writer := &countingWriter{
			writer: client,
			count: func(n int64) {
				p.statuses.AddTraffic(pf.GetHostname(), pf.LocalPort, n, 0)
			},
		}

		_, _ = io.Copy(writer, target)
		closeWrite(client)
	}()

	wg.Wait()

	p.statuses.ConnectionClosed(pf.GetHostname(), pf.LocalPort)
}

// countingWriter wraps a writer and reports the number of bytes written
type countingWriter struct {
	writer io.Writer
	count  func(n int64)
}

func (w *countingWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	if n > 0 {
		w.count(int64(n))
	}

	return n, err
}

// dialTarget tries to connect to the forwarded target, retrying with backoff while
// the underlying forward is reconnecting instead of dropping the client connection
func (p *proxy) dialTarget(pf *ProxyForward) (net.Conn, error) {
	address := net.JoinHostPort(pf.GetProxyHostname(), pf.ProxyPort)
	deadline := time.Now().Add(dialRetryTimeout)

	backoff := wait.Backoff{
		Min:    200 * time.Millisecond,
		Max:    2 * time.Second,
		Factor: 2,
		Jitter: true,
	}

	for {
		target, err := net.DialTimeout("tcp", address, dialTimeout)
		if err == nil {
			return target, nil
		}

		if !p.listening.Load() || time.Now().After(deadline) {
			return nil, err
		}

		time.Sleep(backoff.Duration())
	}
}

// closeWrite half-closes a connection after copying so the peer sees EOF while
// the other direction can still drain
func closeWrite(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.CloseWrite()
		return
	}

	_ = conn.Close()
}

// AddProxyForward creates a new ProxyForward instance and attributes an IP address and a proxy port to it
func (p *proxy) AddProxyForward(name string, proxyForward *ProxyForward) {
	p.addProxyForwardMux.Lock()
	defer p.addProxyForwardMux.Unlock()

	err := p.generateIP(proxyForward)
	if err != nil {
		p.view.Writef("❌  An error has occured while generating IP address for '%s': %v\n", proxyForward.Name, err)
	}

	if proxyForward.ProxyPort == "" {
		p.generateProxyPort(proxyForward)
	}

	if proxyForward.LocalPort != "" {
		p.view.Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", proxyForward.GetHostname(), proxyForward.LocalIP, proxyForward.ProxyPort)
	} else {
		p.view.Writef("✅  Successfully mapped hostname '%s' with IP '%s'\n", proxyForward.GetHostname(), proxyForward.LocalIP)
	}

	p.statuses.Register(
		proxyForward.Name,
		proxyForward.GetHostname(),
		proxyForward.LocalIP,
		proxyForward.LocalPort,
		proxyForward.GetProxyHostname(),
		proxyForward.ProxyPort,
	)

	if pfs, ok := p.ProxyForwards[name]; ok {
		p.ProxyForwards[name] = append(pfs, proxyForward)
	} else {
		p.ProxyForwards[name] = append(pfs, proxyForward)
	}
}

func (p *proxy) generateIP(pf *ProxyForward) error {
	var err error

	if attributedIP, ok := p.attributedIPs[pf.GetHostname()]; ok {
		pf.SetLocalIP(attributedIP)
		return nil
	}

	a, b, c, d := getNextIPAddress(p.lastIpByteA, p.lastIpByteB, p.lastIpByteC, p.lastIpByteD)

	p.lastIpByteA = a
	p.lastIpByteB = b
	p.lastIpByteC = c
	p.lastIpByteD = d

	a, b, c, d, err = assignIpToPort(a, b, c, d, pf.LocalPort)
	if err != nil {
		return err
	}

	ip := net.IPv4(a, b, c, d)

	pf.SetLocalIP(ip.String())
	p.attributedIPs[pf.GetHostname()] = ip.String()

	err = p.hostfile.AddHost(pf.LocalIP, pf.GetHostname())
	if err != nil {
		p.view.Writef("❌  An error has occured while trying to write host file for application '%s' (ip: %s): %v\n", pf.Name, pf.LocalIP, err)
	}

	// Also add a ::1 entry for IPv6 on macOS.
	// This is to avoid a 5-second delay issue in Bonjour service.
	// @see: https://superuser.com/questions/370559/10-second-delay-for-local-tld-in-mac-os-x-lion
	switch runtime.GOOS {
	case "darwin":
		err = p.hostfile.AddHost(fmt.Sprintf("::%d:%d:%d:%d", a, b, c, d), pf.GetHostname())
		if err != nil {
			p.view.Writef("❌  An error has occured while trying to write host file for application '%s' (ip: %s): %v\n", pf.Name, pf.LocalIP, err)
		}
	}

	return nil
}

func getNextIPAddress(a, b, c, d byte) (byte, byte, byte, byte) {
	if b == 255 && c == 255 && d == 255 {
		return a, b, c, d
	} else if c == 255 && d == 255 {
		b++
		c = 1
		d = 1
	} else if d == 255 {
		c++
		d = 1
	} else {
		d++
	}

	return a, b, c, d
}

func (p *proxy) generateProxyPort(proxyForward *ProxyForward) {
	integerPort, _ := strconv.Atoi(p.latestPort)
	p.latestPort = strconv.Itoa(integerPort + 1)

	proxyForward.SetProxyPort(p.latestPort)
}
