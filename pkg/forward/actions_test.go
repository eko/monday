package forward

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// recordingView is a thread-safe ui.View capturing everything written to it
type recordingView struct {
	mux     sync.Mutex
	content strings.Builder
}

func (v *recordingView) GetName() string {
	return "recording"
}

func (v *recordingView) Write(str string) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.content.WriteString(str)
}

func (v *recordingView) Writef(str string, args ...interface{}) {
	v.Write(fmt.Sprintf(str, args...))
}

func (v *recordingView) String() string {
	v.mux.Lock()
	defer v.mux.Unlock()

	return v.content.String()
}

// fakeLogStreamer writes a line then blocks until the context is canceled
type fakeLogStreamer struct{}

func (f *fakeLogStreamer) StreamPodLogs(ctx context.Context, out io.Writer) error {
	_, _ = out.Write([]byte("hello from the pod\n"))

	<-ctx.Done()

	return ctx.Err()
}

func newActionsTestForwarder(t *testing.T) *forwarder {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef(gomock.Any(), gomock.Any()).AnyTimes()

	proxyfier := proxy.NewMockProxy(ctrl)

	return NewForwarder(view, ui.NewStatuses(nil), proxyfier, &config.Project{Name: "test"})
}

func TestReconnectAction(t *testing.T) {
	// Given
	f := newActionsTestForwarder(t)

	reconnected := 0
	f.registerReconnecter("api-forward", func() error {
		reconnected++
		return nil
	})
	f.registerReconnecter("api-forward", func() error {
		reconnected++
		return nil
	})

	// When
	err := f.Reconnect("api-forward")

	// Then: every connection of the forward has been closed
	assert.Nil(t, err)
	assert.Equal(t, 2, reconnected)

	// An unknown forward returns an explicit error
	assert.NotNil(t, f.Reconnect("unknown-forward"))
}

func TestTogglePauseAction(t *testing.T) {
	// Given
	f := newActionsTestForwarder(t)

	reconnected := 0
	f.registerReconnecter("api-forward", func() error {
		reconnected++
		return nil
	})

	// When: pausing
	paused, err := f.TogglePause("api-forward")

	// Then: the connection is closed and the forward marked as paused
	assert.Nil(t, err)
	assert.True(t, paused)
	assert.True(t, f.pauseControlFor("api-forward").isPaused())
	assert.Equal(t, 1, reconnected)

	// The connection loop wait is released on resume
	resume := f.pauseControlFor("api-forward").resumeChannel()

	// When: resuming
	paused, err = f.TogglePause("api-forward")

	// Then
	assert.Nil(t, err)
	assert.False(t, paused)
	assert.False(t, f.pauseControlFor("api-forward").isPaused())

	select {
	case <-resume:
	case <-time.After(time.Second):
		t.Fatal("the resume channel should have been closed")
	}

	// An unknown forward returns an explicit error
	_, err = f.TogglePause("unknown-forward")
	assert.NotNil(t, err)
}

func TestToggleLogsAction(t *testing.T) {
	// Given
	f := newActionsTestForwarder(t)
	f.statuses.Register("api-forward", "kubernetes", []string{"8080:8080"})

	logsView := &recordingView{}
	f.SetLogsView(logsView)
	f.registerLogStreamer("api-forward", &fakeLogStreamer{})

	// When: starting the pod logs stream
	streaming, err := f.ToggleLogs("api-forward")

	// Then: the stream is active and identified on the forward status
	assert.Nil(t, err)
	assert.True(t, streaming)
	assert.True(t, f.statuses.(*ui.Statuses).Snapshot()[0].LogsStreaming)

	assert.Eventually(t, func() bool {
		return strings.Contains(logsView.String(), "hello from the pod")
	}, 2*time.Second, 10*time.Millisecond)

	// When: stopping it
	streaming, err = f.ToggleLogs("api-forward")

	// Then
	assert.Nil(t, err)
	assert.False(t, streaming)
	assert.False(t, f.statuses.(*ui.Statuses).Snapshot()[0].LogsStreaming)

	assert.Eventually(t, func() bool {
		return strings.Contains(logsView.String(), "Stopped streaming pod logs")
	}, 2*time.Second, 10*time.Millisecond)

	// A forward without kubernetes pod returns an explicit error
	_, err = f.ToggleLogs("unknown-forward")
	assert.NotNil(t, err)
}
