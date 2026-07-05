//nolint:err113 // Validation result keeps detailed skip messages inline for user-facing diagnostics.
package yandex

import (
	"errors"
	"fmt"
	"time"
)

// validationResult reports whether a track job passes pre-download validation.
type validationResult struct {
	// allowed reports whether the track may proceed to download.
	allowed bool
	// reason classifies the skip reason when allowed is false.
	reason yandexSkipReason
	// err carries the detailed validation failure message.
	err error
}

// validateTrackJob checks duration bounds and basic job integrity before download.
func (s *ServiceImpl) validateTrackJob(job *trackJob) *validationResult {
	if job == nil || job.track == nil {
		return &validationResult{
			allowed: false,
			reason:  yandexSkipReasonUnknown,
			err:     errors.New("track job is not initialized"),
		}
	}

	if yandexJobTrackID(job) == "" {
		return &validationResult{
			allowed: false,
			reason:  yandexSkipReasonUnknown,
			err:     errors.New("track ID is not initialized"),
		}
	}

	duration := time.Duration(job.track.DurationMs) * time.Millisecond

	if s.cfg.ParsedMinDuration > 0 && duration < s.cfg.ParsedMinDuration {
		return &validationResult{
			allowed: false,
			reason:  yandexSkipReasonDuration,
			err: fmt.Errorf(
				"track duration %s is below min_duration %s",
				duration,
				s.cfg.ParsedMinDuration,
			),
		}
	}

	if s.cfg.ParsedMaxDuration > 0 && duration > s.cfg.ParsedMaxDuration {
		return &validationResult{
			allowed: false,
			reason:  yandexSkipReasonDuration,
			err: fmt.Errorf(
				"track duration %s is above max_duration %s",
				duration,
				s.cfg.ParsedMaxDuration,
			),
		}
	}

	return &validationResult{allowed: true}
}
