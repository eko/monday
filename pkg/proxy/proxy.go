package proxy

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/eko/monday/pkg/hostfile"
)

const (
	ProxyPortStart = "9400"
)

type ProxyInterface interface {
	Listen() error
	Stop() error
	AddProxyForward(name string, proxyForward *ProxyForward)
}

type Proxy struct {
	ProxyForwards      map[string][]*ProxyForward
	listeners          map[string]net.Listener
	listening          bool
	addProxyForwardMux sync.Mutex
	listenerMux        sync.Mutex
	latestPort         string
	attributedIPs      map[string]net.IP
	ipLastByte         int
}

func NewProxy() *Proxy {
	return &Proxy{
		ProxyForwards: make(map[string][]*ProxyForward, 0),
		listeners:     make(map[string]net.Listener),
		listening:     true,
		latestPort:    ProxyPortStart,
		attributedIPs: make(map[string]net.IP, 0),
		ipLastByte:    1,
	}
}

// Listen opens a TCP proxy for each ProxyForward instance
func (p *Proxy) Listen() error {
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

			fmt.Printf("🔌  Proxifying %s locally (%s:%s) <-> forwarding to %s:%s\n", pf.GetHostname(), pf.LocalIP, pf.LocalPort, pf.GetProxyHostname(), pf.ProxyPort)

			go p.handleConnections(pf, key)
		}
	}

	return nil
}

// Stop stops all currently active proxy listeners
func (p *Proxy) Stop() error {
	p.listening = false

	for name, listener := range p.listeners {
		err := listener.Close()
		if err != nil {
			fmt.Printf("❌  An error has occured while stopping proxy listener '%s': %v\n", name, err)
		}
	}

	return nil
}

func (p *Proxy) handleConnections(pf *ProxyForward, key string) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", pf.LocalIP, pf.LocalPort))
	if err != nil {
		fmt.Printf("❌  Could not create proxy listener for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
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
			fmt.Printf("❌  Could not accept client connection for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
		}

		target, err := net.Dial("tcp", fmt.Sprintf("%s:%s", pf.GetProxyHostname(), pf.ProxyPort))
		if err != nil {
			fmt.Printf("❌  Error when dialing with forwarder for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
		}

		go func() {
			defer client.Close()
			defer target.Close()
			io.Copy(client, target)
		}()

		go func() {
			defer client.Close()
			defer target.Close()
			io.Copy(target, client)
		}()
	}
}

// AddProxyForward creates a new ProxyForward instance and attributes an IP address and a proxy port to it
func (p *Proxy) AddProxyForward(name string, proxyForward *ProxyForward) {
	p.addProxyForwardMux.Lock()
	defer p.addProxyForwardMux.Unlock()

	p.generateIP(proxyForward)

	if proxyForward.ProxyPort == "" {
		p.generateProxyPort(proxyForward)
	}

	err := hostfile.AddHost(proxyForward.LocalIP, proxyForward.GetHostname())
	if err != nil {
		fmt.Printf("❌  An error has occured while trying to write host file for application '%s' (ip: %s): %v\n", proxyForward.Name, proxyForward.LocalIP, err)
	}

	if proxyForward.LocalPort != "" {
		fmt.Printf("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", proxyForward.GetHostname(), proxyForward.LocalIP, proxyForward.ProxyPort)
	} else {
		fmt.Printf("✅  Successfully mapped hostname '%s' with IP '%s'\n", proxyForward.GetHostname(), proxyForward.LocalIP)
	}

	if pfs, ok := p.ProxyForwards[name]; ok {
		p.ProxyForwards[name] = append(pfs, proxyForward)
	} else {
		p.ProxyForwards[name] = append(pfs, proxyForward)
	}
}

func (p *Proxy) generateIP(pf *ProxyForward) error {
	// Create new listener on a dedicated IP address if the service
	// does not already have an IP address attributed, elsewhere give the already
	// attributed address
	var localIP net.IP
	var err error
	if v, ok := p.attributedIPs[pf.Name]; ok {
		localIP = v
	} else {
		localIP, err = generateIP(127, 1, 2, p.ipLastByte, pf.LocalPort)
		p.ipLastByte = p.ipLastByte + 1
		if err != nil {
			return err
		}
	}

	p.attributedIPs[pf.Name] = localIP
	pf.SetLocalIP(localIP.String())

	return nil
}

func (p *Proxy) generateProxyPort(proxyForward *ProxyForward) {
	integerPort, _ := strconv.Atoi(p.latestPort)
	p.latestPort = strconv.Itoa(integerPort + 1)

	proxyForward.SetProxyPort(p.latestPort)
}
