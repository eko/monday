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

type Proxy struct {
	ProxyForwards map[string][]*ProxyForward
	listeners     map[string]net.Listener
	listening     bool
	mux           sync.Mutex
	latestPort    string
	attributedIPs map[string]net.IP
}

func NewProxy() *Proxy {
	return &Proxy{
		ProxyForwards: make(map[string][]*ProxyForward, 0),
		listeners:     make(map[string]net.Listener),
		listening:     true,
		latestPort:    ProxyPortStart,
		attributedIPs: make(map[string]net.IP, 0),
	}
}

func (p *Proxy) Listen() error {
	for name, pfs := range p.ProxyForwards {
		for _, pf := range pfs {
			key := fmt.Sprintf("%s_%s", name, pf.LocalPort)

			// We already have a listening port
			if _, ok := p.listeners[key]; ok {
				return nil
			}

			err := hostfile.AddHost(pf.LocalIP, pf.GetHostname())
			if err != nil {
				return err
			}

			fmt.Printf("🔌  Proxifying %s locally on %s (port %s) - forwarding to port %s\n", pf.GetHostname(), pf.LocalIP, pf.LocalPort, pf.ProxyPort)

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

	p.listeners[key] = listener

	// Accept clients and proxify calls
	for {
		client, err := listener.Accept()
		if !p.listening {
			break
		}
		if err != nil {
			fmt.Printf("❌  Could not accept client connection for '%s:%s' (%s): %v\n", pf.LocalIP, pf.LocalPort, pf.GetHostname(), err)
		}

		target, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", pf.ProxyPort))
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

func (p *Proxy) AddProxyForward(name string, proxyForward *ProxyForward) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.generateProxyPort(proxyForward)

	if pfs, ok := p.ProxyForwards[name]; ok {
		p.ProxyForwards[name] = append(pfs, proxyForward)
	} else {
		p.ProxyForwards[name] = append(pfs, proxyForward)
	}
}

func (p *Proxy) GenerateIPs() error {
	d := 1

	for name, proxyForward := range p.ProxyForwards {
		for _, pf := range proxyForward {
			// Create new listener on a dedicated IP address if the service
			// does not already have an IP address attributed, elsewhere give the already
			// attributed address
			var localIP net.IP
			var err error
			if v, ok := p.attributedIPs[name]; ok {
				localIP = v
			} else {
				localIP, err = generateIP(127, 1, 2, d, pf.LocalPort)
				d = d + 1
				if err != nil {
					return err
				}
			}

			p.attributedIPs[name] = localIP
			pf.SetLocalIP(localIP.String())
		}
	}

	return nil
}

func (p *Proxy) GetProxyForwardForLocalPort(name string, localPort string) *ProxyForward {
	if pfs, ok := p.ProxyForwards[name]; ok {
		for _, pf := range pfs {
			if pf.LocalPort == localPort {
				return pf
			}
		}
	}

	return nil
}

func (p *Proxy) GetProxyForwardForForwardPort(name string, forwardPort string) *ProxyForward {
	if pfs, ok := p.ProxyForwards[name]; ok {
		for _, pf := range pfs {
			if pf.ForwardPort == forwardPort {
				return pf
			}
		}
	}

	return nil
}

func (p *Proxy) generateProxyPort(proxyForward *ProxyForward) {
	integerPort, _ := strconv.Atoi(p.latestPort)
	p.latestPort = strconv.Itoa(integerPort + 1)

	proxyForward.SetProxyPort(p.latestPort)
}
