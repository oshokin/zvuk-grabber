package retry

import (
	"context"
	"time"
)

// DefaultSleeper waits for delay while respecting context cancellation.
func DefaultSleeper(ctx context.Context, delay time.Duration) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}

	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
