package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type (
	// DelayPolicy defines how the pause between retry attempts is calculated.
	DelayPolicy interface {
		// Delay returns the pause duration before a retry.
		// attempt is the retry attempt number starting from 1.
		Delay(attempt uint64) time.Duration
	}

	// IsRetryable decides whether an operation should be retried for a given error.
	IsRetryable func(error) bool

	// AttemptInfo contains details about a scheduled retry attempt.
	AttemptInfo struct {
		// Retry is the retry attempt number starting from 1.
		Retry uint64
		// MaxRetries is the retry limit from engine configuration.
		MaxRetries uint64
		// Delay is the calculated pause before the next attempt.
		Delay time.Duration
		// Err is the error that caused this retry to be scheduled.
		Err error
	}

	// OnRetry is called before waiting for the next retry attempt.
	OnRetry func(ctx context.Context, info *AttemptInfo)

	// Sleeper performs waiting between retry attempts.
	Sleeper func(ctx context.Context, delay time.Duration) error

	// EngineConfig describes retry engine behavior.
	EngineConfig struct {
		// MaxRetries is the number of retries after the first failed attempt.
		// The total number of operation runs does not exceed 1 + MaxRetries.
		// A value of 0 means unlimited retries.
		MaxRetries uint64
		// DelayPolicy is used to calculate the pause before the next attempt.
		DelayPolicy DelayPolicy
		// IsRetryable classifies errors eligible for retry.
		IsRetryable IsRetryable
		// Sleeper is the waiting mechanism for retry pauses.
		// If nil, DefaultSleeper is used.
		Sleeper Sleeper
	}

	// Request describes a single operation run via Engine.Run.
	Request struct {
		// Operation is the operation to execute with retries.
		Operation func(context.Context) error
		// OnRetry is an optional callback before a retry attempt.
		OnRetry OnRetry
	}

	// Engine is an immutable retry engine.
	// It is safe for concurrent Run calls.
	Engine struct {
		// maxRetries is the retry limit.
		// A value of 0 means no limit.
		maxRetries uint64
		// delayPolicy defines how to calculate pause before retry.
		delayPolicy DelayPolicy
		// isRetryable classifies errors that can be retried.
		isRetryable IsRetryable
		// sleeper waits between retries.
		sleeper Sleeper
	}
)

var (
	// ErrInvalidConfig indicates an invalid retry engine configuration.
	ErrInvalidConfig = errors.New("invalid retry engine config")
	// ErrInvalidRequest indicates an invalid operation execution request.
	ErrInvalidRequest = errors.New("invalid retry request")
)

// AlwaysRetryable treats any error as retryable.
// Context errors are handled separately and are never retried.
func AlwaysRetryable(error) bool {
	return true
}

// NewEngine creates an Engine and validates its configuration.
func NewEngine(cfg *EngineConfig) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	if cfg.DelayPolicy == nil {
		return nil, fmt.Errorf("%w: delay policy is nil", ErrInvalidConfig)
	}

	if cfg.IsRetryable == nil {
		return nil, fmt.Errorf("%w: retry classifier is nil", ErrInvalidConfig)
	}

	sleeper := cfg.Sleeper
	if sleeper == nil {
		sleeper = DefaultSleeper
	}

	return &Engine{
		maxRetries:  cfg.MaxRetries,
		delayPolicy: cfg.DelayPolicy,
		isRetryable: cfg.IsRetryable,
		sleeper:     sleeper,
	}, nil
}

// validateRequest validates incoming request data for Run.
func validateRequest(req *Request) error {
	if req == nil {
		return fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}

	if req.Operation == nil {
		return fmt.Errorf("%w: operation is nil", ErrInvalidRequest)
	}

	return nil
}

// isContextOperationError checks whether the error is a context termination error.
func isContextOperationError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Run executes the operation and retries it according to configuration.
func (e *Engine) Run(ctx context.Context, req *Request) error {
	if err := validateRequest(req); err != nil {
		return err
	}

	var retries uint64

	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		operationErr := req.Operation(ctx)
		if operationErr == nil {
			return nil
		}

		if stopErr := e.retryStopError(ctx, operationErr, retries); stopErr != nil {
			return stopErr
		}

		retries++
		if sleepErr := e.scheduleRetry(ctx, req, retries, operationErr); sleepErr != nil {
			return sleepErr
		}
	}
}

// retryStopError determines whether retries should stop and return an error.
func (e *Engine) retryStopError(ctx context.Context, operationErr error, retries uint64) error {
	// Context errors take priority and are never retried.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}

	if isContextOperationError(operationErr) {
		return operationErr
	}

	if !e.isRetryable(operationErr) {
		return operationErr
	}

	// With maxRetries == 0, retries are unlimited.
	if e.maxRetries != 0 && retries >= e.maxRetries {
		return operationErr
	}

	return nil
}

// scheduleRetry computes delay, invokes OnRetry, and waits before retrying.
func (e *Engine) scheduleRetry(ctx context.Context, req *Request, retries uint64, operationErr error) error {
	delay := e.delayPolicy.Delay(retries)
	if req.OnRetry != nil {
		info := &AttemptInfo{
			Retry:      retries,
			MaxRetries: e.maxRetries,
			Delay:      delay,
			Err:        operationErr,
		}

		req.OnRetry(ctx, info)
	}

	return e.sleeper(ctx, delay)
}
