//nolint:err113 // Test doubles return inline errors to simulate provider failures.
package yandex

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
	mock_yandex "github.com/oshokin/zvuk-grabber/internal/service/yandex/mocks"
	"github.com/oshokin/zvuk-grabber/internal/transport/download"
)

// TestResolveAudioQuality_FLACPreferredAvailable verifies FLAC is selected when available and preferred.
func TestResolveAudioQuality_FLACPreferredAvailable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().DownloadFLACBytes(gomock.Any(), "123").Return([]byte("fLaCdata"), nil)

	s := newQualityTestService(client, media.QualityFLAC, media.QualityUnknown)

	result, err := resolveTestAudioQuality(t, s)
	require.NoError(t, err)

	want := &qualityResolutionResult{
		payload: &audioPayload{
			quality:       media.QualityFLAC,
			data:          []byte("fLaCdata"),
			contentLength: 8,
			codec:         media.CodecFLAC.String(),
		},
	}
	require.Equal(t, want, result)
}

// TestResolveAudioQuality_FLACFallbackToMP3 verifies MP3 is selected when FLAC is unavailable.
func TestResolveAudioQuality_FLACFallbackToMP3(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().DownloadFLACBytes(gomock.Any(), "123").Return(nil, errors.New("flac unavailable"))
	client.EXPECT().
		ResolveMP3Link(gomock.Any(), "123", gomock.Any()).
		Return("https://example.test/file.mp3", 320, nil)

	s := newQualityTestService(client, media.QualityFLAC, media.QualityUnknown)

	result, err := resolveTestAudioQuality(t, s)
	require.NoError(t, err)

	want := &qualityResolutionResult{
		payload: &audioPayload{
			quality:   media.QualityMP3High,
			streamURL: "https://example.test/file.mp3",
			bitrate:   320,
			codec:     media.CodecMP3.String(),
		},
	}
	require.Equal(t, want, result)
}

// TestResolveAudioQuality_FLACRequiredSkips verifies tracks are skipped when FLAC is required but unavailable.
func TestResolveAudioQuality_FLACRequiredSkips(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().DownloadFLACBytes(gomock.Any(), "123").Return(nil, errors.New("flac unavailable"))

	s := newQualityTestService(client, media.QualityFLAC, media.QualityFLAC)

	result, err := resolveTestAudioQuality(t, s)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.shouldSkip)
	assert.Error(t, result.skipReason)
}

// TestResolveAudioQuality_MP3128BelowMinimumSkips verifies 128 kbps MP3 is skipped when below the minimum quality.
func TestResolveAudioQuality_MP3128BelowMinimumSkips(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().
		ResolveMP3Link(gomock.Any(), "123", gomock.Any()).
		Return("https://example.test/file.mp3", 128, nil)

	s := newQualityTestService(client, media.QualityMP3High, media.QualityMP3High)

	result, err := resolveTestAudioQuality(t, s)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.shouldSkip)
	assert.Error(t, result.skipReason)
}

// TestResolveAudioQuality_MP3320MeetsMinimum verifies 320 kbps MP3 meets the configured minimum quality.
func TestResolveAudioQuality_MP3320MeetsMinimum(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().
		ResolveMP3Link(gomock.Any(), "123", gomock.Any()).
		Return("https://example.test/file.mp3", 320, nil)

	s := newQualityTestService(client, media.QualityMP3High, media.QualityMP3High)

	result, err := resolveTestAudioQuality(t, s)
	require.NoError(t, err)

	want := &qualityResolutionResult{
		payload: &audioPayload{
			quality:   media.QualityMP3High,
			streamURL: "https://example.test/file.mp3",
			bitrate:   320,
			codec:     media.CodecMP3.String(),
		},
	}
	require.Equal(t, want, result)
}

// TestResolveAudioQuality_MP3192DoesNotMeetHighMinimum verifies 192 kbps MP3 does not meet a high minimum quality.
func TestResolveAudioQuality_MP3192DoesNotMeetHighMinimum(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	client := mock_yandex.NewMockMusicClient(ctrl)
	client.EXPECT().
		ResolveMP3Link(gomock.Any(), "123", gomock.Any()).
		Return("https://example.test/file.mp3", 192, nil)

	s := newQualityTestService(client, media.QualityMP3High, media.QualityMP3High)

	result, err := resolveTestAudioQuality(t, s)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.shouldSkip)
	assert.Error(t, result.skipReason)
}

// TestTransferFailureIsNotRetriedOrDowngraded keeps the completed transport budget authoritative.
func TestTransferFailureIsNotRetriedOrDowngraded(t *testing.T) {
	for _, minimum := range []media.Quality{media.QualityUnknown, media.QualityFLAC} {
		ctrl := gomock.NewController(t)
		client := mock_yandex.NewMockMusicClient(ctrl)
		failure := &download.Error{Err: io.ErrUnexpectedEOF}
		client.EXPECT().DownloadFLACBytes(gomock.Any(), "123").Return(nil, failure).Times(1)

		service := newQualityTestService(client, media.QualityFLAC, minimum)
		service.cfg.APIRetryAttemptsCount = 5
		result, err := resolveTestAudioQuality(t, service)
		require.ErrorIs(t, err, failure)
		require.Nil(t, result)
	}
}

// newQualityTestService builds a service configured for quality resolution tests.
func newQualityTestService(client musicClient, quality, minQuality media.Quality) *ServiceImpl {
	return &ServiceImpl{
		cfg: &config.Config{
			Quality:    uint8(quality),
			MinQuality: uint8(minQuality),
		},
		client: client,
	}
}

// resolveTestAudioQuality resolves audio quality for a minimal track job in tests.
func resolveTestAudioQuality(t *testing.T, s *ServiceImpl) (*qualityResolutionResult, error) {
	t.Helper()

	return s.resolveAudioQuality(t.Context(), &trackJob{
		kind:  collectionTrack,
		track: &model.Track{ID: model.NewFlexibleID("123")},
	})
}
