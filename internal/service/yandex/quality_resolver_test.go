//nolint:err113 // Test doubles return inline errors to simulate provider failures.
package yandex

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	yandexclient "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// qualityTestClient stubs download methods for quality resolution tests.
type qualityTestClient struct {
	// downloadFLACBytes is the callback used to simulate FLAC availability checks.
	downloadFLACBytes func(ctx context.Context, trackID string) ([]byte, error)
	// resolveMP3Link is the callback used to simulate MP3 link resolution.
	resolveMP3Link func(ctx context.Context, trackID string, preferredQuality uint8) (string, int, error)
}

// TrackInfo is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) TrackInfo(context.Context, string) (*model.Track, error) {
	return nil, errors.New("not implemented")
}

// AlbumWithTracks is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) AlbumWithTracks(context.Context, string) (*model.Album, error) {
	return nil, errors.New("not implemented")
}

// UsersPlaylist is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) UsersPlaylist(context.Context, string, string) (*model.Playlist, error) {
	return nil, errors.New("not implemented")
}

// PlaylistByUUID is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) PlaylistByUUID(context.Context, string) (*model.Playlist, error) {
	return nil, errors.New("not implemented")
}

// DownloadFLACBytes delegates FLAC availability checks to the configured test callback.
func (c *qualityTestClient) DownloadFLACBytes(ctx context.Context, trackID string) ([]byte, error) {
	if c.downloadFLACBytes == nil {
		return nil, errors.New("downloadFLACBytes is not configured")
	}

	return c.downloadFLACBytes(ctx, trackID)
}

// ResolveMP3Link delegates MP3 link resolution to the configured test callback.
func (c *qualityTestClient) ResolveMP3Link(
	ctx context.Context,
	trackID string,
	preferredQuality uint8,
) (string, int, error) {
	if c.resolveMP3Link == nil {
		return "", 0, errors.New("resolveMP3Link is not configured")
	}

	return c.resolveMP3Link(ctx, trackID, preferredQuality)
}

// OpenAudio is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) OpenAudio(context.Context, string) (*yandexclient.RemoteFile, error) {
	return nil, errors.New("not implemented")
}

// DownloadCoverBytes is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) DownloadCoverBytes(context.Context, string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// TrackLyrics is a stub implementation that reports the method as not implemented.
func (c *qualityTestClient) TrackLyrics(context.Context, string) (*model.TrackLyrics, error) {
	return nil, errors.New("not implemented")
}

// newQualityTestService builds a service configured for quality resolution tests.
func newQualityTestService(client *qualityTestClient, quality, minQuality media.Quality) *ServiceImpl {
	return &ServiceImpl{
		cfg: &config.Config{
			Quality:    uint8(quality),
			MinQuality: uint8(minQuality),
		},
		client: client,
	}
}

// resolveTestAudioQuality resolves audio quality for a minimal track job in tests.
func resolveTestAudioQuality(s *ServiceImpl) (*qualityResolutionResult, error) {
	return s.resolveAudioQuality(context.Background(), &trackJob{
		kind:  collectionTrack,
		track: &model.Track{ID: model.NewFlexibleID("123")},
	})
}

// TestResolveAudioQuality_FLACPreferredAvailable verifies FLAC is selected when available and preferred.
func TestResolveAudioQuality_FLACPreferredAvailable(t *testing.T) {
	t.Parallel()

	client := &qualityTestClient{
		downloadFLACBytes: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("fLaCdata"), nil
		},
	}
	s := newQualityTestService(client, media.QualityFLAC, media.QualityUnknown)

	result, err := resolveTestAudioQuality(s)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.payload)
	assert.Equal(t, media.QualityFLAC, result.payload.quality)
	assert.False(t, result.shouldSkip)
}

// TestResolveAudioQuality_FLACFallbackToMP3 verifies MP3 is selected when FLAC is unavailable.
func TestResolveAudioQuality_FLACFallbackToMP3(t *testing.T) {
	t.Parallel()

	client := &qualityTestClient{
		downloadFLACBytes: func(_ context.Context, _ string) ([]byte, error) {
			return nil, errors.New("flac unavailable")
		},
		resolveMP3Link: func(_ context.Context, _ string, _ uint8) (string, int, error) {
			return "https://example.test/file.mp3", 320, nil
		},
	}
	s := newQualityTestService(client, media.QualityFLAC, media.QualityUnknown)

	result, err := resolveTestAudioQuality(s)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.payload)
	assert.Equal(t, media.QualityMP3High, result.payload.quality)
	assert.False(t, result.shouldSkip)
}

// TestResolveAudioQuality_FLACRequiredSkips verifies tracks are skipped when FLAC is required but unavailable.
func TestResolveAudioQuality_FLACRequiredSkips(t *testing.T) {
	t.Parallel()

	client := &qualityTestClient{
		downloadFLACBytes: func(_ context.Context, _ string) ([]byte, error) {
			return nil, errors.New("flac unavailable")
		},
	}
	s := newQualityTestService(client, media.QualityFLAC, media.QualityFLAC)

	result, err := resolveTestAudioQuality(s)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.shouldSkip)
	assert.Error(t, result.skipReason)
}

// TestResolveAudioQuality_MP3128BelowMinimumSkips verifies 128 kbps MP3 is skipped when below the minimum quality.
func TestResolveAudioQuality_MP3128BelowMinimumSkips(t *testing.T) {
	t.Parallel()

	client := &qualityTestClient{
		resolveMP3Link: func(_ context.Context, _ string, _ uint8) (string, int, error) {
			return "https://example.test/file.mp3", 128, nil
		},
	}
	s := newQualityTestService(client, media.QualityMP3High, media.QualityMP3High)

	result, err := resolveTestAudioQuality(s)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.shouldSkip)
	assert.Error(t, result.skipReason)
}

// TestResolveAudioQuality_MP3320MeetsMinimum verifies 320 kbps MP3 meets the configured minimum quality.
func TestResolveAudioQuality_MP3320MeetsMinimum(t *testing.T) {
	t.Parallel()

	client := &qualityTestClient{
		resolveMP3Link: func(_ context.Context, _ string, _ uint8) (string, int, error) {
			return "https://example.test/file.mp3", 320, nil
		},
	}
	s := newQualityTestService(client, media.QualityMP3High, media.QualityMP3High)

	result, err := resolveTestAudioQuality(s)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.payload)
	assert.Equal(t, media.QualityMP3High, result.payload.quality)
	assert.Equal(t, "https://example.test/file.mp3", result.payload.streamURL)
	assert.False(t, result.shouldSkip)
}

// TestResolveAudioQuality_MP3192DoesNotMeetHighMinimum verifies 192 kbps MP3 does not meet a high minimum quality.
func TestResolveAudioQuality_MP3192DoesNotMeetHighMinimum(t *testing.T) {
	t.Parallel()

	client := &qualityTestClient{
		resolveMP3Link: func(_ context.Context, _ string, _ uint8) (string, int, error) {
			return "https://example.test/file.mp3", 192, nil
		},
	}
	s := newQualityTestService(client, media.QualityMP3High, media.QualityMP3High)

	result, err := resolveTestAudioQuality(s)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.shouldSkip)
	assert.Error(t, result.skipReason)
}
