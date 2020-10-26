package kubernetes

import (
	"testing"

	uimocks "github.com/eko/monday/internal/tests/mocks/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewLogstreamer(t *testing.T) {
	// Given
	podName := "my-test-pod"

	view := &uimocks.ViewInterface{}

	// When
	streamer := NewLogstreamer(view, podName)

	// Then
	assert.IsType(t, new(Logstreamer), streamer)

	assert.Equal(t, podName, streamer.podName)
}

func TestWrite(t *testing.T) {
	// When
	view := &uimocks.ViewInterface{}
	view.On("Writef", "%s %s", "my-test-pod", "This is a sample log from my unit test")

	streamer := NewLogstreamer(view, "my-test-pod")

	// Then
	streamer.Write([]byte("This is a sample log from my unit test"))
}
