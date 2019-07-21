package proxy

import "fmt"

type ProxyForward struct {
	Name          string
	Hostname      string
	ProxyHostname string
	LocalPort     string
	ForwardPort   string
	LocalIP       string
	ProxyPort     string
}

// NewProxyForward returns a new proxy port-forward instance
func NewProxyForward(name, hostname, proxyHostname, localPort, forwardPort string) *ProxyForward {
	proxyForward := &ProxyForward{
		Name:          name,
		Hostname:      hostname,
		ProxyHostname: proxyHostname,
		LocalPort:     localPort,
		ForwardPort:   forwardPort,
	}

	// In case of a forward type 'proxy', just set the proxy port with
	// the given forward port (proxy component will not generate one)
	if proxyHostname != "" {
		proxyForward.ProxyPort = forwardPort
	}

	return proxyForward
}

// SetLocalIP sets local attributed IP to this forward
func (p *ProxyForward) SetLocalIP(ip string) {
	p.LocalIP = ip
}

// SetProxyPort sets proxy attributed port to this forward
func (p *ProxyForward) SetProxyPort(port string) {
	p.ProxyPort = port
}

// GetProxifiedPorts returns the couple of proxified ports (proxy attributed port:forward port)
func (p *ProxyForward) GetProxifiedPorts() string {
	return fmt.Sprintf("%s:%s", p.ProxyPort, p.ForwardPort)
}

// GetHostname returns the hostname (if defined) of this proxy forward, elsewhere uses the
// service name
func (p *ProxyForward) GetHostname() string {
	if p.Hostname != "" {
		return p.Hostname
	}

	return p.Name
}

// GetProxyHostname returns the proxy sender hostname depending on forward type
// In case of a forward type 'proxy', it will return the specified proxy hostname, elsewhere
// it will return 127.0.0.1 because other forwards forward traffic locally
func (p *ProxyForward) GetProxyHostname() string {
	if p.ProxyHostname != "" {
		return p.ProxyHostname
	}

	return "127.0.0.1"
}
