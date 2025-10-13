package zvuk

import (
	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// TrackDownloadContext encapsulates all data needed for downloading a single track.
// This value object reduces parameter passing and makes the download flow clearer.
type TrackDownloadContext struct {
	// Track metadata.
	TrackID       string
	Track         *zvuk.Track
	TrackIndex    int64
	TrackPosition int64

	// Quality and streaming.
	Quality   TrackQuality
	StreamURL string

	// Collection context.
	AudioCollection *audioCollection
	AlbumTags       map[string]string
	Album           *zvuk.Release

	// Category flags.
	IsAudiobook bool
	IsPlaylist  bool
	IsPodcast   bool

	// Error reporting context.
	ParentID       string
	ParentTitle    string
	ParentCategory DownloadCategory

	// File paths.
	TrackFilename string
	TrackPath     string
}

// NewTrackDownloadContext creates a download context with basic information.
func NewTrackDownloadContext(
	trackID string,
	track *zvuk.Track,
	trackIndex int64,
	metadata *downloadTracksMetadata,
) *TrackDownloadContext {
	isAudiobook := metadata.category == DownloadCategoryAudiobook
	isPlaylist := metadata.category == DownloadCategoryPlaylist
	isPodcast := metadata.category == DownloadCategoryPodcast

	return &TrackDownloadContext{
		TrackID:         trackID,
		Track:           track,
		TrackIndex:      trackIndex,
		AudioCollection: metadata.audioCollection,
		IsAudiobook:     isAudiobook,
		IsPlaylist:      isPlaylist,
		IsPodcast:       isPodcast,
		ParentCategory:  metadata.category,
	}
}
