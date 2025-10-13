package zvuk

import (
	"context"
	"errors"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// ErrorHandler provides centralized error handling and recording.
type ErrorHandler struct {
	service *ServiceImpl
}

// NewErrorHandler creates an error handler for the service.
func NewErrorHandler(service *ServiceImpl) *ErrorHandler {
	return &ErrorHandler{service: service}
}

// HandleError handles an error with logging and recording.
// Returns true if the error should stop execution, false if it can be ignored.
func (h *ErrorHandler) HandleError(
	ctx context.Context,
	err error,
	errorCtx *ErrorContext,
	incrementFailed bool,
) bool {
	if err == nil {
		return false
	}

	// Don't log context cancellation - it's expected when user presses CTRL+C.
	if !errors.Is(err, context.Canceled) {
		logger.Errorf(ctx, "%s failed: %v", errorCtx.Phase, err)
	}

	// Record error for statistics.
	h.service.recordError(errorCtx, err)

	// Increment failure counter if requested.
	if incrementFailed {
		h.service.incrementTrackFailed()
	}

	return true
}

// HandleSkip handles a track skip with logging and recording.
func (h *ErrorHandler) HandleSkip(
	ctx context.Context,
	reason SkipReason,
	err error,
	errorCtx *ErrorContext,
) {
	h.service.incrementTrackSkipped(reason)

	if err != nil {
		h.service.recordError(errorCtx, err)
	}
}

// WithErrorContext executes a function and handles any errors with the provided context.
// Returns true if execution should continue, false if it should stop.
func (h *ErrorHandler) WithErrorContext(
	ctx context.Context,
	errorCtx *ErrorContext,
	fn func() error,
) bool {
	if err := fn(); err != nil {
		return !h.HandleError(ctx, err, errorCtx, true)
	}

	return true
}
