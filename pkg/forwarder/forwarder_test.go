package forwarder

import (
	"testing"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/internal/tests/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewForwarder(t *testing.T) {
	// Given
	proxy := &mocks.ProxyInterface{}

	project := &config.Project{
		Name: "My project name",
		Forwards: []*config.Forward{
			&config.Forward{
				Name: "test-kubernetes-forward",
				Type: "kubernetes",
				Values: config.ForwardValues{
					Namespace: "test",
					Labels: map[string]string{
						"app": "my-test-app",
					},
				},
			},
		},
	}

	// When
	forwarder := NewForwarder(proxy, project)

	// Then
	assert.IsType(t, new(Forwarder), forwarder)

	assert.Equal(t, proxy, forwarder.proxy)
	assert.Equal(t, project.Forwards, forwarder.forwards)
}

func newForwarder(project *config.Project) *Forwarder {
	proxy := &mocks.ProxyInterface{}

	return NewForwarder(proxy, project)
}
