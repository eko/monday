package forward

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
)

// logStreamRetryDelay is the delay before re-attaching a pod logs stream after
// it ends (e.g. when the pod is redeployed)
const logStreamRetryDelay = 2 * time.Second

// PodLogStreamer streams the logs of the pod behind a forward
type PodLogStreamer interface {
	StreamPodLogs(
		ctx context.Context,
		out io.Writer,
	) error
}

// pauseControl holds the pause state of a forward, the connection loop waits
// on the resume channel while paused
type pauseControl struct {
	mux    sync.Mutex
	paused bool
	resume chan struct{}
}

func (p *pauseControl) isPaused() bool {
	p.mux.Lock()
	defer p.mux.Unlock()

	return p.paused
}

func (p *pauseControl) resumeChannel() chan struct{} {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.resume == nil {
		p.resume = make(chan struct{})
	}

	return p.resume
}

func (p *pauseControl) setPaused(paused bool) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.paused = paused

	if !paused && p.resume != nil {
		close(p.resume)
		p.resume = nil
	}
}

// pauseControlFor returns the pause control of the given forward name
func (f *forwarder) pauseControlFor(name string) *pauseControl {
	control, _ := f.pauses.LoadOrStore(name, &pauseControl{})
	return control.(*pauseControl)
}

// registerReconnecter registers the function closing the active connection of
// a forward, so it can be reconnected on demand
func (f *forwarder) registerReconnecter(name string, reconnect func() error) {
	reconnecters := make([]func() error, 0)

	if existing, ok := f.reconnecters.Load(name); ok {
		reconnecters = existing.([]func() error)
	}

	f.reconnecters.Store(name, append(reconnecters, reconnect))
}

// Reconnect closes the currently active connections of the given forward so
// fresh ones are established immediately
func (f *forwarder) Reconnect(name string) error {
	reconnecters, ok := f.reconnecters.Load(name)
	if !ok {
		return fmt.Errorf("no active connection found for forward '%s'", name)
	}

	f.statuses.Set(name, ui.StateReconnecting, "reconnection requested")

	for _, reconnect := range reconnecters.([]func() error) {
		if err := reconnect(); err != nil {
			return err
		}
	}

	return nil
}

// TogglePause pauses the given forward (closing its connections) or resumes it,
// returning whether it is now paused
func (f *forwarder) TogglePause(name string) (bool, error) {
	if _, ok := f.reconnecters.Load(name); !ok {
		return false, fmt.Errorf("no active connection found for forward '%s'", name)
	}

	control := f.pauseControlFor(name)

	if control.isPaused() {
		control.setPaused(false)
		return false, nil
	}

	control.setPaused(true)

	// Close the current connections: the connection loops will wait for resume
	return true, f.Reconnect(name)
}

// SetLogsView sets the view receiving the streamed pod logs
func (f *forwarder) SetLogsView(view ui.View) {
	f.logsView = view
}

// registerLogStreamer registers the pod logs source of a forward
func (f *forwarder) registerLogStreamer(name string, streamer PodLogStreamer) {
	f.logStreamers.Store(name, streamer)
}

// ToggleLogs starts streaming the logs of the pod behind the given forward
// into the logs view, or stops the stream when already active. It returns
// whether logs are now being streamed
func (f *forwarder) ToggleLogs(name string) (bool, error) {
	if cancel, ok := f.logStreams.LoadAndDelete(name); ok {
		cancel.(context.CancelFunc)()
		f.statuses.SetLogsStreaming(name, false)

		return false, nil
	}

	streamer, ok := f.logStreamers.Load(name)
	if !ok {
		return false, fmt.Errorf("pod logs are only available for kubernetes forwards")
	}

	if f.logsView == nil {
		return false, fmt.Errorf("no logs view available to stream pod logs")
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.logStreams.Store(name, cancel)
	f.statuses.SetLogsStreaming(name, true)

	go f.streamLogs(ctx, name, streamer.(PodLogStreamer))

	return true, nil
}

// streamLogs streams pod logs into the logs view until the context is
// canceled, re-attaching to fresh pods when the stream ends (e.g. redeploys)
func (f *forwarder) streamLogs(ctx context.Context, name string, streamer PodLogStreamer) {
	defer helper.RecoverAndLog(f.logsView, fmt.Sprintf("pod logs stream '%s'", name))

	f.logsView.Writef("📜  Streaming pod logs of '%s' (press 'L' again to stop)\n", name)

	out := log.NewStreamer(log.StdOut, name, f.logsView)

	for ctx.Err() == nil {
		if err := streamer.StreamPodLogs(ctx, out); err != nil && ctx.Err() == nil {
			f.logsView.Writef("📜  Log stream of '%s' interrupted (%v), re-attaching...\n", name, err)
		}

		select {
		case <-ctx.Done():
		case <-time.After(logStreamRetryDelay):
		}
	}

	f.logsView.Writef("📜  Stopped streaming pod logs of '%s'\n", name)
}

// stopLogStreams cancels all currently active pod logs streams
func (f *forwarder) stopLogStreams() {
	f.logStreams.Range(func(key, value interface{}) bool {
		value.(context.CancelFunc)()
		f.logStreams.Delete(key)
		f.statuses.SetLogsStreaming(key.(string), false)

		return true
	})
}
