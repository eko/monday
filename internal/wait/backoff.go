package wait

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	defaultMin    = 100 * time.Millisecond
	defaultMax    = 30 * time.Second
	defaultFactor = 2

	maxInt64 = float64(math.MaxInt64 - 512)
)

type Backoff struct {
	attempt uint64
	Factor  float64
	Jitter  bool
	Min     time.Duration
	Max     time.Duration
}

func (b *Backoff) Duration() time.Duration {
	d := b.ForAttempt(float64(atomic.AddUint64(&b.attempt, 1) - 1))
	return d
}

func (b *Backoff) ForAttempt(attempt float64) time.Duration {
	min := b.Min
	if min <= 0 {
		min = defaultMin
	}

	max := b.Max
	if max <= 0 {
		max = defaultMax
	}

	if min >= max {
		return max
	}

	factor := b.Factor
	if factor <= 0 {
		factor = defaultFactor
	}

	minFloat := float64(min)
	durationExponential := minFloat * math.Pow(factor, attempt)
	if b.Jitter {
		durationExponential = rand.Float64()*(durationExponential-minFloat) + minFloat
	}

	if durationExponential > maxInt64 {
		return max
	}

	dur := time.Duration(durationExponential)
	if dur < min {
		return min
	}
	if dur > max {
		return max
	}
	return dur
}
