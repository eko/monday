package ssh

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/policygenius/monday/pkg/config"
	uimocks "github.com/policygenius/monday/internal/tests/mocks/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewForwarder(t *testing.T) {
	// Given
	remote := "root@acme.tld"
	localPort := "8080"
	forwardPort := "8081"
	args := []string{"-i /tmp/my/private.key"}

	view := &uimocks.ViewInterface{}

	// When
	forwarder, err := NewForwarder(view, config.ForwarderSSH, remote, localPort, forwardPort, args)

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderSSH, forwarder.forwardType)
	assert.Equal(t, remote, forwarder.remote)
	assert.Equal(t, localPort, forwarder.localPort)
	assert.Equal(t, forwardPort, forwarder.forwardPort)
	assert.Equal(t, args, forwarder.args)

	assert.Nil(t, forwarder.cmd)
}

func TestGetForwardType(t *testing.T) {
	// Given
	view := &uimocks.ViewInterface{}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, "root@acme.tld", "8080", "8081", []string{})

	// When
	forwardType := forwarder.GetForwardType()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderSSHRemote, forwardType)
}

func TestGetReadyChannel(t *testing.T) {
	// Given
	view := &uimocks.ViewInterface{}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, "root@acme.tld", "8080", "8081", []string{})

	// When
	channel := forwarder.GetReadyChannel()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.IsType(t, make(chan struct{}), channel)
}

func TestGetStopChannel(t *testing.T) {
	// Given
	view := &uimocks.ViewInterface{}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, "root@acme.tld", "8080", "8081", []string{})

	// When
	channel := forwarder.GetStopChannel()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.IsType(t, make(chan struct{}), channel)
}

func TestForwardLocal(t *testing.T) {
	// Given
	execCommand = mockExecCommand

	view := &uimocks.ViewInterface{}

	forwarder, err := NewForwarder(view, config.ForwarderSSH, "root@acme.tld", "8080", "8081", []string{})

	// When
	err = forwarder.Forward()

	// Then
	assert.Nil(t, err)

	assert.IsType(t, new(exec.Cmd), forwarder.cmd)

	runCommand := strings.Replace(strings.Join(forwarder.cmd.Args, " "), "echo <ssh>", "ssh", -1)
	assert.Equal(t, "ssh -oUserKnownHostsFile=/dev/null -oStrictHostKeyChecking=no -N -L 8080:127.0.0.1:8081 root@acme.tld", runCommand)
}

func TestForwardRemote(t *testing.T) {
	// Given
	execCommand = mockExecCommand

	view := &uimocks.ViewInterface{}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, "root@acme.tld", "8080", "8081", []string{})

	// When
	err = forwarder.Forward()

	// Then
	assert.Nil(t, err)

	assert.IsType(t, new(exec.Cmd), forwarder.cmd)

	runCommand := strings.Replace(strings.Join(forwarder.cmd.Args, " "), "echo <ssh>", "ssh", -1)
	assert.Equal(t, "ssh -oUserKnownHostsFile=/dev/null -oStrictHostKeyChecking=no -N -R 8080:127.0.0.1:8081 root@acme.tld", runCommand)
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	args = append([]string{"<ssh>"}, args...)
	return exec.Command("echo", args...)
}
