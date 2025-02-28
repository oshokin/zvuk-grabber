package zvuk

import (
	"context"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

func (s *ServiceImpl) getTrackHighestQuality(ctx context.Context, track *zvuk.Track) TrackQuality {
	result := ParseQuality(track.HighestQuality)
	if result == TrackQualityUnknown {
		result = TrackQualityMP3Mid

		logger.Infof(ctx, "Failed to parse highest quality available: %s", track.HighestQuality)
	}

	return result
}

func (s *ServiceImpl) defineQualityByStreamURL(streamURL string) TrackQuality {
	v := strings.ToLower(strings.TrimSpace(streamURL))

	switch {
	case strings.Contains(v, "/stream?"):
		return TrackQualityMP3Mid
	case strings.Contains(v, "/streamhq?"):
		return TrackQualityMP3High
	case strings.Contains(v, "/streamfl?"):
		return TrackQualityFLAC
	default:
		return TrackQualityUnknown
	}
}
