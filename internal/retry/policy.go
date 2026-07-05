//nolint:ireturn // Exported policy factories intentionally return DelayPolicy.
package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

type (
	// randomRangePolicy returns a random delay within a configured range.
	randomRangePolicy struct {
		// minDelay is the lower range bound.
		minDelay time.Duration
		// maxDelay is the upper range bound.
		maxDelay time.Duration
	}

	// exponentialByAttemptPolicy applies exponential delay growth without jitter.
	exponentialByAttemptPolicy struct {
		// baseDelay is the initial delay for exponential growth.
		baseDelay time.Duration
		// maxDelay is the upper delay bound.
		maxDelay time.Duration
	}

	// exponentialBoundedJitterPolicy adds bounded jitter to exponential delay.
	exponentialBoundedJitterPolicy struct {
		// basePolicy is the base exponential policy without jitter.
		basePolicy *exponentialByAttemptPolicy
		// jitterDivisor scales the jitter range relative to baseDelay.
		jitterDivisor uint64
	}

	// exponentialFullJitterPolicy applies full jitter to exponential delay.
	exponentialFullJitterPolicy struct {
		// basePolicy is the base exponential policy without jitter.
		basePolicy *exponentialByAttemptPolicy
	}
)

// NewRandomRangePolicy returns a random-range delay policy.
// Delay is sampled in the [minDelay, maxDelay) interval.
func NewRandomRangePolicy(minDelay, maxDelay time.Duration) DelayPolicy {
	return &randomRangePolicy{
		minDelay: minDelay,
		maxDelay: maxDelay,
	}
}

// NewExponentialByAttemptPolicy returns an exponential policy without jitter.
func NewExponentialByAttemptPolicy(baseDelay, maxDelay time.Duration) DelayPolicy {
	return &exponentialByAttemptPolicy{
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

// NewExponentialBoundedJitterPolicy returns an exponential policy with bounded jitter.
func NewExponentialBoundedJitterPolicy(baseDelay, maxDelay time.Duration, jitterDivisor uint64) DelayPolicy {
	return &exponentialBoundedJitterPolicy{
		basePolicy: &exponentialByAttemptPolicy{
			baseDelay: baseDelay,
			maxDelay:  maxDelay,
		},
		jitterDivisor: jitterDivisor,
	}
}

// NewExponentialFullJitterPolicy returns an exponential policy with full jitter.
func NewExponentialFullJitterPolicy(baseDelay, maxDelay time.Duration) DelayPolicy {
	return &exponentialFullJitterPolicy{
		basePolicy: &exponentialByAttemptPolicy{
			baseDelay: baseDelay,
			maxDelay:  maxDelay,
		},
	}
}

// Delay returns exponential delay for the given attempt.
func (p *exponentialByAttemptPolicy) Delay(attempt uint64) time.Duration {
	return exponentialDelayByAttempt(p.baseDelay, p.maxDelay, attempt)
}

// Delay returns bounded-jitter delay for the given attempt.
func (p *exponentialBoundedJitterPolicy) Delay(attempt uint64) time.Duration {
	baseDelay := p.basePolicy.Delay(attempt)

	return addJitterByFraction(baseDelay, p.jitterDivisor)
}

// Delay returns full-jitter delay for the given attempt.
func (p *exponentialFullJitterPolicy) Delay(attempt uint64) time.Duration {
	return fullJitter(p.basePolicy.Delay(attempt))
}

// Delay returns a random delay in the configured range.
func (p *randomRangePolicy) Delay(uint64) time.Duration {
	return randomDelayInRange(p.minDelay, p.maxDelay)
}

// fullJitter picks a random delay in the range [0, delay).
func fullJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}

	//nolint:gosec // A regular PRNG is sufficient for jitter.
	return time.Duration(rand.Int64N(int64(delay)))
}

// exponentialDelayByAttempt computes exponential delay by attempt number.
// attempt is one-based, while zero is treated as the first attempt.
func exponentialDelayByAttempt(baseDelay, maxDelay time.Duration, attempt uint64) time.Duration {
	if attempt == 0 {
		return exponentialDelayByStep(baseDelay, maxDelay, 0)
	}

	return exponentialDelayByStep(baseDelay, maxDelay, attempt-1)
}

// exponentialDelayByStep computes exponential delay by growth step.
// step is zero-based and scales delay by 2^step, capped by maxDelay.
func exponentialDelayByStep(baseDelay, maxDelay time.Duration, step uint64) time.Duration {
	if baseDelay <= 0 {
		return 0
	}

	if maxDelay <= 0 {
		return baseDelay
	}

	if baseDelay >= maxDelay {
		return maxDelay
	}

	delay := baseDelay
	for range step {
		if delay > maxDelay/2 {
			return maxDelay
		}

		delay *= 2
	}

	return min(delay, maxDelay)
}

// randomDelayInRange returns a random delay in the [minDelay, maxDelay) interval.
func randomDelayInRange(minDelay, maxDelay time.Duration) time.Duration {
	if minDelay > maxDelay {
		minDelay, maxDelay = maxDelay, minDelay
	}

	if minDelay == maxDelay {
		return minDelay
	}

	//nolint:gosec // A regular PRNG is sufficient for jitter.
	return minDelay + time.Duration(rand.Int64N(int64(maxDelay-minDelay)))
}

// addJitterByFraction adds random jitter up to baseDelay/fractionDivisor.
func addJitterByFraction(baseDelay time.Duration, fractionDivisor uint64) time.Duration {
	if baseDelay <= 0 {
		return 0
	}

	if fractionDivisor == 0 || fractionDivisor > math.MaxInt64 {
		return baseDelay
	}

	maxJitter := baseDelay / time.Duration(fractionDivisor)
	if maxJitter == 0 {
		return baseDelay
	}

	//nolint:gosec // A regular PRNG is sufficient for jitter.
	jitter := time.Duration(rand.Int64N(int64(maxJitter) + 1))

	return baseDelay + jitter
}
