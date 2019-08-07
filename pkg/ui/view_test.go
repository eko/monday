package ui

import (
	"testing"

	"github.com/jroimartin/gocui"

	"github.com/stretchr/testify/assert"
)

func TestNewView(t *testing.T) {
	// Given
	gocuiView := &gocui.View{}

	// When
	view := NewView("test-view", "Test View", gocuiView)

	// Then
	assert.IsType(t, new(View), view)

	assert.Equal(t, "test-view", view.GetName())
	assert.Equal(t, "Test View", view.GetTitle())
	assert.Equal(t, gocuiView, view.GetView())
}
