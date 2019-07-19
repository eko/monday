package ssh

import (
	"fmt"
	"os/exec"

	"github.com/eko/monday/internal/config"
)

type Forwarder struct {
	forwardType  string
	remote       string
	localPort    string
	forwardPort  string
	args         []string
	cmd          *exec.Cmd
	stopChannel  chan struct{}
	readyChannel chan struct{}
}

var (
	execCommand = exec.Command
)

func NewForwarder(forwardType, remote, localPort, forwardPort string, args []string) (*Forwarder, error) {
	return &Forwarder{
		forwardType:  forwardType,
		remote:       remote,
		localPort:    localPort,
		forwardPort:  forwardPort,
		args:         args,
		stopChannel:  make(chan struct{}),
		readyChannel: make(chan struct{}, 1),
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

func (f *Forwarder) Forward() error {
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

	mapping := fmt.Sprintf("%s:127.0.0.1:%s", f.localPort, f.forwardPort)
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
func (f *Forwarder) Stop() error {
	if f.cmd == nil {
		return nil
	}

	err := f.cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}
