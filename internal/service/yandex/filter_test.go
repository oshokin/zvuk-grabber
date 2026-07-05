package yandex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestValidateTrackJob_BelowMinDuration verifies tracks shorter than the minimum duration are rejected.
func TestValidateTrackJob_BelowMinDuration(t *testing.T) {
	t.Parallel()

	s := &ServiceImpl{
		cfg: &config.Config{
			ParsedMinDuration: 2 * time.Minute,
		},
	}
	job := &trackJob{
		track: &model.Track{
			ID:         model.NewFlexibleID("1"),
			DurationMs: int((90 * time.Second).Milliseconds()),
		},
	}

	result := s.validateTrackJob(job)
	assert.False(t, result.allowed)
	assert.Equal(t, yandexSkipReasonDuration, result.reason)
	assert.Error(t, result.err)
}

// TestValidateTrackJob_AboveMaxDuration verifies tracks longer than the maximum duration are rejected.
func TestValidateTrackJob_AboveMaxDuration(t *testing.T) {
	t.Parallel()

	s := &ServiceImpl{
		cfg: &config.Config{
			ParsedMaxDuration: 3 * time.Minute,
		},
	}
	job := &trackJob{
		track: &model.Track{
			ID:         model.NewFlexibleID("1"),
			DurationMs: int((5 * time.Minute).Milliseconds()),
		},
	}

	result := s.validateTrackJob(job)
	assert.False(t, result.allowed)
	assert.Equal(t, yandexSkipReasonDuration, result.reason)
	assert.Error(t, result.err)
}

// TestValidateTrackJob_WithinDurationRange verifies tracks within the configured duration range are accepted.
func TestValidateTrackJob_WithinDurationRange(t *testing.T) {
	t.Parallel()

	s := &ServiceImpl{
		cfg: &config.Config{
			ParsedMinDuration: 2 * time.Minute,
			ParsedMaxDuration: 4 * time.Minute,
		},
	}
	job := &trackJob{
		track: &model.Track{
			ID:         model.NewFlexibleID("1"),
			DurationMs: int((3 * time.Minute).Milliseconds()),
		},
	}

	result := s.validateTrackJob(job)
	assert.True(t, result.allowed)
	assert.NoError(t, result.err)
}

// TestValidateTrackJob_MissingTrackID verifies jobs without track ID are rejected.
func TestValidateTrackJob_MissingTrackID(t *testing.T) {
	t.Parallel()

	s := &ServiceImpl{
		cfg: &config.Config{},
	}
	job := &trackJob{
		track: &model.Track{
			DurationMs: int((3 * time.Minute).Milliseconds()),
		},
	}

	result := s.validateTrackJob(job)
	assert.False(t, result.allowed)
	assert.Equal(t, yandexSkipReasonUnknown, result.reason)
	assert.Error(t, result.err)
}
