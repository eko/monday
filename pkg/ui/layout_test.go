package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewLayout(t *testing.T) {
	// When
	layout := NewLayout(true)
	layout.Init()

	// Then
	assert.IsType(t, new(Layout), layout)
	assert.True(t, layout.IsUIEnabled())

	assert.IsType(t, new(view), layout.logsView)
	assert.IsType(t, new(view), layout.forwardsView)
	assert.IsType(t, new(view), layout.proxyView)

	assert.False(t, layout.logsView.stdout)
}

func TestInitWhenUINotEnabled(t *testing.T) {
	// When
	layout := NewLayout(false)
	layout.Init()

	// Then
	assert.False(t, layout.IsUIEnabled())

	assert.True(t, layout.logsView.stdout)
	assert.True(t, layout.forwardsView.stdout)
	assert.True(t, layout.proxyView.stdout)
}

func TestGetViews(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	testCases := []struct {
		view          *view
		expectedName  string
		expectedTitle string
	}{
		{view: layout.GetLogsView(), expectedName: "logs", expectedTitle: "Logs"},
		{view: layout.GetForwardsView(), expectedName: "activity", expectedTitle: "Forwards & Proxy"},
		{view: layout.GetProxyView(), expectedName: "activity", expectedTitle: "Forwards & Proxy"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.expectedName, func(t *testing.T) {
			assert.Equal(t, testCase.expectedName, testCase.view.GetName())
			assert.Equal(t, testCase.expectedTitle, testCase.view.GetTitle())
		})
	}
}

func TestRunWhenUINotEnabled(t *testing.T) {
	// Given
	layout := NewLayout(false)
	layout.Init()

	// When
	err := layout.Run()

	// Then
	assert.Nil(t, err)
}

func TestModelRendersViewContent(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()
	layout.SetProject("my-project")

	layout.GetLogsView().Write("hello from logs\n")
	layout.GetForwardsView().Write("kubernetes forward ready\n")
	layout.GetProxyView().Write("proxy listening\n")

	model := newModel(
		"my-project",
		"v1.0.0",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)

	// When
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rendered := model.View()

	// Then: forward/proxy events are hidden by default in the combined pane
	assert.Contains(t, rendered, "my-project")
	assert.Contains(t, rendered, "hello from logs")
	assert.Contains(t, rendered, "Logs")
	assert.Contains(t, rendered, "Forwards & Proxy")
	assert.NotContains(t, rendered, "kubernetes forward ready")
	assert.NotContains(t, rendered, "proxy listening")

	// When: focusing the combined pane and toggling its events log
	model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	rendered = model.View()

	// Then
	assert.Contains(t, rendered, "kubernetes forward ready")
	assert.Contains(t, rendered, "proxy listening")
}

func TestModelSwitchesFocus(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// When
	model.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Then
	assert.Equal(t, 1, model.focus)

	// When going backward twice: cycles across the two panes
	model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	// Then
	assert.Equal(t, 1, model.focus)
}

func TestModelFiltersFocusedPane(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	layout.GetLogsView().Write("an error occured\n")
	layout.GetLogsView().Write("everything is fine\n")

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// When: typing "/error" then enter
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("error")})
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	rendered := model.View()

	// Then
	assert.Contains(t, rendered, "an error occured")
	assert.NotContains(t, rendered, "everything is fine")
}

func TestModelQuits(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	model := newModel("my-project", "", []*view{layout.logsView}, layout.forwardStatuses, layout.proxyStatuses, &layout.dirty)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// When
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Then
	assert.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd())
}
