package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestProxyStatusesRegisterAndConnections(t *testing.T) {
	// Given
	statuses := NewProxyStatuses()

	// When
	statuses.Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")
	statuses.SetState("api.svc.local", "8080", ProxyStateListening, "")

	statuses.ConnectionOpened("api.svc.local", "8080")
	statuses.ConnectionOpened("api.svc.local", "8080")

	// Traffic is recorded while connections are still open
	statuses.AddTraffic("api.svc.local", "8080", 2048, 512)
	statuses.AddTraffic("api.svc.local", "8080", 1024, 256)

	statuses.ConnectionClosed("api.svc.local", "8080")

	// Then
	snapshot, _ := statuses.snapshot()

	assert.Len(t, snapshot, 1)
	assert.Equal(t, ProxyStateListening, snapshot[0].State)
	assert.Equal(t, 1, snapshot[0].ActiveConnections)
	assert.Equal(t, 2, snapshot[0].TotalConnections)
	assert.Equal(t, int64(3072), snapshot[0].BytesReceived)
	assert.Equal(t, int64(768), snapshot[0].BytesSent)
}

func TestProxyStatusesAlphabeticalOrder(t *testing.T) {
	// Given
	statuses := NewProxyStatuses()

	// When: registering in a non-alphabetical order, with one hostname on two ports
	statuses.Register("postgres-forward", "postgres.svc.local", "127.0.1.3", "5432", "127.0.0.1", "9403")
	statuses.Register("cast-api-forward", "Cast-api.svc.local", "127.0.1.1", "8081", "127.0.0.1", "9402")
	statuses.Register("cast-api-forward", "Cast-api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")
	statuses.Register("kafka-forward", "kafka.svc.local", "127.0.1.2", "9092", "127.0.0.1", "9404")

	// Then: sorted by hostname (case-insensitively), then by local port
	snapshot, _ := statuses.snapshot()

	assert.Equal(t, "Cast-api.svc.local", snapshot[0].Hostname)
	assert.Equal(t, "8080", snapshot[0].LocalPort)
	assert.Equal(t, "Cast-api.svc.local", snapshot[1].Hostname)
	assert.Equal(t, "8081", snapshot[1].LocalPort)
	assert.Equal(t, "kafka.svc.local", snapshot[2].Hostname)
	assert.Equal(t, "postgres.svc.local", snapshot[3].Hostname)
}

func TestProxyStatusesHostnameOnlyIsMapped(t *testing.T) {
	// Given
	statuses := NewProxyStatuses()

	// When: a local application only maps a hostname, without a proxied port
	statuses.Register("app", "app.svc.local", "127.0.1.2", "", "", "")

	// Then
	snapshot, _ := statuses.snapshot()

	assert.Equal(t, ProxyStateMapped, snapshot[0].State)
}

func TestProxyStatusesConnectionFailed(t *testing.T) {
	// Given
	statuses := NewProxyStatuses()
	statuses.Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")

	// When
	statuses.ConnectionFailed("api.svc.local", "8080", "connection refused")
	statuses.ConnectionFailed("api.svc.local", "8080", "connection refused")

	// Then
	snapshot, _ := statuses.snapshot()

	assert.Equal(t, 2, snapshot[0].Errors)
	assert.Equal(t, "connection refused", snapshot[0].Message)
}

func TestRenderProxyDetail(t *testing.T) {
	testCases := []struct {
		name     string
		status   ProxyStatus
		contains []string
	}{
		{
			name: "route, connections and traffic",
			status: ProxyStatus{
				Hostname: "api.svc.local", LocalIP: "127.0.1.1", LocalPort: "8080",
				TargetHost: "127.0.0.1", TargetPort: "9401", State: ProxyStateListening,
				ActiveConnections: 2, TotalConnections: 147,
				BytesReceived: 12898796, BytesSent: 1258291,
			},
			contains: []string{
				"api.svc.local", "127.0.1.1:8080 → 127.0.0.1:9401",
				"2/147 conn(s)", "↓12.3MB", "↑1.2MB", "listening",
			},
		},
		{
			name: "hostname-only mapping",
			status: ProxyStatus{
				Hostname: "app.svc.local", LocalIP: "127.0.1.2", State: ProxyStateMapped,
			},
			contains: []string{"app.svc.local", "hostname only", "mapped"},
		},
		{
			name: "connection failures are highlighted",
			status: ProxyStatus{
				Hostname: "api.svc.local", LocalIP: "127.0.1.1", LocalPort: "8080",
				TargetHost: "127.0.0.1", TargetPort: "9401", State: ProxyStateListening,
				Errors: 3,
			},
			contains: []string{"⚠3"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			line := renderProxyDetail(testCase.status, 160)

			for _, expected := range testCase.contains {
				assert.Contains(t, line, expected)
			}
		})
	}
}

func TestModelRendersProxyOnlyRow(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	layout.GetProxyStatuses().Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")
	layout.GetProxyStatuses().SetState("api.svc.local", "8080", ProxyStateListening, "")
	layout.GetProxyView().Write("🔌  Proxifying api.svc.local...\n")

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)

	// When
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rendered := model.View()

	// Then: the proxy entry without a forward appears as its own row, its
	// events being hidden until the log is toggled
	assert.Contains(t, rendered, "api-forward")
	assert.Contains(t, rendered, "listening")
	assert.NotContains(t, rendered, "Proxifying api.svc.local")

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	rendered = model.View()

	assert.Contains(t, rendered, "Proxifying api.svc.local")
}

func TestHumanBytes(t *testing.T) {
	testCases := []struct {
		count int64
		want  string
	}{
		{count: 0, want: "0B"},
		{count: 512, want: "512B"},
		{count: 2048, want: "2.0KB"},
		{count: 12898796, want: "12.3MB"},
		{count: 5368709120, want: "5.0GB"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.want, func(t *testing.T) {
			assert.Equal(t, testCase.want, humanBytes(testCase.count))
		})
	}
}
