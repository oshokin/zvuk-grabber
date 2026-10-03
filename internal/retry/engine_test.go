package retry

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errTransient is a retryable sentinel used by WaitIfRetryable tests.
var errTransient = errors.New("transient")

// TestWaitIfRetryable_NilError skips scheduling when the operation succeeded.
func TestWaitIfRetryable_NilError(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, 1, AlwaysRetryable)
	require.NoError(t, engine.WaitIfRetryable(t.Context(), 0, nil, nil))
}

// TestWaitIfRetryable_NonRetryable returns the original error without sleeping.
func TestWaitIfRetryable_NonRetryable(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, 3, func(error) bool { return false })
	require.ErrorIs(t, engine.WaitIfRetryable(t.Context(), 0, errTransient, nil), errTransient)
}

// TestWaitIfRetryable_BudgetExhausted stops after the configured retry limit.
func TestWaitIfRetryable_BudgetExhausted(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, 2, AlwaysRetryable)
	require.ErrorIs(t, engine.WaitIfRetryable(t.Context(), 2, errTransient, nil), errTransient)
}

// TestWaitIfRetryable_SchedulesRetry waits and reports the next attempt number.
func TestWaitIfRetryable_SchedulesRetry(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		cfg := new(EngineConfig)
		cfg.MaxRetries = 2
		cfg.DelayPolicy = NewRandomRangePolicy(time.Millisecond, time.Millisecond)
		cfg.IsRetryable = AlwaysRetryable

		engine, err := NewEngine(cfg)
		require.NoError(t, err)

		var info *AttemptInfo

		start := time.Now()

		require.NoError(t, engine.WaitIfRetryable(
			t.Context(),
			0,
			errTransient,
			func(_ context.Context, attempt *AttemptInfo) {
				info = attempt
			},
		))
		assert.Equal(t, time.Millisecond, time.Since(start))
		require.NotNil(t, info)
		assert.Equal(t, uint64(1), info.Retry)
		assert.ErrorIs(t, info.Err, errTransient)
	})
}

// newTestEngine builds an engine with a no-op sleeper for WaitIfRetryable tests.
func newTestEngine(t *testing.T, maxRetries uint64, classifier IsRetryable) *Engine {
	t.Helper()

	cfg := new(EngineConfig)
	cfg.MaxRetries = maxRetries
	cfg.DelayPolicy = NewRandomRangePolicy(time.Millisecond, time.Millisecond)
	cfg.IsRetryable = classifier
	cfg.Sleeper = func(context.Context, time.Duration) error { return nil }

	engine, err := NewEngine(cfg)
	require.NoError(t, err)

	return engine
}
