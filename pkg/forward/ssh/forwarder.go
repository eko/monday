package ssh

import (
	"context"
	"fmt"
	"os/exec"

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
	cmd             *exec.Cmd
	stopChannel     chan struct{}
	readyChannel    chan struct{}
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

// GetReadyChannel returns the Kubernetes go client channel for ready event
func (f *Forwarder) GetReadyChannel() chan struct{} {
	return f.readyChannel
}

// GetStopChannel returns the Kubernetes go client channel for stop event
func (f *Forwarder) GetStopChannel() chan struct{} {
	return f.readyChannel
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
		"-N",
		forwardOption,
		mapping,
		host,
	}, f.args...)

	f.cmd = execCommand("ssh", arguments...)

	if err := f.cmd.Start(); err != nil {
		return fmt.Errorf("Cannot run the SSH command for port-forwarding '%s' on host '%s': %v", mapping, host, err)
	}

	if err := f.cmd.Wait(); err != nil {
		return fmt.Errorf("SSH forwarding of '%s' on host '%s' returned an error: %v", mapping, host, err)
	}

	return nil
}

// Stop stops the current forwarder
func (f *Forwarder) Stop(_ context.Context) error {
	if f.cmd == nil {
		return nil
	}

	err := f.cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}
