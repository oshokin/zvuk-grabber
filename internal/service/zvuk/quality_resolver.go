package zvuk

import (
	"context"
	"errors"
	"fmt"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// QualityResolutionResult contains the result of quality resolution.
type QualityResolutionResult struct {
	// Quality is the final quality determined for the track.
	Quality TrackQuality
	// StreamURL is the URL to stream/download the track.
	StreamURL string
	// ShouldSkip indicates if the track should be skipped due to quality constraints.
	ShouldSkip bool
	// SkipReason provides the reason for skipping (if ShouldSkip is true).
	SkipReason error
}

// QualityResolver resolves the actual quality and stream URL for tracks.
// Different implementations handle tracks vs audiobook chapters.
type QualityResolver interface {
	// ResolveQuality determines the final quality and stream URL for a track.
	ResolveQuality(
		ctx context.Context,
		trackID string,
		track *zvuk.Track,
		desiredQuality TrackQuality,
		minQuality TrackQuality,
	) (*QualityResolutionResult, error)
}

// trackQualityResolver handles quality resolution for regular tracks.
type trackQualityResolver struct {
	zvukClient zvuk.Client
}

// NewTrackQualityResolver creates a resolver for regular tracks.
func NewTrackQualityResolver(client zvuk.Client) QualityResolver {
	return &trackQualityResolver{zvukClient: client}
}

// ResolveQuality resolves quality for regular tracks using API stream metadata.
func (r *trackQualityResolver) ResolveQuality(
	ctx context.Context,
	trackID string,
	track *zvuk.Track,
	desiredQuality TrackQuality,
	minQuality TrackQuality,
) (*QualityResolutionResult, error) {
	// Determine highest quality available for this track.
	highestQuality := ParseQuality(track.HighestQuality)
	if highestQuality == TrackQualityUnknown {
		highestQuality = TrackQualityMP3Mid

		logger.Infof(ctx, "Failed to parse highest quality available: %s", track.HighestQuality)
	}

	// Cap desired quality at what's available.
	finalQuality := desiredQuality
	if highestQuality < desiredQuality {
		finalQuality = highestQuality
		logger.Infof(ctx, "Track is only available in quality: %s", highestQuality)
	}

	// Check minimum quality threshold.
	if minQuality > 0 && finalQuality < minQuality {
		logger.Warnf(ctx, "Track quality %s is below minimum threshold %s, skipping",
			finalQuality, minQuality)

		return &QualityResolutionResult{
			ShouldSkip: true,
			SkipReason: fmt.Errorf("%w: %s below %s",
				ErrQualityBelowThreshold, finalQuality, minQuality),
		}, nil
	}

	// Fetch stream metadata from API.
	streamMetadata, err := r.zvukClient.GetStreamMetadata(ctx, trackID, finalQuality.AsStreamURLParameterValue())
	if err != nil {
		return nil, fmt.Errorf("failed to get stream metadata: %w", err)
	}

	// Verify actual quality from stream URL.
	streamURL := streamMetadata.Stream

	actualQuality := defineQualityByStreamURL(streamURL)
	if actualQuality != TrackQualityUnknown {
		finalQuality = actualQuality
	}

	return &QualityResolutionResult{
		Quality:    finalQuality,
		StreamURL:  streamURL,
		ShouldSkip: false,
	}, nil
}

// audiobookQualityResolver handles quality resolution for audiobook chapters.
type audiobookQualityResolver struct {
	chapterStreams map[string]*zvuk.ChapterStreamMetadata
}

// NewAudiobookQualityResolver creates a resolver for audiobook chapters.
func NewAudiobookQualityResolver(chapterStreams map[string]*zvuk.ChapterStreamMetadata) QualityResolver {
	return &audiobookQualityResolver{chapterStreams: chapterStreams}
}

// ResolveQuality resolves quality for audiobook chapters using pre-fetched stream metadata.
func (r *audiobookQualityResolver) ResolveQuality(
	ctx context.Context,
	trackID string,
	track *zvuk.Track,
	desiredQuality TrackQuality,
	minQuality TrackQuality,
) (*QualityResolutionResult, error) {
	// Retrieve pre-fetched chapter stream metadata.
	streamMetadata, ok := r.chapterStreams[trackID]
	if !ok || streamMetadata == nil {
		return nil, fmt.Errorf("%w: chapter '%s'", ErrChapterStreamNotFound, trackID)
	}

	// Determine highest available quality for this chapter.
	highestAvailable := getHighestAvailableQuality(streamMetadata)
	if highestAvailable == TrackQualityUnknown {
		return nil, fmt.Errorf("%w: chapter '%s'", ErrChapterNoStreams, trackID)
	}

	// Check minimum quality threshold.
	if minQuality > 0 && highestAvailable < minQuality {
		logger.Warnf(ctx, "Chapter quality %s is below minimum threshold %s, skipping",
			highestAvailable, minQuality)

		return &QualityResolutionResult{
			ShouldSkip: true,
			SkipReason: fmt.Errorf("%w: %s below %s",
				ErrQualityBelowThreshold, highestAvailable, minQuality),
		}, nil
	}

	// Cap desired quality at what's available.
	finalQuality := desiredQuality
	if desiredQuality > highestAvailable {
		finalQuality = highestAvailable
		logger.Infof(ctx, "Chapter is only available in quality: %s", highestAvailable)
	}

	// Select stream URL with fallback logic.
	streamURL := selectChapterStreamURL(streamMetadata, finalQuality)
	if streamURL == "" {
		return nil, fmt.Errorf("%w: chapter '%s' at quality %s", ErrChapterNoStreamURL, trackID, finalQuality)
	}

	// Verify actual quality from stream URL.
	actualQuality := defineQualityByStreamURL(streamURL)
	if actualQuality != TrackQualityUnknown {
		finalQuality = actualQuality
	}

	return &QualityResolutionResult{
		Quality:    finalQuality,
		StreamURL:  streamURL,
		ShouldSkip: false,
	}, nil
}

// getHighestAvailableQuality determines the highest available quality from chapter stream metadata.
func getHighestAvailableQuality(streamMetadata *zvuk.ChapterStreamMetadata) TrackQuality {
	if streamMetadata.FLAC != "" {
		return TrackQualityFLAC
	}

	if streamMetadata.High != "" {
		return TrackQualityMP3High
	}

	if streamMetadata.Mid != "" {
		return TrackQualityMP3Mid
	}

	return TrackQualityUnknown
}

// selectChapterStreamURL selects the appropriate stream URL based on desired quality with fallback.
func selectChapterStreamURL(
	streamMetadata *zvuk.ChapterStreamMetadata,
	desiredQuality TrackQuality,
) string {
	switch desiredQuality {
	case TrackQualityFLAC:
		if streamMetadata.FLAC != "" {
			return streamMetadata.FLAC
		}

		if streamMetadata.High != "" {
			return streamMetadata.High
		}

		return streamMetadata.Mid

	case TrackQualityMP3High:
		if streamMetadata.High != "" {
			return streamMetadata.High
		}

		return streamMetadata.Mid

	case TrackQualityMP3Mid:
		return streamMetadata.Mid

	default:
		return streamMetadata.Mid
	}
}

// defineQualityByStreamURL determines quality by analyzing the stream URL pattern.
func defineQualityByStreamURL(streamURL string) TrackQuality {
	switch {
	case contains(streamURL, "/stream?"):
		return TrackQualityMP3Mid
	case contains(streamURL, "/streamhq?"):
		return TrackQualityMP3High
	case contains(streamURL, "/streamfl?"), contains(streamURL, "/streamhls?"):
		return TrackQualityFLAC
	default:
		return TrackQualityUnknown
	}
}

// contains is a case-insensitive substring check helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// createQualityResolver creates the appropriate quality resolver based on category.
func createQualityResolver(
	category DownloadCategory,
	zvukClient zvuk.Client,
	chapterStreams map[string]*zvuk.ChapterStreamMetadata,
) QualityResolver {
	if category == DownloadCategoryAudiobook || category == DownloadCategoryPodcast {
		return NewAudiobookQualityResolver(chapterStreams)
	}

	return NewTrackQualityResolver(zvukClient)
}

// resolveTrackQuality is a convenience method for ServiceImpl to resolve quality.
func (s *ServiceImpl) resolveTrackQuality(
	ctx context.Context,
	trackID string,
	track *zvuk.Track,
	metadata *downloadTracksMetadata,
) (*QualityResolutionResult, error) {
	desiredQuality := TrackQuality(s.cfg.Quality)
	minQuality := TrackQuality(s.cfg.MinQuality)

	resolver := createQualityResolver(metadata.category, s.zvukClient, metadata.chapterStreams)

	result, err := resolver.ResolveQuality(ctx, trackID, track, desiredQuality, minQuality)
	if err != nil {
		// Don't log context cancellation - it's expected when user presses CTRL+C.
		if !errors.Is(err, context.Canceled) {
			logger.Errorf(ctx, "Failed to resolve quality: %v", err)
		}

		return nil, err
	}

	return result, nil
}
