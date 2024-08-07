package log

import (
	"testing"

	"github.com/eko/monday/pkg/ui"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
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
