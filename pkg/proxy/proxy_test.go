//go:build !ci
// +build !ci

package proxy

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/ui"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewProxy(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hostfileMock := hostfile.NewMockHostfile(ctrl)

	view := ui.NewMockView(ctrl)

	// When
	p := NewProxy(view, ui.NewProxyStatuses(), hostfileMock)

	// Then
	assert.IsType(t, new(proxy), p)
	assert.Implements(t, new(Proxy), p)

	assert.Len(t, p.ProxyForwards, 0)
	assert.Equal(t, p.latestPort, "9400")

	assert.Equal(t, p.lastIpByteA, byte(127))
	assert.Equal(t, p.lastIpByteB, byte(0))
	assert.Equal(t, p.lastIpByteC, byte(1))
	assert.Equal(t, p.lastIpByteD, byte(0))
}

func TestAddProxyForward(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pf := NewProxyForward("test", "hostname.svc.local", "", "8080", "8080")

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost("127.0.1.1", "hostname.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("::127:0:1:1", "hostname.svc.local").Return(nil)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.0.1.1", "9401")

	proxy := NewProxy(view, ui.NewProxyStatuses(), hostfileMock)

	// When
	proxy.AddProxyForward("test", pf)

	// Then
	assert.Len(t, proxy.ProxyForwards, 1)
	assert.Equal(t, proxy.latestPort, "9401")

	assert.Equal(t, proxy.lastIpByteA, byte(127))
	assert.Equal(t, proxy.lastIpByteB, byte(0))
	assert.Equal(t, proxy.lastIpByteC, byte(1))
	assert.Equal(t, proxy.lastIpByteD, byte(1))
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
		{name: "test1", hostname: "hostname.svc.local", localPort: "8080", forwardPort: "8081"},
		{name: "test1-2", hostname: "hostname.svc.local", localPort: "8081", forwardPort: "8082"},
		{name: "test2", hostname: "hostname2.svc.local", localPort: "8080", forwardPort: "8081"},
		{name: "test2", hostname: "hostname3.svc.local", localPort: "8081", forwardPort: "8082"},
	}

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost("127.0.1.1", "hostname.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("::127:0:1:1", "hostname.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("127.0.1.2", "hostname2.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("::127:0:1:2", "hostname2.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("127.0.1.3", "hostname3.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("::127:0:1:3", "hostname3.svc.local").Return(nil)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.0.1.1", "9401")
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.0.1.1", "9402")
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname2.svc.local", "127.0.1.2", "9403")
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname3.svc.local", "127.0.1.3", "9404")

	proxy := NewProxy(view, ui.NewProxyStatuses(), hostfileMock)

	// When
	for _, testCase := range testCases {
		pf := NewProxyForward(testCase.name, testCase.hostname, "", testCase.localPort, testCase.forwardPort)
		proxy.AddProxyForward(testCase.name, pf)
	}

	// Then
	assert.Len(t, proxy.ProxyForwards, 3)
	assert.Equal(t, proxy.latestPort, "9404")
}

func TestListen(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pf := NewProxyForward("test", "hostname.svc.local", "", "8080", "8080")

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost("127.0.1.1", "hostname.svc.local").Return(nil)
	hostfileMock.EXPECT().AddHost("::127:0:1:1", "hostname.svc.local").Return(nil)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("✅  Successfully mapped hostname '%s' with IP '%s' and port %s\n", "hostname.svc.local", "127.0.1.1", "9401")
	view.EXPECT().Writef("🔌  Proxifying %s locally (%s:%s) <-> forwarding to %s:%s\n", "hostname.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")

	proxy := NewProxy(view, ui.NewProxyStatuses(), hostfileMock)
	proxy.AddProxyForward("test", pf)

	// When
	err := proxy.Listen()

	// Then
	assert.Nil(t, err)

	assert.Len(t, proxy.ProxyForwards, 1)
	assert.Equal(t, proxy.latestPort, "9401")

	assert.Equal(t, proxy.lastIpByteA, byte(127))
	assert.Equal(t, proxy.lastIpByteB, byte(0))
	assert.Equal(t, proxy.lastIpByteC, byte(1))
	assert.Equal(t, proxy.lastIpByteD, byte(1))
}

// TestListenConcurrentWithAddProxyForward is a regression test for a
// "concurrent map read and map write" crash: the runner registers proxy
// forwards while the forwarder concurrently starts listening
func TestListenConcurrentWithAddProxyForward(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hostfileMock := hostfile.NewMockHostfile(ctrl)
	hostfileMock.EXPECT().AddHost(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef(gomock.Any(), gomock.Any()).AnyTimes()

	proxy := NewProxy(view, ui.NewProxyStatuses(), hostfileMock)

	// When: adding forwards while listening concurrently
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()

			// Same hostname on purpose: the attributed IP is cached so no
			// network interface alias is created during tests
			pf := NewProxyForward("concurrent-app", "concurrent.svc.local", "", "", "")
			proxy.AddProxyForward("concurrent-app", pf)
		}()

		go func() {
			defer wg.Done()

			assert.Nil(t, proxy.Listen())
		}()
	}

	wg.Wait()

	// Then
	assert.Len(t, proxy.ProxyForwards["concurrent-app"], 20)
}

// TestProxifyConnectionCountsLiveTraffic verifies that bytes flowing through a
// proxied connection are metered while the connection is still open, so
// long-lived connections (databases, brokers...) display live traffic
func TestProxifyConnectionCountsLiveTraffic(t *testing.T) {
	// Given: a real TCP echo server as the forwarded target
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	go func() {
		conn, err := target.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buffer := make([]byte, 1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				return
			}
			if _, err := conn.Write(buffer[:n]); err != nil {
				return
			}
		}
	}()

	targetPort := strconv.Itoa(target.Addr().(*net.TCPAddr).Port)

	// Reserve a local port for the proxy listener
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	localPort := strconv.Itoa(reservation.Addr().(*net.TCPAddr).Port)
	reservation.Close()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef(gomock.Any(), gomock.Any()).AnyTimes()

	hostfileMock := hostfile.NewMockHostfile(ctrl)

	statuses := ui.NewProxyStatuses()
	statuses.Register("traffic-test", "traffic.svc.local", "127.0.0.1", localPort, "127.0.0.1", targetPort)

	proxy := NewProxy(view, statuses, hostfileMock)

	// Register the forward directly with a resolved local IP, bypassing the
	// network interface aliasing that requires root privileges
	pf := NewProxyForward("traffic-test", "traffic.svc.local", "127.0.0.1", localPort, targetPort)
	pf.SetLocalIP("127.0.0.1")
	proxy.ProxyForwards["traffic-test"] = []*ProxyForward{pf}

	assert.Nil(t, proxy.Listen())

	// When: sending and receiving data over a connection kept open
	client, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", localPort))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	message := []byte("hello, monday!")
	if _, err := client.Write(message); err != nil {
		t.Fatal(err)
	}

	echo := make([]byte, len(message))
	if _, err := io.ReadFull(client, echo); err != nil {
		t.Fatal(err)
	}

	// Then: traffic is visible before the connection is closed
	assert.Eventually(t, func() bool {
		snapshot := statuses.Snapshot()

		return snapshot[0].ActiveConnections == 1 &&
			snapshot[0].BytesSent >= int64(len(message)) &&
			snapshot[0].BytesReceived >= int64(len(message))
	}, 2*time.Second, 10*time.Millisecond)
}

func TestGetNextIPAddress(t *testing.T) {
	testCases := []struct {
		a byte
		b byte
		c byte
		d byte

		expectedA byte
		expectedB byte
		expectedC byte
		expectedD byte
	}{
		{ // Case incrementing d
			a: 127, b: 0, c: 0, d: 1,
			expectedA: 127, expectedB: 0, expectedC: 0, expectedD: 2,
		},
		{ // Case incrementing d to last byte
			a: 127, b: 0, c: 0, d: 254,
			expectedA: 127, expectedB: 0, expectedC: 0, expectedD: 255,
		},
		{ // Case incrementing c and reset d to 1
			a: 127, b: 0, c: 0, d: 255,
			expectedA: 127, expectedB: 0, expectedC: 1, expectedD: 1,
		},
		{ // Case incrementing d to last byte
			a: 127, b: 0, c: 1, d: 254,
			expectedA: 127, expectedB: 0, expectedC: 1, expectedD: 255,
		},
		{ // Case incrementing c and reset d to 1
			a: 127, b: 0, c: 1, d: 255,
			expectedA: 127, expectedB: 0, expectedC: 2, expectedD: 1,
		},
		{ // Case incrementing d to last byte
			a: 127, b: 0, c: 254, d: 254,
			expectedA: 127, expectedB: 0, expectedC: 254, expectedD: 255,
		},
		{ // Case incrementing c and reset d to 1
			a: 127, b: 0, c: 254, d: 255,
			expectedA: 127, expectedB: 0, expectedC: 255, expectedD: 1,
		},
		{ // Case incrementing d to last byte when c is already on latest byte
			a: 127, b: 0, c: 255, d: 254,
			expectedA: 127, expectedB: 0, expectedC: 255, expectedD: 255,
		},
		{ // Case incrementing b and reset c and d to last byte
			a: 127, b: 0, c: 255, d: 255,
			expectedA: 127, expectedB: 1, expectedC: 1, expectedD: 1,
		},
		{ // Case incrementing d to last byte when b and c are already on latest byte
			a: 127, b: 255, c: 255, d: 254,
			expectedA: 127, expectedB: 255, expectedC: 255, expectedD: 255,
		},
		{ // Reached max level
			a: 127, b: 255, c: 255, d: 255,
			expectedA: 127, expectedB: 255, expectedC: 255, expectedD: 255,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("Case %d", i), func(t *testing.T) {
			a, b, c, d := getNextIPAddress(testCase.a, testCase.b, testCase.c, testCase.d)

			assert.Equal(t, testCase.expectedA, a)
			assert.Equal(t, testCase.expectedB, b)
			assert.Equal(t, testCase.expectedC, c)
			assert.Equal(t, testCase.expectedD, d)
		})
	}
}
