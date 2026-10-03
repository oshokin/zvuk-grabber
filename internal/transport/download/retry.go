package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/retry"
)

// statusError carries retry metadata without leaking response bodies or signed URLs.
type statusError struct {
	// code is the HTTP response status.
	code int
	// retryAfter is the server's requested delay, if present.
	retryAfter string
}

// retryAfterAwarePolicy raises exponential delay to a server Retry-After value.
type retryAfterAwarePolicy struct {
	// inner is the exponential equal-jitter policy from download HTTP settings.
	inner retry.DelayPolicy
	// after is the parsed Retry-After delay for the current failure, if any.
	after time.Duration
}

// Error reports only the status code.
func (e *statusError) Error() string {
	return fmt.Sprintf("HTTP %d", e.code)
}

// Delay returns the larger of exponential backoff and Retry-After.
func (p *retryAfterAwarePolicy) Delay(attempt uint64) time.Duration {
	delay := p.inner.Delay(attempt)
	if p.after > delay {
		return p.after
	}

	return delay
}

// newAudioRetryEngine builds one retry engine for a stream's header and body attempts.
func newAudioRetryEngine(cfg *config.DownloadHTTPConfig, delayPolicy retry.DelayPolicy) (*retry.Engine, error) {
	classifier := isTransient
	maxRetries := uint64(1)

	if cfg.MaxRetries <= 0 {
		classifier = func(error) bool { return false }
	} else {
		maxRetries = uint64(cfg.MaxRetries)
	}

	engineCfg := &retry.EngineConfig{
		MaxRetries:  maxRetries,
		DelayPolicy: delayPolicy,
		IsRetryable: classifier,
	}

	return retry.NewEngine(engineCfg)
}

// retry waits before one additional request; every retry consumes the same finite budget.
func (s *Stream) retry(err error) error {
	s.delayPolicy.after = 0

	if status, ok := errors.AsType[*statusError](err); ok {
		after := retryAfterDelay(status.retryAfter, time.Now())
		if after > s.config.RetryMaxDelay {
			return fmt.Errorf("server Retry-After exceeds retry_max_delay: %w", err)
		}

		s.delayPolicy.after = after
	}

	waitErr := s.engine.WaitIfRetryable(s.request.Context(), s.retries, err, s.logRetry)
	if waitErr != nil {
		return waitErr
	}

	s.retries++

	return nil
}

// logRetry reports a scheduled audio retry before the engine sleeps.
func (s *Stream) logRetry(ctx context.Context, info *retry.AttemptInfo) {
	logger.Warnf(
		ctx,
		"Audio transfer interrupted; retry %d/%d in %s: %v",
		info.Retry,
		s.config.MaxRetries,
		info.Delay.Round(time.Millisecond),
		info.Err,
	)
}

// isTransient restricts retries to interrupted bodies, temporary network failures and selected statuses.
func isTransient(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, errResumeRejected) {
		return true
	}

	if status, ok := errors.AsType[*statusError](err); ok {
		switch status.code {
		case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError,
			http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) || isConnectionReset(err) {
		return true
	}

	var netErr net.Error

	return errors.As(err, &netErr) && netErr.Timeout()
}

// retryAfterDelay parses both HTTP-date and delta-seconds, saturating oversized values.
func retryAfterDelay(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)

	seconds, err := strconv.ParseUint(value, 10, 64)
	if errors.Is(err, strconv.ErrRange) {
		return time.Duration(1<<63 - 1)
	}

	if err == nil {
		// maxSeconds prevents overflow when converting seconds to nanoseconds.
		const maxSeconds = uint64((1<<63 - 1) / int64(time.Second))
		if seconds > maxSeconds {
			return time.Duration(1<<63 - 1)
		}

		return time.Duration(seconds) * time.Second
	}

	if date, dateErr := http.ParseTime(value); dateErr == nil {
		return max(time.Duration(0), date.Sub(now))
	}

	return 0
}
