package forward

import (
	"context"
	"fmt"
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewForwarder(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	proxyfier := proxy.NewMockProxy(ctrl)

	project := &config.Project{
		Name: "My project name",
		Forwards: []*config.Forward{
			{
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

	view := ui.NewMockView(ctrl)

	// When
	f := NewForwarder(view, proxyfier, project)

	// Then
	assert.IsType(t, new(forwarder), f)
	assert.Implements(t, new(Forwarder), f)

	assert.Equal(t, proxyfier, f.proxy)
	assert.Equal(t, project.Forwards, f.forwards)
}

func TestForwardAll(t *testing.T) {
	// Given
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	proxyForward := proxy.NewProxyForward("test-ssh-forward", "", "", "8080", "8080")

	proxyfier := proxy.NewMockProxy(ctrl)
	proxyfier.EXPECT().AddProxyForward("test-ssh-forward", proxyForward)
	proxyfier.EXPECT().Listen().Return(nil).AnyTimes()

	project := &config.Project{
		Name: "My project name",
		Forwards: []*config.Forward{
			{
				Name: "test-ssh-forward",
				Type: "ssh",
				Values: config.ForwardValues{
					Remote: "root@acme.tld",
					Ports:  []string{"8080:8080"},
				},
			},
		},
	}

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("📡  Forwarding '%s' over %s...\n", "test-ssh-forward", "ssh")

	forwarder := NewForwarder(view, proxyfier, project)

	// When
	forwarder.ForwardAll(ctx)

	// Then
	assert.Len(t, forwarder.forwards, 1)

	for _, forward := range project.Forwards {
		if v, ok := forwarder.forwarders.Load(forward.Name); ok {
			assert.Len(t, v.([]ForwarderType), 1)
		} else {
			t.Fatal(fmt.Sprintf("No forwarder found for forward named '%s'", forward.Name))
		}
	}
}

func TestForwardRemoteSSH(t *testing.T) {
	// Given
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	proxy := proxy.NewMockProxy(ctrl)
	proxy.EXPECT().Listen().Return(nil).AnyTimes()

	project := &config.Project{
		Name: "My project name",
		Forwards: []*config.Forward{
			{
				Name: "test-ssh-forward",
				Type: config.ForwarderSSHRemote,
				Values: config.ForwardValues{
					Remote: "root@acme.tld",
					Ports:  []string{"8080:8080"},
				},
			},
		},
	}

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("📡  Forwarding '%s' over %s...\n", "test-ssh-forward", "ssh-remote")

	forwarder := NewForwarder(view, proxy, project)

	// When
	forwarder.ForwardAll(ctx)

	// Then
	assert.Len(t, forwarder.forwards, 1)

	for _, forward := range project.Forwards {
		if v, ok := forwarder.forwarders.Load(forward.Name); ok {
			assert.Len(t, v.([]ForwarderType), 1)
		} else {
			t.Fatal(fmt.Sprintf("No forwarder found for forward named '%s'", forward.Name))
		}
	}
}
