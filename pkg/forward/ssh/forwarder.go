package ssh

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/eko/monday/pkg/ui"

	"github.com/eko/monday/pkg/config"
)

type Forwarder struct {
	view            ui.View
	forwardType     string
	remote          string
	forwardHostname string
	localPort       string
	forwardPort     string
	args            []string
	mux             sync.Mutex
	cmd             *exec.Cmd
	stopped         bool
	stopChannel     chan struct{}
	readyChannel    chan struct{}

	// onStateChange, when set, receives the forward state transitions
	onStateChange func(state ui.ForwardState, message string)
}

var (
	execCommand = exec.Command
)

func NewForwarder(view ui.View, forwardType string, values config.ForwardValues, localPort, forwardPort string) (*Forwarder, error) {
	return &Forwarder{
		view:            view,
		forwardType:     forwardType,
		remote:          values.Remote,
		forwardHostname: values.ForwardHostname,
		localPort:       localPort,
		forwardPort:     forwardPort,
		args:            values.Args,
		stopChannel:     make(chan struct{}),
		readyChannel:    make(chan struct{}, 1),
	}, nil
}

// GetForwardType returns the type of the forward specified in the configuration (ssh, ssh-remote, kubernetes, ...)
func (f *Forwarder) GetForwardType() string {
	return f.forwardType
}

// OnStateChange registers a callback receiving the forward state transitions
func (f *Forwarder) OnStateChange(callback func(state ui.ForwardState, message string)) {
	f.onStateChange = callback
}

func (f *Forwarder) reportState(state ui.ForwardState, message string) {
	if f.onStateChange != nil {
		f.onStateChange(state, message)
	}
}

// GetReadyChannel returns the channel signaled when the SSH tunnel is started
func (f *Forwarder) GetReadyChannel() chan struct{} {
	return f.readyChannel
}

// GetStopChannel returns the channel signaled when the SSH tunnel is stopped
func (f *Forwarder) GetStopChannel() chan struct{} {
	return f.stopChannel
}

// Reconnect kills the current SSH tunnel so a fresh one is established by the
// connection loop
func (f *Forwarder) Reconnect() error {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.cmd != nil && f.cmd.Process != nil {
		return f.cmd.Process.Kill()
	}

	return nil
}

func (f *Forwarder) Forward(_ context.Context) error {
	if f.remote == "" {
		return fmt.Errorf("Please provide a 'remote' attribute specifing the host you want to SSH on")
	}

	var forwardOption string

	switch f.forwardType {
	case config.ForwarderSSH:
		forwardOption = "-L"
	case config.ForwarderSSHRemote:
		forwardOption = "-R"
	}

	var forwardHostname = "127.0.0.1" // Default SSH forward hostname if none specified in config
	if f.forwardHostname != "" {
		forwardHostname = f.forwardHostname
	}

	mapping := fmt.Sprintf("%s:%s:%s", f.localPort, forwardHostname, f.forwardPort)
	host := f.remote

	arguments := append([]string{
		"-oUserKnownHostsFile=/dev/null",
		"-oStrictHostKeyChecking=no",
		"-oServerAliveInterval=10",
		"-oServerAliveCountMax=3",
		"-oExitOnForwardFailure=yes",
		"-oConnectTimeout=10",
		"-N",
		forwardOption,
		mapping,
		host,
	}, f.args...)

	f.mux.Lock()
	if f.stopped {
		f.mux.Unlock()
		return nil
	}

	cmd := execCommand("ssh", arguments...)
	f.cmd = cmd
	f.mux.Unlock()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Cannot run the SSH command for port-forwarding '%s' on host '%s': %v", mapping, host, err)
	}

	select {
	case f.readyChannel <- struct{}{}:
	default:
	}

	f.reportState(ui.StateReady, fmt.Sprintf("tunnel '%s' established on '%s'", mapping, host))

	if err := cmd.Wait(); err != nil {
		f.mux.Lock()
		stopped := f.stopped
		f.mux.Unlock()

		if stopped {
			return nil
		}

		return fmt.Errorf("SSH forwarding of '%s' on host '%s' returned an error: %v", mapping, host, err)
	}

	return nil
}

// Stop stops the current forwarder
func (f *Forwarder) Stop(_ context.Context) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	f.stopped = true

	if f.cmd == nil || f.cmd.Process == nil {
		return nil
	}

	return f.cmd.Process.Kill()
}
