package proxy

import (
	"testing"

	mocks "github.com/eko/monday/internal/tests/mocks/hostfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewProxy(t *testing.T) {
	// Given

	hostfileMock := &mocks.HostfileInterface{}

	// When
	proxy := NewProxy(hostfileMock)

	// Then
	assert.IsType(t, new(Proxy), proxy)

	assert.Len(t, proxy.ProxyForwards, 0)
	assert.Equal(t, proxy.latestPort, "9400")
	assert.Equal(t, proxy.ipLastByte, 1)
}

func TestAddProxyForward(t *testing.T) {
	// Given
	pf := NewProxyForward("test", "hostname.svc.local", "", "8080", "8080")

	hostfileMock := &mocks.HostfileInterface{}
	hostfileMock.On("AddHost", mock.AnythingOfType("string"), "hostname.svc.local").Return(nil)

	proxy := NewProxy(hostfileMock)

	// When
	proxy.AddProxyForward("test", pf)

	// Then
	assert.Len(t, proxy.ProxyForwards, 1)
	assert.Equal(t, proxy.latestPort, "9401")
	assert.Equal(t, proxy.ipLastByte, 2)
}

func TestAddProxyForwardWhenMultiple(t *testing.T) {
	// Given
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

	hostfileMock := &mocks.HostfileInterface{}
	hostfileMock.ExpectedCalls = []*mock.Call{
		&mock.Call{
			Method: "AddHost",
			Arguments: mock.Arguments{
				mock.AnythingOfType("string"), "hostname.svc.local",
			},
			ReturnArguments: mock.Arguments{nil},
		},
		&mock.Call{
			Method: "AddHost",
			Arguments: mock.Arguments{
				mock.AnythingOfType("string"), "hostname2.svc.local",
			},
			ReturnArguments: mock.Arguments{nil},
		},
		&mock.Call{
			Method: "AddHost",
			Arguments: mock.Arguments{
				mock.AnythingOfType("string"), "hostname3.svc.local",
			},
			ReturnArguments: mock.Arguments{nil},
		},
	}

	proxy := NewProxy(hostfileMock)

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
	pf := NewProxyForward("test", "hostname.svc.local", "", "8080", "8080")

	hostfileMock := &mocks.HostfileInterface{}
	hostfileMock.On("AddHost", mock.AnythingOfType("string"), "hostname.svc.local").Return(nil)

	proxy := NewProxy(hostfileMock)
	proxy.AddProxyForward("test", pf)

	// When
	err := proxy.Listen()

	// Then
	assert.Nil(t, err)

	assert.Len(t, proxy.ProxyForwards, 1)
	assert.Equal(t, proxy.latestPort, "9401")
	assert.Equal(t, proxy.ipLastByte, 2)
}
