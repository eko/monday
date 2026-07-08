package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewView(t *testing.T) {
	// When
	v := NewView("test-view", "Test View")

	// Then
	assert.IsType(t, new(view), v)
	assert.Implements(t, new(View), v)

	assert.Equal(t, "test-view", v.GetName())
	assert.Equal(t, "Test View", v.GetTitle())
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		name          string
		writes        []string
		expectedLines []string
	}{
		{
			name:          "single line",
			writes:        []string{"hello world\n"},
			expectedLines: []string{"hello world"},
		},
		{
			name:          "multiple lines in one write",
			writes:        []string{"line1\nline2\n"},
			expectedLines: []string{"line1", "line2"},
		},
		{
			name:          "partial line kept until completed",
			writes:        []string{"partial", " line\n"},
			expectedLines: []string{"partial line"},
		},
		{
			name:          "trailing partial line included in snapshot",
			writes:        []string{"complete\nincomplete"},
			expectedLines: []string{"complete", "incomplete"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			v := NewView("test", "Test")

			for _, write := range testCase.writes {
				v.Write(write)
			}

			lines, _ := v.snapshot()

			assert.Equal(t, testCase.expectedLines, lines)
		})
	}
}

func TestWritef(t *testing.T) {
	// Given
	v := NewView("test", "Test")

	// When
	v.Writef("hello %s, you have %d forwards\n", "vincent", 3)

	// Then
	lines, _ := v.snapshot()

	assert.Equal(t, []string{"hello vincent, you have 3 forwards"}, lines)
}

func TestWriteNotifies(t *testing.T) {
	// Given
	v := NewView("test", "Test")

	notified := 0
	v.setNotify(func() {
		notified++
	})

	// When
	v.Write("a line\n")
	v.Write("another line\n")

	// Then
	assert.Equal(t, 2, notified)
}

func TestWriteVersionIncrements(t *testing.T) {
	// Given
	v := NewView("test", "Test")

	_, initialVersion := v.snapshot()

	// When
	v.Write("a line\n")

	// Then
	_, version := v.snapshot()

	assert.Greater(t, version, initialVersion)
}

func TestWriteCapsBuffer(t *testing.T) {
	// Given
	v := NewView("test", "Test")

	// When
	for i := 0; i < maxBufferLines+100; i++ {
		v.Write("line\n")
	}

	// Then
	lines, _ := v.snapshot()

	assert.Len(t, lines, maxBufferLines)
}

func TestFilterLines(t *testing.T) {
	testCases := []struct {
		name     string
		lines    []string
		filter   string
		expected []string
	}{
		{
			name:     "matches case-insensitively",
			lines:    []string{"Hello World", "goodbye", "HELLO again"},
			filter:   "hello",
			expected: []string{"Hello World", "HELLO again"},
		},
		{
			name:     "no match",
			lines:    []string{"one", "two"},
			filter:   "three",
			expected: []string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := filterLines(testCase.lines, testCase.filter)

			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestSnapshotIsolation(t *testing.T) {
	// Given
	v := NewView("test", "Test")
	v.Write("first\n")

	lines, _ := v.snapshot()

	// When: writing after a snapshot must not mutate the snapshot content
	v.Write("second\n")

	// Then
	assert.Equal(t, []string{"first"}, lines)
	assert.False(t, strings.Contains(strings.Join(lines, "\n"), "second"))
}
