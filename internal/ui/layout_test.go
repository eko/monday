package ui

import (
	"testing"

	"github.com/jroimartin/gocui"

	"github.com/stretchr/testify/assert"
)

func TestNewLayout(t *testing.T) {
	// When
	layout := NewLayout()
	layout.gui.Close()

	// Then
	assert.IsType(t, new(Layout), layout)
	assert.IsType(t, new(gocui.Gui), layout.gui)
}

func TestInit(t *testing.T) {
	// Given
	layout := NewLayout()
	layout.gui.Close()

	// When
	layout.Init()

	// Then
	assert.IsType(t, new(View), layout.statusView)
	assert.IsType(t, new(View), layout.fullscreenView)
	assert.IsType(t, new(View), layout.logsView)
	assert.IsType(t, new(View), layout.forwardsView)
	assert.IsType(t, new(View), layout.proxyView)
}

func TestGetGui(t *testing.T) {
	// Given
	layout := NewLayout()
	layout.gui.Close()

	layout.Init()

	// When
	result := layout.GetGui()

	// Then
	assert.IsType(t, new(gocui.Gui), result)
}

func TestGetLogsView(t *testing.T) {
	// Given
	layout := NewLayout()
	layout.gui.Close()

	layout.Init()

	// When
	result := layout.GetLogsView()

	// Then
	assert.IsType(t, new(View), result)

	assert.Equal(t, "logs", result.GetName())
	assert.Equal(t, " Logs ", result.GetTitle())
}

func TestGetForwardsView(t *testing.T) {
	// Given
	layout := NewLayout()
	layout.gui.Close()

	layout.Init()

	// When
	result := layout.GetForwardsView()

	// Then
	assert.IsType(t, new(View), result)

	assert.Equal(t, "forwards", result.GetName())
	assert.Equal(t, " Forwards ", result.GetTitle())
}

func TestGetProxyView(t *testing.T) {
	// Given
	layout := NewLayout()
	layout.gui.Close()

	layout.Init()

	// When
	result := layout.GetProxyView()

	// Then
	assert.IsType(t, new(View), result)

	assert.Equal(t, "proxy", result.GetName())
	assert.Equal(t, " Proxy ", result.GetTitle())
}

func TestGetStatusView(t *testing.T) {
	// Given
	layout := NewLayout()
	layout.gui.Close()

	layout.Init()

	// When
	result := layout.GetStatusView()

	// Then
	assert.IsType(t, new(View), result)

	assert.Equal(t, "status", result.GetName())
}
