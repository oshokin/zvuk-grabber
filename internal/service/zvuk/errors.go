package zvuk

import (
	"context"
	"errors"
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

// ErrorContext provides context information for download errors.
type ErrorContext struct {
	// Category is the type of item that failed (track, album, playlist, artist).
	Category DownloadCategory
	// ItemID is the unique identifier of the item that failed.
	ItemID string
	// ItemTitle is the human-readable title of the item.
	ItemTitle string
	// ItemURL is the URL of the failed item (for albums/playlists/artists).
	ItemURL string
	// Phase indicates when the error occurred (e.g., "fetching metadata", "downloading track").
	Phase string
	// ParentCategory is the type of parent collection (album/playlist) for tracks.
	ParentCategory DownloadCategory
	// ParentID is the ID of the parent collection.
	ParentID string
	// ParentTitle is the title of the parent collection.
	ParentTitle string
}

// recordError records an error in the statistics with proper context.
// Context cancellation errors are ignored as they are expected during graceful shutdown.
func (s *ServiceImpl) recordError(errCtx *ErrorContext, err error) {
	if errCtx == nil || err == nil {
		return
	}

	// Don't record context cancellation as an error - it's expected when user presses CTRL+C.
	if errors.Is(err, context.Canceled) {
		return
	}

	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	downloadErr := DownloadError{
		Category:       errCtx.Category,
		ItemID:         errCtx.ItemID,
		ItemTitle:      errCtx.ItemTitle,
		ItemURL:        errCtx.ItemURL,
		ErrorMessage:   err.Error(),
		Phase:          errCtx.Phase,
		ParentCategory: errCtx.ParentCategory,
		ParentID:       errCtx.ParentID,
		ParentTitle:    errCtx.ParentTitle,
	}

	s.stats.Errors = append(s.stats.Errors, downloadErr)
}
