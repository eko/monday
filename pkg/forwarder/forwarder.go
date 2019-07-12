package forwarder

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/pkg/forwarder/kubernetes"
	"github.com/eko/monday/pkg/forwarder/ssh"
	"github.com/eko/monday/pkg/proxy"
)

// ForwarderInterface represents all kinds of forwarders (Kubernetes, others...)
type ForwarderInterface interface {
	Forward() error
}

// Forwarder is the struct that manage running local applications
type Forwarder struct {
	proxy    *proxy.Proxy
	forwards []*config.Forward
}

// NewForwarder instancites a Forwarder struct from configuration data
func NewForwarder(proxy *proxy.Proxy, project *config.Project) *Forwarder {
	return &Forwarder{
		proxy:    proxy,
		forwards: project.Forwards,
	}
}

// ForwardAll runs all local applications in separated goroutines
func (f *Forwarder) ForwardAll() {
	var wg sync.WaitGroup
	for _, forward := range f.forwards {
		wg.Add(1)
		go f.forward(forward, &wg)
	}

	wg.Wait()

	// Run proxy for port-forwarning
	go func() {
		err := f.proxy.Listen()
		if err != nil {
			fmt.Printf("❌  %s\n", err.Error())
			return
		}
	}()
}

func (f *Forwarder) forward(forward *config.Forward, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := f.checkForwardEnvironment(forward); err != nil {
		fmt.Printf("❌  %s\n", err.Error())
		return
	}

	fmt.Printf("📡  Forwarding '%s' over %s...\n", forward.Name, forward.Type)

	values := forward.Values

	// Initiates proxy for port-forwarding with hostnames
	proxifiedPorts := make([]string, 0)

	if forward.IsProxified() {
		for _, ports := range values.Ports {
			localPort, forwardPort := splitLocalAndForwardPorts(ports)

			proxyForward := proxy.NewProxyForward(forward.Name, values.Hostname, localPort, forwardPort)
			f.proxy.AddProxyForward(forward.Name, proxyForward)

			proxifiedPorts = append(proxifiedPorts, proxyForward.GetProxifiedPorts())
		}
	}

	// Run forwards depending on types
	var forwarders = make([]ForwarderInterface, 0)

	switch forward.Type {
	// Kubernetes local port-forward: give proxy port as local port and forwarded port, use proxy
	case config.ForwarderKubernetes:
		forwarder, err := kubernetes.NewForwarder(values.Context, values.Namespace, proxifiedPorts, values.Labels)
		if err != nil {
			fmt.Printf("❌  %s\n", err.Error())
			return
		}

		forwarders = append(forwarders, forwarder)

	// SSH local forward: give proxy port as local port and forwarded port, use proxy
	case config.ForwarderSSH:
		proxyForward := f.proxy.GetFirstProxyForward(forward.Name)
		forwarder, err := ssh.NewForwarder(forward.Type, values.Remote, proxyForward.ProxyPort, proxyForward.ForwardPort, values.Args)
		if err != nil {
			fmt.Printf("❌  %s\n", err.Error())
			return
		}

		forwarders = append(forwarders, forwarder)

	// SSH remote forward: give local port and forwarded port, do not proxy
	case config.ForwarderSSHRemote:
		for _, ports := range values.Ports {
			localPort, forwardPort := splitLocalAndForwardPorts(ports)
			forwarder, err := ssh.NewForwarder(forward.Type, values.Remote, localPort, forwardPort, values.Args)
			if err != nil {
				fmt.Printf("❌  %s\n", err.Error())
				return
			}

			forwarders = append(forwarders, forwarder)
		}
	}

	for _, forwarder := range forwarders {
		go func(forwarder ForwarderInterface) {
			for {
				err := forwarder.Forward()
				if err != nil {
					time.Sleep(1 * time.Second)
					fmt.Printf("%v\n👓  Forwarder: lost port-forward connection trying to reconnect...\n", err)
				}
			}
		}(forwarder)
	}
}

func (f *Forwarder) checkForwardEnvironment(forward *config.Forward) error {
	// Check executable is already managed
	if result, ok := config.AvailableForwarders[forward.Type]; !ok || !result {
		return fmt.Errorf("The '%s' specified forward type named '%s' is not managed actually", forward.Type, forward.Name)
	}

	// Check if at least 1 port is filled
	if len(forward.Values.Ports) < 1 {
		return fmt.Errorf("The '%s' specified forward type named '%s' does not have any port to forward, please specify them", forward.Type, forward.Name)
	}

	return nil
}

// Returns first local port and forwarded port as second value
func splitLocalAndForwardPorts(ports string) (string, string) {
	parts := strings.Split(ports, ":")
	return parts[0], parts[1]
}
