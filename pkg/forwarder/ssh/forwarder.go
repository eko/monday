package ssh

import (
	"fmt"
	"os/exec"
)

type Forwarder struct {
	remote      string
	localPort   string
	forwardPort string
}

func NewForwarder(remote, localPort, forwardPort string) (*Forwarder, error) {
	return &Forwarder{
		remote:      remote,
		localPort:   localPort,
		forwardPort: forwardPort,
	}, nil
}

func (f *Forwarder) Forward() error {
	if f.remote == "" {
		return fmt.Errorf("Please provide a 'remote' attribute specifing the host you want to SSH on")
	}

	mapping := fmt.Sprintf("%s:127.0.0.1:%s", f.localPort, f.forwardPort)
	host := f.remote

	cmd := exec.Command("ssh", "-N", "-L", mapping, host)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Cannot run the SSH command for port-forwarding '%s' on host '%s': %v", mapping, host, err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("SSH forwarding of '%s' on host '%s' returned an error: %v", mapping, host, err)
	}

	return nil
}
