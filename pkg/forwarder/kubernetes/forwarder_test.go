package kubernetes

import (
	"testing"

	"github.com/eko/monday/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewForwarder(t *testing.T) {
	// Given
	name := "test-forward"
	context := "aws-int1"
	namespace := "platform"
	ports := []string{"8080:8080"}
	labels := map[string]string{
		"app": "my-test-app",
	}

	// When
	forwarder, err := NewForwarder(config.ForwarderKubernetes, name, context, namespace, ports, labels)

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderKubernetes, forwarder.forwardType)
	assert.Equal(t, name, forwarder.name)
	assert.Equal(t, context, forwarder.context)
	assert.Equal(t, namespace, forwarder.namespace)
	assert.Equal(t, ports, forwarder.ports)

	assert.Len(t, forwarder.portForwarders, 0)
	assert.Len(t, forwarder.deployments, 0)
}
