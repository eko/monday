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
	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
)

// stableConnectionDuration is the minimum uptime after which a connection is
// considered stable again, resetting the reconnection backoff
const stableConnectionDuration = 15 * time.Second

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

// StatusNotifier receives the latest state of each forwarded application
type StatusNotifier interface {
	Register(
		name string,
		forwardType string,
		ports []string,
	)
	Set(
		name string,
		state ui.ForwardState,
		message string,
	)
	SetLogsStreaming(
		name string,
		streaming bool,
	)
}

// forwarder is the struct that manage running local applications
type forwarder struct {
	view       ui.View
	logsView   ui.View
	statuses   StatusNotifier
	proxy      proxy.Proxy
	forwards   []*config.Forward
	forwarders sync.Map

	// pauses holds the pause control of each forward name
	pauses sync.Map
	// reconnecters holds the functions closing the active connections of each forward name
	reconnecters sync.Map
	// logStreamers holds the pod logs source of each forward name
	logStreamers sync.Map
	// logStreams holds the cancel function of each active pod logs stream
	logStreams sync.Map
}

// NewForwarder instanciates a Forwarder struct from configuration data
func NewForwarder(
	view ui.View,
	statuses StatusNotifier,
	proxy proxy.Proxy,
	project *config.Project,
) *forwarder {
	return &forwarder{
		view:     view,
		statuses: statuses,
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
	f.stopLogStreams()

	f.forwarders.Range(func(key, value interface{}) bool {
		for _, forwarder := range value.([]ForwarderType) {
			forwarder.Stop(ctx)
		}

		return true
	})
}

// stateCallback returns the state transitions callback bound to a forward name
func (f *forwarder) stateCallback(name string) func(state ui.ForwardState, message string) {
	return func(state ui.ForwardState, message string) {
		f.statuses.Set(name, state, message)
	}
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
	defer helper.RecoverAndLog(f.view, "forwarder")

	if err := f.checkForwardEnvironment(forward); err != nil {
		f.view.Writef("❌  %s\n", err.Error())
		return
	}

	f.view.Writef("📡  Forwarding '%s' over %s (%s)...\n", forward.Name, forward.Type, strings.Join(forward.Values.Ports, ", "))

	f.statuses.Register(forward.Name, forward.Type, forward.Values.Ports)

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

		forwarder.OnStateChange(f.stateCallback(forward.Name))
		f.registerReconnecter(forward.Name, forwarder.Reconnect)
		f.registerLogStreamer(forward.Name, forwarder)
		f.addForwarder(forward.Name, forwarder)

	// Kubernetes remote forward: open both a SSH remote-forward connection and a Kubernetes port-forward, use proxy
	case config.ForwarderKubernetesRemote:
		// First, set pod's proxy
		forwarder, err := kubernetes.NewForwarder(f.view, forward.Type, forward.Name, values.Context, values.Namespace, proxifiedPorts, values.Labels)
		if err != nil {
			f.view.Writef("❌  %s\n", err.Error())
			return
		}

		forwarder.OnStateChange(f.stateCallback(forward.Name))
		f.registerReconnecter(forward.Name, forwarder.Reconnect)
		f.registerLogStreamer(forward.Name, forwarder)
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

				forwarder.OnStateChange(f.stateCallback(forward.Name))
				f.registerReconnecter(forward.Name, forwarder.Reconnect)
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

			forwarder.OnStateChange(f.stateCallback(forward.Name))
			f.registerReconnecter(forward.Name, forwarder.Reconnect)
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

			forwarder.OnStateChange(f.stateCallback(forward.Name))
			f.registerReconnecter(forward.Name, forwarder.Reconnect)
			f.addForwarder(forward.Name, forwarder)
		}
	}

	if forwarders, ok := f.forwarders.Load(forward.Name); ok {
		for _, forwarder := range forwarders.([]ForwarderType) {
			backoff := wait.Backoff{
				Min:    250 * time.Millisecond,
				Max:    10 * time.Second,
				Factor: 2,
				Jitter: true,
			}

			go func(forwarder ForwarderType) {
				defer helper.RecoverAndLog(f.view, fmt.Sprintf("forward '%s' connection loop", forward.Name))

				for {
					// When paused by the user, wait for a resume before reconnecting
					if control := f.pauseControlFor(forward.Name); control.isPaused() {
						f.statuses.Set(forward.Name, ui.StatePaused, "paused, press 'p' to resume")

						select {
						case <-ctx.Done():
							return
						case <-control.resumeChannel():
						}

						backoff.Reset()
						continue
					}

					f.statuses.Set(forward.Name, ui.StateConnecting, "establishing connection...")

					startedAt := time.Now()

					err := forwarder.Forward(ctx)
					if err == nil || ctx.Err() != nil {
						// Forwarder returned without error: it has been intentionally stopped
						f.statuses.Set(forward.Name, ui.StateStopped, "forward has been stopped")
						return
					}

					// The connection has been closed by a user pause: wait silently
					if f.pauseControlFor(forward.Name).isPaused() {
						continue
					}

					// The connection stayed up long enough to consider the previous
					// failure resolved: restart the backoff from its minimum delay
					if time.Since(startedAt) >= stableConnectionDuration {
						backoff.Reset()
					}

					delay := backoff.Duration()
					f.statuses.Set(forward.Name, ui.StateReconnecting, fmt.Sprintf("%v", err))
					f.view.Writef("👓  Forwarder: lost connection for '%s': %v. Reconnecting in %s...\n", forward.Name, err, delay.Round(time.Millisecond))

					select {
					case <-ctx.Done():
						return
					case <-time.After(delay):
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
