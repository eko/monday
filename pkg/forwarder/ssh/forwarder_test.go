package ssh

import (
	"testing"

	"github.com/eko/monday/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewForwarder(t *testing.T) {
	// Given
	remote := "root@acme.tld"
	localPort := "8080"
	forwardPort := "8081"
	args := []string{"-i /tmp/my/private.key"}

	// forwardType, remote, localPort, forwardPort string, args []string

	// When
	forwarder, err := NewForwarder(config.ForwarderSSH, remote, localPort, forwardPort, args)

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
