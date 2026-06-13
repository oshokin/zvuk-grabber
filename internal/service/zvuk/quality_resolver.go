package zvuk

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	// zvukClient is the API client used to fetch stream metadata.
	zvukClient zvuk.Client
}

// audiobookQualityResolver handles quality resolution for audiobook chapters.
type audiobookQualityResolver struct {
	// chapterStreams contains pre-fetched stream URLs keyed by chapter ID.
	chapterStreams map[string]*zvuk.StreamQualities
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
	if result := skipBelowMinimumQuality(ctx, "Track", finalQuality, minQuality); result != nil {
		return result, nil
	}

	// Fetch stream metadata from API.
	streamMetadata, err := r.zvukClient.GetStreamMetadata(ctx, trackID, finalQuality.AsStreamURLParameterValue())
	if err != nil {
		return nil, fmt.Errorf("failed to get stream metadata: %w", err)
	}

	return qualityResult(finalQuality, streamMetadata.Stream), nil
}

// NewAudiobookQualityResolver creates a resolver for audiobook chapters.
func NewAudiobookQualityResolver(chapterStreams map[string]*zvuk.StreamQualities) QualityResolver {
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
	if result := skipBelowMinimumQuality(ctx, "Chapter", highestAvailable, minQuality); result != nil {
		return result, nil
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

	return qualityResult(finalQuality, streamURL), nil
}

// skipBelowMinimumQuality returns a skip result when quality is below the configured minimum.
func skipBelowMinimumQuality(
	ctx context.Context,
	subject string,
	quality TrackQuality,
	minimum TrackQuality,
) *QualityResolutionResult {
	if minimum == TrackQualityUnknown || quality >= minimum {
		return nil
	}

	logger.Warnf(ctx, "%s quality %s is below minimum threshold %s, skipping", subject, quality, minimum)

	return &QualityResolutionResult{
		ShouldSkip: true,
		SkipReason: fmt.Errorf("%w: %s below %s", ErrQualityBelowThreshold, quality, minimum),
	}
}

// qualityResult builds a successful quality resolution result from quality and stream URL.
func qualityResult(quality TrackQuality, streamURL string) *QualityResolutionResult {
	if actual := defineQualityByStreamURL(streamURL); actual != TrackQualityUnknown {
		quality = actual
	}

	return &QualityResolutionResult{Quality: quality, StreamURL: streamURL}
}

// getHighestAvailableQuality determines the highest available quality from chapter stream metadata.
func getHighestAvailableQuality(streamMetadata *zvuk.StreamQualities) TrackQuality {
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
	streamMetadata *zvuk.StreamQualities,
	desiredQuality TrackQuality,
) string {
	switch desiredQuality {
	case TrackQualityFLAC:
		if streamMetadata.FLAC != "" {
			return streamMetadata.FLAC
		}

		fallthrough
	case TrackQualityMP3High:
		if streamMetadata.High != "" {
			return streamMetadata.High
		}
	}

	return streamMetadata.Mid
}

// defineQualityByStreamURL determines quality by analyzing the stream URL pattern.
func defineQualityByStreamURL(streamURL string) TrackQuality {
	switch {
	case strings.Contains(streamURL, "/stream?"):
		return TrackQualityMP3Mid
	case strings.Contains(streamURL, "/streamhq?"):
		return TrackQualityMP3High
	case strings.Contains(streamURL, "/streamfl?"), strings.Contains(streamURL, "/streamhls?"):
		return TrackQualityFLAC
	default:
		return TrackQualityUnknown
	}
}

// createQualityResolver creates the appropriate quality resolver based on category.
func createQualityResolver(
	category DownloadCategory,
	zvukClient zvuk.Client,
	chapterStreams map[string]*zvuk.StreamQualities,
) QualityResolver {
	if category.IsChapterCollection() {
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
	var (
		desiredQuality = TrackQuality(s.cfg.Quality)
		minQuality     = TrackQuality(s.cfg.MinQuality)
		resolver       = createQualityResolver(metadata.category, s.zvukClient, metadata.chapterStreamsMetadata)
	)

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
