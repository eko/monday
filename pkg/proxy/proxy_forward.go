package proxy

import "fmt"

type ProxyForward struct {
	Name        string
	Hostname    string
	LocalPort   string
	ForwardPort string
	ProxyPort   string
}

func NewProxyForward(name, hostname, localPort, forwardPort string) *ProxyForward {
	return &ProxyForward{
		Name:        name,
		Hostname:    hostname,
		LocalPort:   localPort,
		ForwardPort: forwardPort,
	}
}

func (p *ProxyForward) SetProxyPort(port string) {
	p.ProxyPort = port
}

func (p *ProxyForward) GetProxifiedPorts() string {
	return fmt.Sprintf("%s:%s", p.ProxyPort, p.ForwardPort)
}
