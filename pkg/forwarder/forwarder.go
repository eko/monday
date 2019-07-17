package forwarder

import (
	"fmt"
	"strconv"
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
	GetForwardType() string
	Forward() error
	Stop() error
	GetReadyChannel() chan struct{}
	GetStopChannel() chan struct{}
}

// Forwarder is the struct that manage running local applications
type Forwarder struct {
	proxy      *proxy.Proxy
	forwards   []*config.Forward
	forwarders sync.Map
}

// NewForwarder instancites a Forwarder struct from configuration data
func NewForwarder(proxy *proxy.Proxy, project *config.Project) *Forwarder {
	return &Forwarder{
		proxy:    proxy,
		forwards: project.Forwards,
	}
}

// ForwardAll runs all applications forwarders in separated goroutines
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

// Stop stops all currently active forwarders
func (f *Forwarder) Stop() {
	f.forwarders.Range(func(key, value interface{}) bool {
		for _, forwarder := range value.([]ForwarderInterface) {
			forwarder.Stop()
		}

		return true
	})
}

func (f *Forwarder) addForwarder(name string, forwarder ForwarderInterface) {
	var forwarders = make([]ForwarderInterface, 0)

	if forwarders, ok := f.forwarders.Load(name); ok {
		forwarders = forwarders.([]ForwarderInterface)
	}

	forwarders = append(forwarders, forwarder)

	f.forwarders.Store(name, forwarders)
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
	proxyForwards := make([]*proxy.ProxyForward, 0)

	if forward.IsProxified() {
	PortsLoop:
		for _, ports := range values.Ports {
			localPort, forwardPort := splitLocalAndForwardPorts(ports)

			var proxyForward *proxy.ProxyForward

			switch forward.Type {
			case config.ForwarderKubernetesRemote:
				remoteProxyPort := strconv.Itoa(kubernetes.RemoteSSHProxyPort)
				proxyForward = proxy.NewProxyForward(forward.Name, values.Hostname, remoteProxyPort, remoteProxyPort)
				proxyForwards = append(proxyForwards, proxyForward)
				f.proxy.AddProxyForward(forward.Name, proxyForward)

				proxifiedPorts = append(proxifiedPorts, proxyForward.GetProxifiedPorts())

				break PortsLoop

			default:
				proxyForward = proxy.NewProxyForward(forward.Name, values.Hostname, localPort, forwardPort)
				proxyForwards = append(proxyForwards, proxyForward)
				f.proxy.AddProxyForward(forward.Name, proxyForward)

				proxifiedPorts = append(proxifiedPorts, proxyForward.GetProxifiedPorts())
			}
		}
	}

	switch forward.Type {
	// Kubernetes local port-forward: give proxy port as local port and forwarded port, use proxy
	case config.ForwarderKubernetes:
		forwarder, err := kubernetes.NewForwarder(forward.Type, forward.Name, values.Context, values.Namespace, proxifiedPorts, values.Labels)
		if err != nil {
			fmt.Printf("❌  %s\n", err.Error())
			return
		}

		f.addForwarder(forward.Name, forwarder)

	// Kubernetes remote forward: open both a SSH remote-forward connection and a Kubernetes port-forward, use proxy
	case config.ForwarderKubernetesRemote:
		// First, set pod's proxy
		forwarder, err := kubernetes.NewForwarder(forward.Type, forward.Name, values.Context, values.Namespace, proxifiedPorts, values.Labels)
		if err != nil {
			fmt.Printf("❌  %s\n", err.Error())
			return
		}

		f.addForwarder(forward.Name, forwarder)

		// Then, ssh remote-forward for all specified ports to pod's container
		for _, ports := range values.Ports {
			for _, proxyForward := range proxyForwards {
				localPort, forwardPort := splitLocalAndForwardPorts(ports)
				values.Remote = "root@127.0.0.1"
				args := append(values.Args, fmt.Sprintf("-p %s", proxyForward.ProxyPort))

				forwarder, err := ssh.NewForwarder(config.ForwarderSSHRemote, values.Remote, localPort, forwardPort, args)
				if err != nil {
					fmt.Printf("❌  %s\n", err.Error())
					return
				}

				f.addForwarder(forward.Name, forwarder)
			}
		}

	// SSH local forward: give proxy port as local port and forwarded port, use proxy
	case config.ForwarderSSH:
		for _, proxyForward := range proxyForwards {
			forwarder, err := ssh.NewForwarder(forward.Type, values.Remote, proxyForward.ProxyPort, proxyForward.ForwardPort, values.Args)
			if err != nil {
				fmt.Printf("❌  %s\n", err.Error())
				return
			}

			f.addForwarder(forward.Name, forwarder)
		}

	// SSH remote forward: give local port and forwarded port, do not proxy
	case config.ForwarderSSHRemote:
		for _, proxyForward := range proxyForwards {
			forwarder, err := ssh.NewForwarder(forward.Type, values.Remote, proxyForward.LocalPort, proxyForward.ForwardPort, values.Args)
			if err != nil {
				fmt.Printf("❌  %s\n", err.Error())
				return
			}

			f.addForwarder(forward.Name, forwarder)
		}
	}

	if forwarders, ok := f.forwarders.Load(forward.Name); ok {
		for _, forwarder := range forwarders.([]ForwarderInterface) {
			go func(forwarder ForwarderInterface) {
				for {
					err := forwarder.Forward()
					if err != nil {
						time.Sleep(1 * time.Second)
						fmt.Printf("%v\n👓  Forwarder: lost port-forward connection trying to reconnect...\n", err)
					}
				}
			}(forwarder)

			switch forwarder.GetForwardType() {
			case config.ForwarderKubernetesRemote:
				// Wait for the proxy to be ready before going next with the SSH remote-forwards
				<-forwarder.GetReadyChannel()
			}
		}
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
