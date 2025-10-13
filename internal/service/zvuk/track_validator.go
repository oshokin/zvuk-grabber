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

// ValidationRule defines a single validation check for tracks.
type ValidationRule struct {
	// Name is a human-readable name for the rule.
	Name string
	// Check performs the validation and returns true if valid.
	Check func(context.Context, *zvuk.Track, *config.Config) bool
	// SkipReason is the reason to use if validation fails.
	SkipReason SkipReason
	// ErrorFunc generates an error message for failed validation.
	ErrorFunc func(*zvuk.Track, *config.Config) error
}

// TrackValidator validates tracks against configured constraints.
type TrackValidator struct {
	cfg   *config.Config
	rules []*ValidationRule
}

// NewTrackValidator creates a validator with standard validation rules.
func NewTrackValidator(cfg *config.Config) *TrackValidator {
	return &TrackValidator{
		cfg: cfg,
		rules: []*ValidationRule{
			{
				Name:       "minimum duration",
				Check:      checkMinDuration,
				SkipReason: SkipReasonDuration,
				ErrorFunc:  errMinDuration,
			},
			{
				Name:       "maximum duration",
				Check:      checkMaxDuration,
				SkipReason: SkipReasonDuration,
				ErrorFunc:  errMaxDuration,
			},
		},
	}
}

// Validate runs all validation rules against a track.
func (v *TrackValidator) Validate(ctx context.Context, track *zvuk.Track) *ValidationResult {
	for _, rule := range v.rules {
		if !rule.Check(ctx, track, v.cfg) {
			logger.Warnf(ctx, "Track validation failed: %s", rule.Name)

			return &ValidationResult{
				IsValid:    false,
				SkipReason: rule.SkipReason,
				Error:      rule.ErrorFunc(track, v.cfg),
			}
		}
	}

	return &ValidationResult{
		IsValid: true,
	}
}

// checkMinDuration validates minimum duration constraint.
func checkMinDuration(ctx context.Context, track *zvuk.Track, cfg *config.Config) bool {
	if cfg.ParsedMinDuration <= 0 {
		return true // No constraint.
	}

	trackDuration := time.Duration(track.Duration) * time.Second
	if trackDuration < cfg.ParsedMinDuration {
		logger.Warnf(ctx, "Track duration %ds is below minimum threshold %s, skipping",
			track.Duration, cfg.ParsedMinDuration)

		return false
	}

	return true
}

// checkMaxDuration validates maximum duration constraint.
func checkMaxDuration(ctx context.Context, track *zvuk.Track, cfg *config.Config) bool {
	if cfg.ParsedMaxDuration <= 0 {
		return true // No constraint.
	}

	trackDuration := time.Duration(track.Duration) * time.Second
	if trackDuration > cfg.ParsedMaxDuration {
		logger.Warnf(ctx, "Track duration %ds exceeds maximum threshold %s, skipping",
			track.Duration, cfg.ParsedMaxDuration)

		return false
	}

	return true
}

// errMinDuration generates error for minimum duration violation.
func errMinDuration(track *zvuk.Track, cfg *config.Config) error {
	return fmt.Errorf("%w: %ds below %s",
		ErrDurationBelowThreshold, track.Duration, cfg.ParsedMinDuration)
}

// errMaxDuration generates error for maximum duration violation.
func errMaxDuration(track *zvuk.Track, cfg *config.Config) error {
	return fmt.Errorf("%w: %ds exceeds %s",
		ErrDurationAboveThreshold, track.Duration, cfg.ParsedMaxDuration)
}
