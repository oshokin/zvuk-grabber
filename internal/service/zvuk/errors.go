package zvuk

import (
	"context"
	"errors"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// Common errors for the service layer.
var (
	// ErrTrackNotFound indicates that the requested track was not found.
	ErrTrackNotFound = errors.New("track not found")
	// ErrTrackAlbumNotFound indicates that the album for the track was not found.
	ErrTrackAlbumNotFound = errors.New("track album not found")
	// ErrIncompleteDownload indicates that the downloaded file size doesn't match expected size.
	ErrIncompleteDownload = errors.New("incomplete download")
	// ErrQualityBelowThreshold indicates that track quality is below the configured minimum.
	ErrQualityBelowThreshold = errors.New("quality below minimum threshold")
	// ErrDurationBelowThreshold indicates that track duration is below the configured minimum.
	ErrDurationBelowThreshold = errors.New("duration below minimum threshold")
	// ErrDurationAboveThreshold indicates that track duration exceeds the configured maximum.
	ErrDurationAboveThreshold = errors.New("duration above maximum threshold")
	// ErrChapterStreamNotFound indicates that stream metadata for a chapter was not found.
	ErrChapterStreamNotFound = errors.New("chapter stream metadata not found")
	// ErrChapterNoStreams indicates that a chapter has no available streams.
	ErrChapterNoStreams = errors.New("chapter has no available streams")
	// ErrChapterNoStreamURL indicates that no stream URL is available for a chapter.
	ErrChapterNoStreamURL = errors.New("no stream URL available for chapter")
	// ErrAudiobookContextFailed indicates that preparing audiobook context failed.
	ErrAudiobookContextFailed = errors.New("failed to prepare audiobook context")
	// ErrAlbumContextFailed indicates that preparing album context failed.
	ErrAlbumContextFailed = errors.New("failed to prepare album context")
	// ErrPodcastContextFailed indicates that preparing podcast context failed.
	ErrPodcastContextFailed = errors.New("failed to prepare podcast context")
)

// handleError handles an error with logging and recording.
// Returns true if the error should stop execution, false if it can be ignored.
func (s *ServiceImpl) handleError(
	ctx context.Context,
	e *DownloadError,
	incrementFailed bool,
) {
	if e == nil || e.Error == nil {
		return
	}

	isContextCanceled := errors.Is(e.Error, context.Canceled)

	// Don't log context cancellation - it's expected when user presses CTRL+C.
	if !isContextCanceled {
		logger.Errorf(ctx, "%s failed: %v", e.Phase, e.Error)
	}

	// Record error for statistics.
	s.recordError(e)

	// Increment failure counter if requested.
	if incrementFailed && !isContextCanceled {
		s.incrementTrackFailed()
	}
}

// handleTrackSkipped handles a track skip with logging and recording.
func (s *ServiceImpl) handleTrackSkipped(
	reason SkipReason,
	e *DownloadError,
) {
	s.incrementTrackSkipped(reason)

	if e != nil {
		s.recordError(e)
	}
}

// recordError records an error in the statistics with proper context.
// Context cancellation errors are ignored as they are expected during graceful shutdown.
func (s *ServiceImpl) recordError(e *DownloadError) {
	if e == nil || e.Error == nil {
		return
	}

	// Don't record context cancellation as an error - it's expected when user presses CTRL+C.
	if errors.Is(e.Error, context.Canceled) {
		return
	}

	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.Errors = append(s.stats.Errors, e)
}
