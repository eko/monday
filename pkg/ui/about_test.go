package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestModelShowsAboutOverlay(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	layout.GetForwardStatuses().Register("api-forward", "kubernetes", []string{"8080:8080"})

	model := newModel(
		"my-project",
		"v2.0.0",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// When: pressing "?"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	rendered := model.View()

	// Then: the about overlay is displayed
	assert.Contains(t, rendered, "Vincent Composieux")
	assert.Contains(t, rendered, "monday@composieux.fr")
	assert.Contains(t, rendered, "https://github.com/eko/monday")
	assert.Contains(t, rendered, "v2.0.0")
	assert.Contains(t, rendered, "press any key to close")

	// The panes are hidden behind the overlay
	assert.NotContains(t, rendered, "Forwards & Proxy")

	// When: pressing any key
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	rendered = model.View()

	// Then: back to the normal display
	assert.NotContains(t, rendered, "Vincent Composieux")
	assert.Contains(t, rendered, "Forwards & Proxy")
}

func TestModelAboutOverlayCtrlCQuits(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	model := newModel("my-project", "", []*view{layout.logsView}, layout.forwardStatuses, layout.proxyStatuses, &layout.dirty)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// When
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Then
	assert.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd())
}
