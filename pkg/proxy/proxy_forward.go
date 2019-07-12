package proxy

import "fmt"

type ProxyForward struct {
	Name        string
	Hostname    string
	LocalPort   string
	ForwardPort string
	LocalIP     string
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

func (p *ProxyForward) SetLocalIP(ip string) {
	p.LocalIP = ip
}

func (p *ProxyForward) SetProxyPort(port string) {
	p.ProxyPort = port
}

func (p *ProxyForward) GetProxifiedPorts() string {
	return fmt.Sprintf("%s:%s", p.ProxyPort, p.ForwardPort)
}

func (p *ProxyForward) GetHostname() string {
	if p.Hostname != "" {
		return p.Hostname
	}

	return p.Name
}
