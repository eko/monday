package ssh

import (
	"fmt"
	"os/exec"

	"github.com/eko/monday/internal/config"
)

type Forwarder struct {
	forwardType string
	remote      string
	localPort   string
	forwardPort string
	args        []string
}

func NewForwarder(forwardType, remote, localPort, forwardPort string, args []string) (*Forwarder, error) {
	return &Forwarder{
		forwardType: forwardType,
		remote:      remote,
		localPort:   localPort,
		forwardPort: forwardPort,
		args:        args,
	}, nil
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

	cmd := exec.Command("ssh", arguments...)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Cannot run the SSH command for port-forwarding '%s' on host '%s': %v", mapping, host, err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("SSH forwarding of '%s' on host '%s' returned an error: %v", mapping, host, err)
	}

	return nil
}
