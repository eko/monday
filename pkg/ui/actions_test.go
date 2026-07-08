package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// stubActions records the actions triggered by the terminal UI
type stubActions struct {
	reconnected []string
	paused      map[string]bool
	logs        map[string]bool
}

func newStubActions() *stubActions {
	return &stubActions{
		paused: make(map[string]bool),
		logs:   make(map[string]bool),
	}
}

func (s *stubActions) Reconnect(name string) error {
	s.reconnected = append(s.reconnected, name)
	return nil
}

func (s *stubActions) TogglePause(name string) (bool, error) {
	s.paused[name] = !s.paused[name]
	return s.paused[name], nil
}

func (s *stubActions) ToggleLogs(name string) (bool, error) {
	s.logs[name] = !s.logs[name]
	return s.logs[name], nil
}

func newActionsTestModel(t *testing.T) (*model, *stubActions) {
	layout := NewLayout(true)
	layout.Init()

	layout.GetForwardStatuses().Register("api-forward", "kubernetes", []string{"8080:8080"})
	layout.GetForwardStatuses().Register("database-forward", "kubernetes", []string{"5432:5432"})
	layout.GetProxyStatuses().Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")

	actions := newStubActions()
	layout.SetActions(actions)

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.actions = layout.actions
	model.Update(tea.WindowSizeMsg{Width: 130, Height: 40})

	return model, actions
}

func TestModelActionKeys(t *testing.T) {
	// Given
	model, actions := newActionsTestModel(t)

	// When: pressing 'r' on the selected row (api-forward is first)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Then
	assert.Equal(t, []string{"api-forward"}, actions.reconnected)

	// When: pressing 'p' twice
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	assert.True(t, actions.paused["api-forward"])

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	assert.False(t, actions.paused["api-forward"])

	// When: pressing 'L' for pod logs
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	assert.True(t, actions.logs["api-forward"])

	// When: moving the selection and reconnecting the second forward
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Then
	assert.Equal(t, []string{"api-forward", "database-forward"}, actions.reconnected)
}

func TestModelIdentifiesStreamingLogsRows(t *testing.T) {
	// Given
	model, _ := newActionsTestModel(t)

	// When: the pod logs streaming is reported for the selected forward
	model.statuses.SetLogsStreaming("api-forward", true)
	model.Update(refreshMsg{})
	rendered := model.View()

	// Then: the row is identified with a logs marker
	assert.Contains(t, rendered, "📜 logs")

	// When: the streaming stops
	model.statuses.SetLogsStreaming("api-forward", false)
	model.Update(refreshMsg{})
	rendered = model.View()

	// Then
	assert.NotContains(t, rendered, "📜 logs")
}

func TestModelActionFeedbackInEvents(t *testing.T) {
	// Given
	model, _ := newActionsTestModel(t)

	// When: reconnecting then displaying the events log
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then
	assert.Contains(t, model.View(), "Reconnection of 'api-forward' requested")
}

func TestRowAddress(t *testing.T) {
	testCases := []struct {
		name        string
		row         combinedRow
		wantAddress string
		wantOk      bool
	}{
		{
			name: "hostname and port",
			row: combinedRow{
				proxies: []ProxyStatus{{Hostname: "api.svc.local", LocalPort: "8080"}},
			},
			wantAddress: "api.svc.local:8080",
			wantOk:      true,
		},
		{
			name: "hostname only",
			row: combinedRow{
				proxies: []ProxyStatus{{Hostname: "app.svc.local"}},
			},
			wantAddress: "app.svc.local",
			wantOk:      true,
		},
		{
			name:   "no proxy entry",
			row:    combinedRow{},
			wantOk: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			address, ok := rowAddress(testCase.row)

			assert.Equal(t, testCase.wantOk, ok)
			assert.Equal(t, testCase.wantAddress, address)
		})
	}
}
