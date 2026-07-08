package wait

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(
	t *testing.T,
) {
	testCases := []struct {
		name     string
		attempts int
		want     time.Duration
	}{
		{
			name:     "first attempt returns min",
			attempts: 1,
			want:     100 * time.Millisecond,
		},
		{
			name:     "second attempt doubles",
			attempts: 2,
			want:     200 * time.Millisecond,
		},
		{
			name:     "capped at max",
			attempts: 20,
			want:     10 * time.Second,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			backoff := Backoff{
				Min:    100 * time.Millisecond,
				Max:    10 * time.Second,
				Factor: 2,
			}

			var duration time.Duration
			for i := 0; i < testCase.attempts; i++ {
				duration = backoff.Duration()
			}

			assert.Equal(t, testCase.want, duration)
		})
	}
}

func TestReset(
	t *testing.T,
) {
	backoff := Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}

	for i := 0; i < 5; i++ {
		backoff.Duration()
	}

	backoff.Reset()

	assert.Equal(t, 100*time.Millisecond, backoff.Duration())
}
