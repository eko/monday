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
	v := NewView("test-view", "Test View", gocuiView)

	// Then
	assert.IsType(t, new(view), v)
	assert.Implements(t, new(View), v)

	assert.Equal(t, "test-view", v.GetName())
	assert.Equal(t, "Test View", v.GetTitle())
	assert.Equal(t, gocuiView, v.GetView())
}
