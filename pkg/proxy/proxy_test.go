// +build !ci

package proxy

import (
	"testing"

	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/ui"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewProxy(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hostfileMock := hostfile.NewMockHostfile(ctrl)

	view := ui.NewMockView(ctrl)

	// When
	p := NewProxy(view, hostfileMock)

	// Then
	assert.IsType(t, new(proxy), p)
	assert.Implements(t, new(Proxy), p)

	assert.Len(t, p.ProxyForwards, 0)
	assert.Equal(t, p.latestPort, "9400")
	assert.Equal(t, p.ipLastByte, 1)
}

func TestAddProxyForward(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pf := NewProxyForward("test", "hostname.svc.local", "", "8080", "8080")

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost("127.1.2.1", "hostname.svc.local").Return(nil)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.1.2.1", "9401")

	proxy := NewProxy(view, hostfileMock)

	// When
	proxy.AddProxyForward("test", pf)

	// Then
	assert.Len(t, proxy.ProxyForwards, 1)
	assert.Equal(t, proxy.latestPort, "9401")
	assert.Equal(t, proxy.ipLastByte, 2)
}

func TestAddProxyForwardWhenMultiple(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name        string
		hostname    string
		localPort   string
		forwardPort string
	}{
		{name: "test", hostname: "hostname.svc.local", localPort: "8080", forwardPort: "8081"},
		{name: "test-2", hostname: "hostname2.svc.local", localPort: "8080", forwardPort: "8081"},
		{name: "test-2", hostname: "hostname3.svc.local", localPort: "8081", forwardPort: "8082"},
	}

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost("127.1.2.1", "hostname.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("127.1.2.2", "hostname2.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("127.1.2.2", "hostname3.svc.local").Return(nil)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.1.2.1", "9401")
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname2.svc.local", "127.1.2.2", "9402")
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname3.svc.local", "127.1.2.2", "9403")

	proxy := NewProxy(view, hostfileMock)

	// When
	for _, testCase := range testCases {
		pf := NewProxyForward(testCase.name, testCase.hostname, "", testCase.localPort, testCase.forwardPort)
		proxy.AddProxyForward(testCase.name, pf)
	}

	// Then
	assert.Len(t, proxy.ProxyForwards, 2)
	assert.Equal(t, proxy.latestPort, "9403")
}

func TestListen(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pf := NewProxyForward("test", "hostname.svc.local", "", "8080", "8080")

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost("127.1.2.1", "hostname.svc.local").Return(nil)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.1.2.1", "9401")
	view.EXPECT().Writef("🔌  Proxifying %s locally (%s:%s) <-> forwarding to %s:%s\n", "hostname.svc.local", "127.1.2.1", "8080", "127.0.0.1", "9401")

	proxy := NewProxy(view, hostfileMock)
	proxy.AddProxyForward("test", pf)

	// When
	err := proxy.Listen()

	// Then
	assert.Nil(t, err)

	assert.Len(t, proxy.ProxyForwards, 1)
	assert.Equal(t, proxy.latestPort, "9401")
	assert.Equal(t, proxy.ipLastByte, 2)
}
