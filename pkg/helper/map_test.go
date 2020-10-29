package helper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMapString(t *testing.T) {
	// Given
	testCases := []struct {
		map1     map[string]string
		map2     map[string]string
		expected map[string]string
	}{
		{
			map1:     map[string]string{},
			map2:     map[string]string{},
			expected: map[string]string{},
		},
		{
			map1:     nil,
			map2:     nil,
			expected: map[string]string{},
		},
		{
			map1: nil,
			map2: map[string]string{
				"foo1": "bar1",
			},
			expected: map[string]string{
				"foo1": "bar1",
			},
		},
		{
			map1: map[string]string{
				"foo1": "bar1",
			},
			map2: nil,
			expected: map[string]string{
				"foo1": "bar1",
			},
		},
		{
			map1: map[string]string{
				"foo1": "bar1",
			},
			map2: map[string]string{
				"foo2": "bar2",
			},
			expected: map[string]string{
				"foo1": "bar1",
				"foo2": "bar2",
			},
		},
		{
			map1: map[string]string{
				"foo1": "bar1",
			},
			map2: map[string]string{
				"foo1": "bar2",
			},
			expected: map[string]string{
				"foo1": "bar1",
			},
		},
		{
			map1: map[string]string{
				"foo1": "bar1",
				"foo3": "bar3",
			},
			map2: map[string]string{
				"foo2": "bar2",
			},
			expected: map[string]string{
				"foo1": "bar1",
				"foo3": "bar3",
				"foo2": "bar2",
			},
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("Case #%d", i), func(t *testing.T) {
			result := MergeMapString(testCase.map1, testCase.map2)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
