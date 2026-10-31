package log

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/eko/monday/pkg/ui"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewStreamer(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)

	testCases := []struct {
		stdType string
		name    string
	}{
		{stdType: StdOut, name: "test-stdout"},
		{stdType: StdErr, name: "test-stderr"},
	}

	for _, testCase := range testCases {
		// When
		streamer := NewStreamer(testCase.stdType, testCase.name, view)

		// Then
		assert.IsType(t, new(Streamer), streamer)

		assert.Equal(t, testCase.stdType, streamer.stdType)
		assert.Equal(t, testCase.name, streamer.name)
	}
}

func TestStreamerConcurrentWrites(t *testing.T) {
	// Given
	const (
		writers = 32
		lines   = 2000
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		mux      sync.Mutex
		received []string
	)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Write(gomock.Any()).DoAndReturn(func(str string) {
		mux.Lock()
		defer mux.Unlock()

		received = append(received, str)
	}).Times(writers * lines)

	streamer := NewStreamer(StdOut, "registry-0", view)

	// When
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range lines {
				fmt.Fprintf(streamer, "Handling connection for %d\n", 5000)
			}
		}()
	}

	wg.Wait()

	// Then
	assert.Len(t, received, writers*lines)

	for _, line := range received {
		assert.True(t, strings.HasSuffix(line, "Handling connection for 5000\n"), line)
	}
}
