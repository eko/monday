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
	Servers       map[string]net.Listener
	mux           sync.Mutex
}

func NewProxy() *Proxy {
	return &Proxy{
		ProxyForwards: make(map[string][]*ProxyForward, 0),
		Servers:       make(map[string]net.Listener),
	}
}

func (p *Proxy) Listen() error {
	d := 1

	for name, pfs := range p.ProxyForwards {
		for _, pf := range pfs {
			err := func() error {
				p.mux.Lock()
				defer p.mux.Unlock()
				key := fmt.Sprintf("%s_%s", name, pf.LocalPort)

				// We already have a listening port
				if _, ok := p.Servers[key]; ok {
					return nil
				}

				// Create new listener on a dedicated IP address
				localIp, err := generateIP(127, 1, 2, d, pf.LocalPort)
				d = d + 1
				if err != nil {
					return err
				}

				// Add hostname to /etc/hosts file
				hostname := name
				if pf.Hostname != "" {
					hostname = pf.Hostname
				}

				err = hostfile.AddHost(localIp.String(), hostname)
				if err != nil {
					return err
				}

				fmt.Printf("🔌  Proxifying %s locally on %s:%s\n", name, localIp, pf.LocalPort)

				listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", localIp, pf.LocalPort))
				if err != nil {
					return err
				}

				p.Servers[key] = listener

				// Accept clients and proxify calls
				go func() {
					for {
						client, err := listener.Accept()
						if err != nil {
							fmt.Printf("❌  Could not accept client connection for '%s:%s' (%s): %v\n", name, localIp, pf.LocalPort, err)
							return
						}

						defer client.Close()

						target, err := net.Dial("tcp", fmt.Sprintf(":%s", pf.ProxyPort))
						if err != nil {
							fmt.Printf("❌  Error when dialing with forwarder for '%s:%s' (%s): %v\n", name, localIp, pf.LocalPort, err)
							return
						}
						defer target.Close()

						go func() { io.Copy(target, client) }()
						go func() { io.Copy(client, target) }()
					}
				}()

				return nil
			}()

			if err != nil {
				return err
			}
		}
	}

	return nil
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

func (p *Proxy) GetFirstProxyForward(name string) *ProxyForward {
	for forwardName, pf := range p.ProxyForwards {
		if forwardName == name {
			return pf[0]
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
	port := ProxyPortStart

	for _, pfs := range p.ProxyForwards {
		for _, pf := range pfs {
			if pf.ProxyPort != "" {
				integerPort, _ := strconv.Atoi(pf.ProxyPort)
				port = strconv.Itoa(integerPort + 1)
				break
			}
		}
	}

	proxyForward.SetProxyPort(port)
}
