//nolint:err113 // Quality resolution keeps compact sentinel/wrapping logic close to bitrate mapping rules.
package yandex

import (
	"context"
	"errors"
	"fmt"

	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/transport/download"
)

// qualityResolutionResult holds the outcome of audio quality resolution for a track.
type qualityResolutionResult struct {
	// payload contains resolved audio when download should proceed.
	payload *audioPayload
	// shouldSkip reports whether the track should be skipped.
	shouldSkip bool
	// skipReason explains why the track was skipped when shouldSkip is true.
	skipReason error
}

// yandexBitrate320 is the MP3 bitrate threshold for high-quality classification.
const yandexBitrate320 = 320

// qualitySkip builds a skip result with the given reason.
func qualitySkip(reason error) *qualityResolutionResult {
	return &qualityResolutionResult{
		shouldSkip: true,
		skipReason: reason,
	}
}

// qualityPayload builds a successful resolution result with the given audio payload.
func qualityPayload(payload *audioPayload) *qualityResolutionResult {
	return &qualityResolutionResult{payload: payload}
}

// resolveAudioQuality selects FLAC or MP3 audio for a track based on config and availability.
func (s *ServiceImpl) resolveAudioQuality(ctx context.Context, job *trackJob) (*qualityResolutionResult, error) {
	if job == nil || job.track == nil {
		return nil, errors.New("invalid yandex track job for quality resolution")
	}

	trackID := yandexJobTrackID(job)
	if trackID == "" {
		return nil, errors.New("invalid yandex track ID for quality resolution")
	}

	desired := media.FromConfig(s.cfg.Quality)
	minimum := media.FromConfig(s.cfg.MinQuality)

	if desired == media.QualityUnknown {
		desired = media.QualityMP3High
	}

	if s.client == nil {
		return nil, errors.New("yandex client is not configured")
	}

	if desired == media.QualityFLAC {
		result, err := s.resolveLosslessAudio(ctx, job, trackID, minimum)
		if err != nil || result.payload != nil || result.shouldSkip {
			return result, err
		}
	}

	link, bitrate, err := s.resolveMP3LinkWithRetry(ctx, trackID, s.cfg.Quality)
	if err != nil {
		return nil, err
	}

	actualQuality := media.QualityMP3Mid
	if bitrate >= yandexBitrate320 {
		actualQuality = media.QualityMP3High
	}

	if minimum != media.QualityUnknown && actualQuality < minimum {
		return qualitySkip(
			fmt.Errorf("selected MP3 bitrate %d Kbps is below min_quality %s", bitrate, minimum.Description()),
		), nil
	}

	return qualityPayload(&audioPayload{
		quality:   actualQuality,
		streamURL: link,
		bitrate:   bitrate,
		codec:     media.CodecMP3.String(),
	}), nil
}

// resolveLosslessAudio returns an empty resolution only when an availability failure permits MP3 fallback.
func (s *ServiceImpl) resolveLosslessAudio(
	ctx context.Context,
	job *trackJob,
	trackID string,
	minimum media.Quality,
) (*qualityResolutionResult, error) {
	data, err := retryValue(ctx, s, "downloading Yandex FLAC audio", func() ([]byte, error) {
		return s.client.DownloadFLACBytes(ctx, trackID)
	})
	if err == nil {
		return qualityPayload(
			&audioPayload{
				quality:       media.QualityFLAC,
				data:          data,
				contentLength: int64(len(data)),
				codec:         media.CodecFLAC.String(),
			},
		), nil
	}

	var transferErr *download.Error

	if errors.As(err, &transferErr) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}

	if minimum >= media.QualityFLAC {
		return qualitySkip(fmt.Errorf("FLAC is required but unavailable: %w", err)), nil
	}

	s.warnLosslessFallbackOnce(ctx, job, err)

	return new(qualityResolutionResult), nil
}
