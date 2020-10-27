package kubernetes

import (
	"testing"

	"github.com/eko/monday/pkg/ui"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewLogstreamer(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	podName := "my-test-pod"

	view := ui.NewMockView(ctrl)

	// When
	streamer := NewLogstreamer(view, podName)

	// Then
	assert.IsType(t, new(Logstreamer), streamer)

	assert.Equal(t, podName, streamer.podName)
}

func TestWrite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// When
	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("%s %s", "my-test-pod", "This is a sample log from my unit test")

	streamer := NewLogstreamer(view, "my-test-pod")

	// Then
	streamer.Write([]byte("This is a sample log from my unit test"))
}
