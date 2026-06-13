package zvuk

import (
	"context"
	"fmt"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// ValidationResult contains the result of track validation.
type ValidationResult struct {
	// IsValid indicates if the track passed all validation rules.
	IsValid bool
	// SkipReason indicates why the track should be skipped (if IsValid is false).
	SkipReason SkipReason
	// Error contains the validation error (if IsValid is false).
	Error error
}

// TrackValidator validates tracks against configured constraints.
type TrackValidator struct {
	// cfg contains duration and quality validation thresholds.
	cfg *config.Config
}

// NewTrackValidator creates a validator with standard validation rules.
func NewTrackValidator(cfg *config.Config) *TrackValidator {
	return &TrackValidator{cfg: cfg}
}

// Validate runs all validation rules against a track.
func (v *TrackValidator) Validate(ctx context.Context, track *zvuk.Track) *ValidationResult {
	trackDuration := time.Duration(track.Duration) * time.Second

	if minimum := v.cfg.ParsedMinDuration; minimum > 0 && trackDuration < minimum {
		logger.Warnf(ctx, "Track duration %ds is below minimum threshold %s, skipping", track.Duration, minimum)

		return invalidDurationResult(ctx, "minimum duration", fmt.Errorf(
			"%w: %ds below %s", ErrDurationBelowThreshold, track.Duration, minimum,
		))
	}

	if maximum := v.cfg.ParsedMaxDuration; maximum > 0 && trackDuration > maximum {
		logger.Warnf(ctx, "Track duration %ds exceeds maximum threshold %s, skipping", track.Duration, maximum)

		return invalidDurationResult(ctx, "maximum duration", fmt.Errorf(
			"%w: %ds exceeds %s", ErrDurationAboveThreshold, track.Duration, maximum,
		))
	}

	return &ValidationResult{IsValid: true}
}

// invalidDurationResult builds a validation failure result for duration rule violations.
func invalidDurationResult(ctx context.Context, ruleName string, err error) *ValidationResult {
	logger.Warnf(ctx, "Track validation failed: %s", ruleName)

	return &ValidationResult{
		SkipReason: SkipReasonDuration,
		Error:      err,
	}
}
