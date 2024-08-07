package proxy

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"

	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/ui"
)

const (
	// ProxyPortStart is the first port that will be allocated by the proxy component.
	// Others will be incremented by 1 each time
	ProxyPortStart = "9400"
)

type Proxy interface {
	Listen() error
	Stop() error
	AddProxyForward(name string, proxyForward *ProxyForward)
}

// proxy represents the proxy component instance
type proxy struct {
	ProxyForwards      map[string][]*ProxyForward
	hostfile           hostfile.Hostfile
	listeners          map[string]net.Listener
	listening          bool
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
func NewProxy(view ui.View, hostfile hostfile.Hostfile) *proxy {
	return &proxy{
		ProxyForwards: make(map[string][]*ProxyForward, 0),
		hostfile:      hostfile,
		listeners:     make(map[string]net.Listener),
		listening:     true,
		latestPort:    ProxyPortStart,
		lastIpByteA:   127,
		lastIpByteB:   0,
		lastIpByteC:   1,
		lastIpByteD:   0,
		attributedIPs: make(map[string]string),
		view:          view,
	}
}

// Listen opens a TCP proxy for each ProxyForward instance
func (p *proxy) Listen() error {
	for name, pfs := range p.ProxyForwards {
		for _, pf := range pfs {
			if pf.LocalPort == "" {
				// In case no local port is specified: don't handle connections
				continue
			}

			key := fmt.Sprintf("%s_%s", name, pf.LocalPort)

			// We already have a listening port
			if _, ok := p.listeners[key]; ok {
				return nil
			}

			p.view.Writef("🔌  Proxifying %s locally (%s:%s) <-> forwarding to %s:%s\n", pf.GetHostname(), pf.LocalIP, pf.LocalPort, pf.GetProxyHostname(), pf.ProxyPort)

			go p.handleConnections(pf, key)
		}
	}

	return nil
}

// Stop stops all currently active proxy listeners
func (p *proxy) Stop() error {
	p.listening = false

	for name, listener := range p.listeners {
		err := listener.Close()
		if err != nil {
			p.view.Writef("❌  An error has occured while stopping proxy listener '%s': %v\n", name, err)
		}
	}

	for _, proxyForwards := range p.ProxyForwards {
		for _, pf := range proxyForwards {
			err := p.hostfile.RemoveHost(pf.GetHostname())
			if err != nil {
				p.view.Writef("❌  An error has occured while trying to remove host from file for application '%s' (ip: %s): %v\n", pf.Name, pf.LocalIP, err)
			}
		}
	}

	return nil
}

func (p *proxy) handleConnections(pf *ProxyForward, key string) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", pf.LocalIP, pf.LocalPort))
	if err != nil {
		p.view.Writef("❌  Could not create proxy listener for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
		return
	}

	p.listenerMux.Lock()
	p.listeners[key] = listener
	p.listenerMux.Unlock()

	// Accept clients and proxify calls
	for {
		client, err := listener.Accept()
		if !p.listening {
			break
		}
		if err != nil {
			p.view.Writef("❌  Could not accept client connection for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
			return
		}

		defer client.Close()

		target, err := net.Dial("tcp", fmt.Sprintf("%s:%s", pf.GetProxyHostname(), pf.ProxyPort))
		if err != nil {
			p.view.Writef("❌  Error when dialing with target for '%s:%s' (%s): %v\n", pf.GetProxyHostname(), pf.LocalPort, pf.ProxyPort, err)
			return
		}

		defer target.Close()

		go io.Copy(client, target)
		go io.Copy(target, client)
	}
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
