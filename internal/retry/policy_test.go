package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRandomDelayInRange verifies random delays stay within configured bounds.
func TestRandomDelayInRange(t *testing.T) {
	t.Parallel()

	const (
		minDelay = 10 * time.Millisecond
		maxDelay = 20 * time.Millisecond
	)

	for range 500 {
		delay := randomDelayInRange(minDelay, maxDelay)
		assert.GreaterOrEqual(t, delay, minDelay)
		assert.Less(t, delay, maxDelay)
	}
}

// TestRandomDelayInRange_SwapsBounds verifies reversed bounds are normalized before sampling.
func TestRandomDelayInRange_SwapsBounds(t *testing.T) {
	t.Parallel()

	const (
		minDelay = 5 * time.Millisecond
		maxDelay = 15 * time.Millisecond
	)

	for range 500 {
		delay := randomDelayInRange(maxDelay, minDelay)
		assert.GreaterOrEqual(t, delay, minDelay)
		assert.Less(t, delay, maxDelay)
	}
}

// TestRandomDelayInRange_EqualBounds verifies equal bounds return the exact configured delay.
func TestRandomDelayInRange_EqualBounds(t *testing.T) {
	t.Parallel()

	expectedDelay := 42 * time.Millisecond
	assert.Equal(t, expectedDelay, randomDelayInRange(expectedDelay, expectedDelay))
}

// TestEqualJitter stays in the half-to-full interval used by audio download retries.
func TestEqualJitter(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Duration(1), equalJitter(1))

	delay := 16 * time.Millisecond
	for range 500 {
		got := equalJitter(delay)
		assert.GreaterOrEqual(t, got, delay/2)
		assert.Less(t, got, delay)
	}
}
