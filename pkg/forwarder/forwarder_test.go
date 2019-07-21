package forwarder

import (
	"fmt"
	"testing"

	"github.com/eko/monday/internal/config"
	mocks "github.com/eko/monday/internal/tests/mocks/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestForwardAll(t *testing.T) {
	// Given
	proxy := &mocks.ProxyInterface{}
	proxy.
		On("AddProxyForward", mock.AnythingOfType("string"), mock.AnythingOfType("*proxy.ProxyForward")).
		Times(2)
	proxy.On("Listen").Once().Return(nil)

	project := &config.Project{
		Name: "My project name",
		Forwards: []*config.Forward{
			&config.Forward{
				Name: "test-ssh-forward",
				Type: "ssh",
				Values: config.ForwardValues{
					Remote: "root@acme.tld",
					Ports:  []string{"8080:8080"},
				},
			},
		},
	}

	forwarder := NewForwarder(proxy, project)

	// When
	forwarder.ForwardAll()

	// Then
	assert.Len(t, forwarder.forwards, 1)

	for _, forward := range project.Forwards {
		if v, ok := forwarder.forwarders.Load(forward.Name); ok {
			assert.Len(t, v.([]ForwarderTypeInterface), 1)
		} else {
			t.Fatal(fmt.Sprintf("No forwarder found for forward named '%s'", forward.Name))
		}
	}
}
