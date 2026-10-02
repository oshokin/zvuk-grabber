package zvuk

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// partialReadCloser is a fake ReadCloser for partial reads.
type partialReadCloser struct {
	// Reader provides the underlying byte stream.
	io.Reader
}

// slowReadCloser fakes a slow network stream.
type slowReadCloser struct {
	// Reader provides the underlying byte stream.
	io.Reader

	// delay is the per-read sleep duration to simulate network latency.
	delay time.Duration
}

// errUnauthorizedTest simulates an invalid-token error response from the API.
var errUnauthorizedTest = errors.New("unauthorized: invalid token")

// TestNewService tests the NewService function.
func TestNewService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: t.TempDir(),
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := newStubURLProcessor(ctrl)
	mockTemplateManager := newStubTemplateManager(ctrl)
	mockTagProcessor := newStubTagProcessor(ctrl)

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	assert.NotNil(t, service)
}

// Close is here solely to satisfy the io.ReadCloser contract in our tests.
func (p *partialReadCloser) Close() error {
	return nil
}

// Read throttles the underlying reader to simulate slow network conditions.
func (s *slowReadCloser) Read(p []byte) (n int, err error) {
	time.Sleep(s.delay)
	return s.Reader.Read(p)
}

// Close completes the io.ReadCloser contract for the throttled reader.
func (s *slowReadCloser) Close() error { return nil }

// TestServiceImpl_DownloadURLs makes sure the happy path doesn't implode.
func TestServiceImpl_DownloadURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: t.TempDir(),
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := newStubURLProcessor(ctrl)
	mockTemplateManager := newStubTemplateManager(ctrl)
	mockTagProcessor := newStubTagProcessor(ctrl)

	// Setup mock expectations.
	getUserProfileResponse := &zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil).AnyTimes()

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	ctx := t.Context()
	urls := []string{"https://zvuk.com/track/123"}

	// This should not panic.
	require.NoError(t, service.DownloadURLs(ctx, urls))
}

// TestServiceImpl_DownloadURLs_EmptyURLs tests DownloadURLs with empty URLs.
func TestServiceImpl_DownloadURLs_EmptyURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: t.TempDir(),
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := newStubURLProcessor(ctrl)
	mockTemplateManager := newStubTemplateManager(ctrl)
	mockTagProcessor := newStubTagProcessor(ctrl)

	// Setup mock expectations for empty URLs.
	getUserProfileResponse := &zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil).AnyTimes()

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	ctx := t.Context()
	urls := []string{}

	// This should not panic.
	require.NoError(t, service.DownloadURLs(ctx, urls))
}

// TestServiceImpl_DownloadURLs_NilURLs tests DownloadURLs with nil URLs.
func TestServiceImpl_DownloadURLs_NilURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: t.TempDir(),
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := newStubURLProcessor(ctrl)
	mockTemplateManager := newStubTemplateManager(ctrl)
	mockTagProcessor := newStubTagProcessor(ctrl)

	// Setup mock expectations for nil URLs.
	getUserProfileResponse := &zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{
			Title:      "Premium",
			Expiration: 1234567890,
		},
	}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil).AnyTimes()

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	ctx := t.Context()

	var urls []string

	// This should not panic.
	require.NoError(t, service.DownloadURLs(ctx, urls))
}

// TestDownloadURLs_Integration_FullPipeline tests the full download pipeline with mocked client responses.
func TestDownloadURLs_Integration_FullPipeline(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	cfg := &config.Config{
		OutputPath:             t.TempDir(),
		Quality:                1,
		ReplaceTracks:          false,
		ParsedLogLevel:         zapcore.ErrorLevel,
		ParsedMaxDownloadPause: 1 * time.Nanosecond, // Basically instant but not zero.
		MaxConcurrentDownloads: 1,
		TrackFilenameTemplate:  "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:    "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := NewURLProcessor()
	mockTemplateManager := media.NewTemplateManager(ctx, cfg)
	mockTagProcessor := media.NewTagProcessor()

	trackID := "1337"
	albumID := "420"
	labelID := "69"
	urls := []string{"https://zvuk.com/track/" + trackID}
	trackIDs := []string{trackID}
	albumIDs := []string{albumID}
	labelIDs := []string{labelID}

	// Setup expectations.
	getUserProfileResponse := &zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{Title: "Ultra Mega Premium", Expiration: 9001},
	}

	getTracksMetadataResponse := map[string]*zvuk.Track{
		trackID: {ID: 1337, Title: "Mercury's Retrograde Blues", ReleaseID: 420, Position: 1},
	}

	getAlbumsMetadataResponse := &zvuk.GetAlbumsMetadataResponse{
		Releases: map[string]*zvuk.Release{
			albumID: {
				ID:          420,
				Title:       "Existential Crisis at 3 AM",
				Date:        1609459200000, // Some random timestamp that doesn't matter anyway.
				ArtistNames: []string{"The Philosophizing Beavers"},
				LabelID:     69,
			},
		},
	}

	getLabelsMetadataResponse := map[string]*zvuk.Label{
		labelID: {Title: "Sad Penguins Records"},
	}

	getStreamMetadataResponse := &zvuk.StreamMetadata{
		Stream: "/stream/" + trackID,
	}

	mockTrackContent := []byte("definitely real audio data and not just bytes")

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil).Times(1)
	mockClient.EXPECT().GetTracksMetadata(gomock.Any(), trackIDs).Return(getTracksMetadataResponse, nil).Times(1)
	mockClient.EXPECT().
		GetAlbumsMetadata(gomock.Any(), albumIDs).
		Return(getAlbumsMetadataResponse, nil).
		Times(1)
	mockClient.EXPECT().GetLabelsMetadata(gomock.Any(), labelIDs).Return(getLabelsMetadataResponse, nil).Times(1)
	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), trackID, gomock.Any()).
		Return(getStreamMetadataResponse, nil).
		Times(1)

	streamReader := io.NopCloser(bytes.NewReader(mockTrackContent))
	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       streamReader,
		TotalBytes: int64(len(mockTrackContent)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/stream/"+trackID).
		Return(fetchTrackResult, nil).
		Times(1)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	// This should not panic and should complete successfully.
	require.NoError(t, service.DownloadURLs(ctx, urls))
}

// TestDownloadURLs_InvalidToken verifies fatal handling triggered by invalid authentication tokens.
func TestDownloadURLs_InvalidToken(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := t.Context()

	cfg := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 1,
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := NewURLProcessor()
	mockTemplateManager := media.NewTemplateManager(ctx, cfg)
	mockTagProcessor := media.NewTagProcessor()

	urls := []string{"https://zvuk.com/track/123"}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(nil, errUnauthorizedTest)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	t.Helper()

	require.ErrorContains(t, service.DownloadURLs(ctx, urls), "failed to retrieve user profile")
}

// TestDownloadURLs_ExpiredSubscription ensures the service exits when subscription data is missing.
func TestDownloadURLs_ExpiredSubscription(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 1,
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := NewURLProcessor()
	mockTemplateManager := media.NewTemplateManager(t.Context(), cfg)
	mockTagProcessor := media.NewTagProcessor()

	urls := []string{"https://zvuk.com/track/123"}

	getUserProfileResponse := &zvuk.UserProfile{Subscription: nil}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	require.ErrorContains(t, service.DownloadURLs(t.Context(), urls), "active subscription")
}

// TestDownloadURLs_PartialDownload tests partial stream via mocked ReadCloser that fails midway.
func TestDownloadURLs_PartialDownload(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		OutputPath:             t.TempDir(),
		Quality:                1,
		ParsedMaxDownloadPause: 1 * time.Nanosecond, // Basically instant but not zero.
		MaxConcurrentDownloads: 1,
		TrackFilenameTemplate:  "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:    "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := NewURLProcessor()
	mockTemplateManager := media.NewTemplateManager(t.Context(), cfg)
	mockTagProcessor := media.NewTagProcessor()

	trackID := "1487"
	albumID := "228"
	labelID := "1312"
	urls := []string{"https://zvuk.com/track/" + trackID}
	trackIDs := []string{trackID}
	albumIDs := []string{albumID}
	labelIDs := []string{labelID}

	// Setup expectations for a download that gives up halfway through, like my motivation on Mondays.
	getUserProfileResponse := &zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{Title: "Basic Subscription That Kinda Works"},
	}

	getTracksMetadataResponse := map[string]*zvuk.Track{
		trackID: {ID: 1487, Title: "The Network Gave Up on Life", ReleaseID: 228, Position: 1},
	}

	getAlbumsMetadataResponse := &zvuk.GetAlbumsMetadataResponse{
		Releases: map[string]*zvuk.Release{
			albumID: {
				ID:          228,
				Title:       "Incomplete Downloads: A Tragedy",
				Date:        1234567890000,
				ArtistNames: []string{"Unstable Connection Orchestra"},
				LabelID:     1312,
			},
		},
	}

	getLabelsMetadataResponse := map[string]*zvuk.Label{
		labelID: {Title: "Buffering Records"},
	}

	getStreamMetadataResponse := &zvuk.StreamMetadata{
		Stream: "/stream/" + trackID,
	}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil)
	mockClient.EXPECT().GetTracksMetadata(gomock.Any(), trackIDs).Return(getTracksMetadataResponse, nil)
	mockClient.EXPECT().GetAlbumsMetadata(gomock.Any(), albumIDs).Return(getAlbumsMetadataResponse, nil)
	mockClient.EXPECT().GetLabelsMetadata(gomock.Any(), labelIDs).Return(getLabelsMetadataResponse, nil)
	mockClient.EXPECT().GetStreamMetadata(gomock.Any(), trackID, gomock.Any()).Return(getStreamMetadataResponse, nil)

	// Mock partial reader that returns EOF early because the internet is a lie.
	fullContent := []byte("this should be full audio but nope")
	partialReader := &partialReadCloser{Reader: bytes.NewReader(fullContent[:len(fullContent)/2])}

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       partialReader,
		TotalBytes: int64(len(fullContent)),
	}

	mockClient.EXPECT().FetchTrack(gomock.Any(), "/stream/"+trackID).Return(fetchTrackResult, nil)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	ctx := t.Context()
	// This should not panic, even though the download is incomplete.
	require.Error(t, service.DownloadURLs(ctx, urls))
}

// TestDownloadURLs_NonASCIIFilename tests non-ASCII handling.
func TestDownloadURLs_NonASCIIFilename(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		OutputPath:             t.TempDir(),
		Quality:                1,
		ParsedMaxDownloadPause: 1 * time.Nanosecond,
		MaxConcurrentDownloads: 1,
		TrackFilenameTemplate:  "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:    "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := NewURLProcessor()
	mockTemplateManager := media.NewTemplateManager(t.Context(), cfg)
	mockTagProcessor := media.NewTagProcessor()

	trackID := "42"
	albumID := "1984"
	labelID := "777"
	urls := []string{"https://zvuk.com/track/" + trackID}
	trackIDs := []string{trackID}
	albumIDs := []string{albumID}
	labelIDs := []string{labelID}

	// Setup expectations for Unicode that Windows will probably mess up somehow.
	getUserProfileResponse := &zvuk.UserProfile{
		Subscription: &zvuk.UserSubscription{Title: "Подписка Которая Работает"},
	}

	getTracksMetadataResponse := map[string]*zvuk.Track{
		trackID: {ID: 42, Title: "Енот Жарит Котлеты", ReleaseID: 1984, Position: 1},
	}

	getAlbumsMetadataResponse := &zvuk.GetAlbumsMetadataResponse{
		Releases: map[string]*zvuk.Release{
			albumID: {
				ID:          1984,
				Title:       "Философия Грустного Хомяка",
				Date:        1337000000000,
				ArtistNames: []string{"Депрессивный Бобёр и Его Друзья"},
				LabelID:     777,
			},
		},
	}

	getLabelsMetadataResponse := map[string]*zvuk.Label{
		labelID: {Title: "Лейбл Экзистенциальных Кризисов"},
	}

	getStreamMetadataResponse := &zvuk.StreamMetadata{
		Stream: "/stream/" + trackID,
	}

	mockTrackContent := []byte("аудио которое точно не сломает кодировку")

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil)
	mockClient.EXPECT().GetTracksMetadata(gomock.Any(), trackIDs).Return(getTracksMetadataResponse, nil)
	mockClient.EXPECT().GetAlbumsMetadata(gomock.Any(), albumIDs).Return(getAlbumsMetadataResponse, nil)
	mockClient.EXPECT().GetLabelsMetadata(gomock.Any(), labelIDs).Return(getLabelsMetadataResponse, nil)
	mockClient.EXPECT().GetStreamMetadata(gomock.Any(), trackID, gomock.Any()).Return(getStreamMetadataResponse, nil)

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       io.NopCloser(bytes.NewReader(mockTrackContent)),
		TotalBytes: int64(len(mockTrackContent)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/stream/"+trackID).
		Return(fetchTrackResult, nil)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	ctx := t.Context()
	// This should not panic and should handle Cyrillic characters like a champ.
	require.NoError(t, service.DownloadURLs(ctx, urls))
}

// TestDownloadURLs_SpeedLimiting tests speed limiting with a large mock stream.
func TestDownloadURLs_SpeedLimiting(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := &config.Config{
			OutputPath:               t.TempDir(),
			Quality:                  1,
			ParsedDownloadSpeedLimit: 512 * 1024, // 512 KB/s because we're not animals.
			ParsedMaxDownloadPause:   1 * time.Nanosecond,
			MaxConcurrentDownloads:   1,
			TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
			AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		}

		mockClient := mock_zvuk_client.NewMockClient(ctrl)
		mockURLProcessor := NewURLProcessor()
		mockTemplateManager := media.NewTemplateManager(t.Context(), cfg)
		mockTagProcessor := media.NewTagProcessor()

		trackID := "1488"
		albumID := "2517"
		labelID := "404"
		urls := []string{"https://zvuk.com/track/" + trackID}
		trackIDs := []string{trackID}
		albumIDs := []string{albumID}
		labelIDs := []string{labelID}

		// Setup expectations for throttled downloads because bandwidth costs money apparently.
		getUserProfileResponse := &zvuk.UserProfile{
			Subscription: &zvuk.UserSubscription{Title: "Premium But Still Slow"},
		}

		getTracksMetadataResponse := map[string]*zvuk.Track{
			trackID: {ID: 1488, Title: "Waiting For The Download Bar", ReleaseID: 2517, Position: 1},
		}

		getAlbumsMetadataResponse := &zvuk.GetAlbumsMetadataResponse{
			Releases: map[string]*zvuk.Release{
				albumID: {
					ID:          2517,
					Title:       "The Art of Patience",
					Date:        9001000000000,
					ArtistNames: []string{"Dial-Up Memories"},
					LabelID:     404,
				},
			},
		}

		getLabelsMetadataResponse := map[string]*zvuk.Label{
			labelID: {Title: "Error Not Found Records"},
		}

		getStreamMetadataResponse := &zvuk.StreamMetadata{
			Stream: "/stream/" + trackID,
		}

		mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil)
		mockClient.EXPECT().GetTracksMetadata(gomock.Any(), trackIDs).Return(getTracksMetadataResponse, nil)
		mockClient.EXPECT().
			GetAlbumsMetadata(gomock.Any(), albumIDs).
			Return(getAlbumsMetadataResponse, nil)
		mockClient.EXPECT().GetLabelsMetadata(gomock.Any(), labelIDs).Return(getLabelsMetadataResponse, nil)
		mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), trackID, gomock.Any()).
			Return(getStreamMetadataResponse, nil)

		// Large content to test limiting, like downloading on rural internet.
		mockTrackContent := make([]byte, 1024*1024) // 1MB that feels like 1GB.
		slowReader := &slowReadCloser{Reader: bytes.NewReader(mockTrackContent), delay: 100 * time.Millisecond}

		fetchTrackResult := &zvuk.FetchTrackResult{
			Body:       slowReader,
			TotalBytes: int64(len(mockTrackContent)),
		}

		mockClient.EXPECT().
			FetchTrack(gomock.Any(), "/stream/"+trackID).
			Return(fetchTrackResult, nil)

		service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

		ctx := t.Context()
		start := time.Now()

		require.NoError(t, service.DownloadURLs(ctx, urls))

		duration := time.Since(start)

		// At 512KB/s, 1MB should take ~2s, assuming the universe cooperates.
		// Note: Actual timing may vary because mocks are fast, but the throttling logic should still execute.
		assert.GreaterOrEqual(t, duration, 1*time.Second, "Download should show some evidence of throttling")
	})
}
