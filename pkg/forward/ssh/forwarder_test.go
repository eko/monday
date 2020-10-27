package ssh

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/ui"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewForwarder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	localPort := "8080"
	forwardPort := "8081"

	values := config.ForwardValues{
		Remote: "root@acme.tld",
		Args:   []string{"-i /tmp/my/private.key"},
	}

	view := ui.NewMockView(ctrl)

	// When
	forwarder, err := NewForwarder(view, config.ForwarderSSH, values, localPort, forwardPort)

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderSSH, forwarder.forwardType)
	assert.Equal(t, values.Remote, forwarder.remote)
	assert.Equal(t, localPort, forwarder.localPort)
	assert.Equal(t, forwardPort, forwarder.forwardPort)
	assert.Equal(t, values.Args, forwarder.args)

	assert.Nil(t, forwarder.cmd)
}

func TestGetForwardType(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)

	values := config.ForwardValues{
		Remote: "root@acme.tld",
	}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, values, "8080", "8081")

	// When
	forwardType := forwarder.GetForwardType()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderSSHRemote, forwardType)
}

func TestGetReadyChannel(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)

	values := config.ForwardValues{
		Remote: "root@acme.tld",
	}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, values, "8080", "8081")

	// When
	channel := forwarder.GetReadyChannel()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.IsType(t, make(chan struct{}), channel)
}

func TestGetStopChannel(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)

	values := config.ForwardValues{
		Remote: "root@acme.tld",
	}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, values, "8080", "8081")

	// When
	channel := forwarder.GetStopChannel()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.IsType(t, make(chan struct{}), channel)
}

func TestForwardLocal(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	execCommand = mockExecCommand

	view := ui.NewMockView(ctrl)

	values := config.ForwardValues{
		Remote:          "root@acme.tld",
		ForwardHostname: "myforwardhostname.svc.local",
	}

	forwarder, err := NewForwarder(view, config.ForwarderSSH, values, "8080", "8081")

	// When
	err = forwarder.Forward()

	// Then
	assert.Nil(t, err)

	assert.IsType(t, new(exec.Cmd), forwarder.cmd)

	runCommand := strings.Replace(strings.Join(forwarder.cmd.Args, " "), "echo <ssh>", "ssh", -1)
	assert.Equal(t, "ssh -oUserKnownHostsFile=/dev/null -oStrictHostKeyChecking=no -N -L 8080:myforwardhostname.svc.local:8081 root@acme.tld", runCommand)
}

func TestForwardLocalWithForwardHostname(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	execCommand = mockExecCommand

	view := ui.NewMockView(ctrl)

	values := config.ForwardValues{
		Remote: "root@acme.tld",
	}

	forwarder, err := NewForwarder(view, config.ForwarderSSH, values, "8080", "8081")

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	execCommand = mockExecCommand

	view := ui.NewMockView(ctrl)

	values := config.ForwardValues{
		Remote: "root@acme.tld",
	}

	forwarder, err := NewForwarder(view, config.ForwarderSSHRemote, values, "8080", "8081")

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
