package forward

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eko/monday/internal/wait"
	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/forward/kubernetes"
	"github.com/eko/monday/pkg/forward/ssh"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
)

// Forwarder represents all kinds of forwarders (Kubernetes, others...)
type Forwarder interface {
	ForwardAll(ctx context.Context)
	Stop(ctx context.Context)
}

type ForwarderType interface {
	GetForwardType() string
	Forward(ctx context.Context) error
	Stop(ctx context.Context) error
	GetReadyChannel() chan struct{}
	GetStopChannel() chan struct{}
}

// forwarder is the struct that manage running local applications
type forwarder struct {
	view       ui.View
	proxy      proxy.Proxy
	forwards   []*config.Forward
	forwarders sync.Map
}

// NewForwarder instanciates a Forwarder struct from configuration data
func NewForwarder(view ui.View, proxy proxy.Proxy, project *config.Project) *forwarder {
	return &forwarder{
		view:     view,
		proxy:    proxy,
		forwards: project.Forwards,
	}
}

// ForwardAll runs all applications forwarders in separated goroutines
func (f *forwarder) ForwardAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, forward := range f.forwards {
		wg.Add(1)
		go f.forward(ctx, forward, &wg)
	}

	wg.Wait()

	// Run proxy for port-forwarning
	go func() {
		err := f.proxy.Listen()
		if err != nil {
			f.view.Writef("❌  %s\n", err.Error())
			return
		}
	}()
}

// Stop stops all currently active forwarders
func (f *forwarder) Stop(ctx context.Context) {
	f.forwarders.Range(func(key, value interface{}) bool {
		for _, forwarder := range value.([]ForwarderType) {
			forwarder.Stop(ctx)
		}

		return true
	})
}

func (f *forwarder) addForwarder(name string, forwarder ForwarderType) {
	var forwarders = make([]ForwarderType, 0)

	if fwds, ok := f.forwarders.Load(name); ok {
		forwarders = fwds.([]ForwarderType)
	}

	forwarders = append(forwarders, forwarder)

	f.forwarders.Store(name, forwarders)
}

func (f *forwarder) forward(ctx context.Context, forward *config.Forward, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := f.checkForwardEnvironment(forward); err != nil {
		f.view.Writef("❌  %s\n", err.Error())
		return
	}

	f.view.Writef("📡  Forwarding '%s' over %s...\n", forward.Name, forward.Type)

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
				proxyForward = proxy.NewProxyForward(forward.Name, values.Hostname, values.ProxyHostname, remoteProxyPort, remoteProxyPort)
				proxyForwards = append(proxyForwards, proxyForward)
				f.proxy.AddProxyForward(forward.Name, proxyForward)

				proxifiedPorts = append(proxifiedPorts, proxyForward.GetProxifiedPorts())

				break PortsLoop

			case config.ForwarderProxy:
				proxyForward = proxy.NewProxyForward(forward.Name, values.Hostname, values.ProxyHostname, localPort, forwardPort)
			default:
				proxyForward = proxy.NewProxyForward(forward.Name, values.Hostname, values.ProxyHostname, localPort, forwardPort)
			}

			proxyForwards = append(proxyForwards, proxyForward)
			f.proxy.AddProxyForward(forward.Name, proxyForward)
			proxifiedPorts = append(proxifiedPorts, proxyForward.GetProxifiedPorts())

		}
	}

	switch forward.Type {
	// Kubernetes local port-forward: give proxy port as local port and forwarded port, use proxy
	case config.ForwarderKubernetes:
		forwardPorts := values.Ports
		if forward.IsProxified() {
			forwardPorts = proxifiedPorts
		}
		forwarder, err := kubernetes.NewForwarder(f.view, forward.Type, forward.Name, values.Context, values.Namespace, forwardPorts, values.Labels)
		if err != nil {
			f.view.Writef("❌  %s\n", err.Error())
			return
		}

		f.addForwarder(forward.Name, forwarder)

	// Kubernetes remote forward: open both a SSH remote-forward connection and a Kubernetes port-forward, use proxy
	case config.ForwarderKubernetesRemote:
		// First, set pod's proxy
		forwarder, err := kubernetes.NewForwarder(f.view, forward.Type, forward.Name, values.Context, values.Namespace, proxifiedPorts, values.Labels)
		if err != nil {
			f.view.Writef("❌  %s\n", err.Error())
			return
		}

		f.addForwarder(forward.Name, forwarder)

		// Then, ssh remote-forward for all specified ports to pod's container
		for _, ports := range values.Ports {
			for _, proxyForward := range proxyForwards {
				localPort, forwardPort := splitLocalAndForwardPorts(ports)
				values.Remote = "root@127.0.0.1"
				values.Args = append(values.Args, fmt.Sprintf("-p %s", proxyForward.ProxyPort))

				forwarder, err := ssh.NewForwarder(f.view, config.ForwarderSSHRemote, values, localPort, forwardPort)
				if err != nil {
					f.view.Writef("❌  %s\n", err.Error())
					return
				}

				f.addForwarder(forward.Name, forwarder)
			}
		}

	// SSH local forward: give proxy port as local port and forwarded port, use proxy
	case config.ForwarderSSH:
		for _, proxyForward := range proxyForwards {
			forwarder, err := ssh.NewForwarder(f.view, forward.Type, values, proxyForward.ProxyPort, proxyForward.ForwardPort)
			if err != nil {
				f.view.Writef("❌  %s\n", err.Error())
				return
			}

			f.addForwarder(forward.Name, forwarder)
		}

	// SSH remote forward: give local port and forwarded port, do not proxy
	case config.ForwarderSSHRemote:
		for _, ports := range values.Ports {
			localPort, forwardPort := splitLocalAndForwardPorts(ports)
			forwarder, err := ssh.NewForwarder(f.view, forward.Type, values, localPort, forwardPort)
			if err != nil {
				f.view.Writef("❌  %s\n", err.Error())
				return
			}

			f.addForwarder(forward.Name, forwarder)
		}
	}

	if forwarders, ok := f.forwarders.Load(forward.Name); ok {
		for _, forwarder := range forwarders.([]ForwarderType) {
			backoff := wait.Backoff{
				Min:    100 * time.Millisecond,
				Max:    10 * time.Second,
				Factor: 2,
			}

			go func(forwarder ForwarderType) {
				for {
					err := forwarder.Forward(ctx)
					if err != nil {
						time.Sleep(backoff.Duration())
						f.view.Writef("%v\n👓  Forwarder: lost port-forward connection trying to reconnect...\n", err)
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

func (f *forwarder) checkForwardEnvironment(forward *config.Forward) error {
	// Check forward type is already managed
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
