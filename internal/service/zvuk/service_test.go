package zvuk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// testErrUnauthorized simulates an invalid-token error response from the API.
//
//nolint:errname,revive,staticcheck // This is a test error, not intended to be used in production.
var testErrUnauthorized = errors.New("unauthorized: invalid token")

// assertFatalExit runs fn and asserts that the custom fatal handler would exit the process.
func assertFatalExit(t *testing.T, fn func()) {
	t.Helper()

	logger.SetFatalHandler(func(code int) {
		panic(fmt.Sprintf("fatal-exit-%d", code))
	})
	defer logger.SetFatalHandler(nil)

	assert.PanicsWithValue(t, "fatal-exit-1", fn)
}

// mockURLProcessor is a mock implementation of the URLProcessor interface.
type mockURLProcessor struct{}

// ExtractDownloadItems pretends to understand URLs and dutifully returns an empty response.
func (m *mockURLProcessor) ExtractDownloadItems(
	_ context.Context,
	_ []string,
) (*ExtractDownloadItemsResponse, error) {
	return new(ExtractDownloadItemsResponse), nil
}

// DeduplicateDownloadItems is a no-op mock that passes through the incoming slice.
func (m *mockURLProcessor) DeduplicateDownloadItems(items []*DownloadItem) []*DownloadItem {
	return items
}

// mockTemplateManager is a mock implementation of the TemplateManager interface.
type mockTemplateManager struct{}

// GetTrackFilename returns a deterministic filename to keep tests predictable.
// Uses trackID from tags to ensure unique filenames in concurrent tests.
func (m *mockTemplateManager) GetTrackFilename(
	_ context.Context,
	_ bool,
	tags map[string]string,
	_ int64,
) string {
	// Use trackID to make filenames unique and avoid race conditions
	// in concurrent download tests where multiple tracks might be downloaded simultaneously.
	if trackID, ok := tags["trackID"]; ok && trackID != "" {
		return "test_track_" + trackID + ".mp3"
	}

	// Fallback for tests that don't provide trackID.
	return "test_track.mp3"
}

// GetAlbumFolderName returns a placeholder album folder name for the mock universe.
func (m *mockTemplateManager) GetAlbumFolderName(_ context.Context, _ map[string]string) string {
	return "test_album"
}

// mockTagProcessor is a mock implementation of the TagProcessor interface.
type mockTagProcessor struct{}

// WriteTags pretends the tags were written successfully for the purpose of tests.
func (m *mockTagProcessor) WriteTags(_ context.Context, _ *WriteTagsRequest) error {
	return nil
}

// partialReadCloser is a mock ReadCloser for partial reads.
type partialReadCloser struct {
	io.Reader
}

// Close is here solely to satisfy the io.ReadCloser contract in our tests.
func (p *partialReadCloser) Close() error {
	return nil
}

// slowReadCloser mocks a slow network stream.
type slowReadCloser struct {
	io.Reader

	delay time.Duration
}

// Read throttles the underlying reader to simulate slow network conditions.
func (s *slowReadCloser) Read(p []byte) (n int, err error) {
	time.Sleep(s.delay)
	return s.Reader.Read(p)
}

// Close completes the io.ReadCloser contract for the throttled reader.
func (s *slowReadCloser) Close() error { return nil }

// TestNewService tests the NewService function.
func TestNewService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := new(mockURLProcessor)
	mockTemplateManager := new(mockTemplateManager)
	mockTagProcessor := new(mockTagProcessor)

	service := NewService(
		config,
		mockClient,
		mockURLProcessor,
		mockTemplateManager,
		mockTagProcessor,
	)

	assert.NotNil(t, service)
}

// TestServiceImpl_DownloadURLs makes sure the happy path doesn't implode.
func TestServiceImpl_DownloadURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := new(mockURLProcessor)
	mockTemplateManager := new(mockTemplateManager)
	mockTagProcessor := new(mockTagProcessor)

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

	ctx := context.Background()
	urls := []string{"https://zvuk.com/track/123"}

	// This should not panic.
	service.DownloadURLs(ctx, urls)
}

// TestServiceImpl_DownloadURLs_EmptyURLs tests DownloadURLs with empty URLs.
func TestServiceImpl_DownloadURLs_EmptyURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := new(mockURLProcessor)
	mockTemplateManager := new(mockTemplateManager)
	mockTagProcessor := new(mockTagProcessor)

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

	ctx := context.Background()
	urls := []string{}

	// This should not panic.
	service.DownloadURLs(ctx, urls)
}

// TestServiceImpl_DownloadURLs_NilURLs tests DownloadURLs with nil URLs.
func TestServiceImpl_DownloadURLs_NilURLs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &config.Config{
		OutputPath: "/tmp/test",
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := new(mockURLProcessor)
	mockTemplateManager := new(mockTemplateManager)
	mockTagProcessor := new(mockTagProcessor)

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

	ctx := context.Background()

	var urls []string

	// This should not panic.
	service.DownloadURLs(ctx, urls)
}

// TestDownloadURLs_Integration_FullPipeline tests the full download pipeline with mocked client responses.
func TestDownloadURLs_Integration_FullPipeline(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

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
	mockTemplateManager := NewTemplateManager(ctx, cfg)
	mockTagProcessor := NewTagProcessor()

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
		Tracks: map[string]*zvuk.Track{
			trackID: {ID: 1337, Title: "Mercury's Retrograde Blues", ReleaseID: 420, Position: 1},
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
		GetAlbumsMetadata(gomock.Any(), albumIDs, gomock.Any()).
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
	service.DownloadURLs(ctx, urls)
}

// TestDownloadURLs_InvalidToken verifies fatal handling triggered by invalid authentication tokens.
func TestDownloadURLs_InvalidToken(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	cfg := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 1,
	}

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	mockURLProcessor := NewURLProcessor()
	mockTemplateManager := NewTemplateManager(ctx, cfg)
	mockTagProcessor := NewTagProcessor()

	urls := []string{"https://zvuk.com/track/123"}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(nil, testErrUnauthorized)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	t.Helper()

	assertFatalExit(t, func() {
		service.DownloadURLs(ctx, urls)
	})
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
	mockTemplateManager := NewTemplateManager(context.Background(), cfg)
	mockTagProcessor := NewTagProcessor()

	urls := []string{"https://zvuk.com/track/123"}

	getUserProfileResponse := &zvuk.UserProfile{Subscription: nil}

	mockClient.EXPECT().GetUserProfile(gomock.Any()).Return(getUserProfileResponse, nil)

	service := NewService(cfg, mockClient, mockURLProcessor, mockTemplateManager, mockTagProcessor)

	assertFatalExit(t, func() {
		service.DownloadURLs(context.Background(), urls)
	})
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
	mockTemplateManager := NewTemplateManager(context.Background(), cfg)
	mockTagProcessor := NewTagProcessor()

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
		Tracks: map[string]*zvuk.Track{
			trackID: {ID: 1487, Title: "The Network Gave Up on Life", ReleaseID: 228, Position: 1},
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
	mockClient.EXPECT().GetAlbumsMetadata(gomock.Any(), albumIDs, gomock.Any()).Return(getAlbumsMetadataResponse, nil)
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

	ctx := context.Background()
	// This should not panic, even though the download is incomplete.
	service.DownloadURLs(ctx, urls)
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
	mockTemplateManager := NewTemplateManager(context.Background(), cfg)
	mockTagProcessor := NewTagProcessor()

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
		Tracks: map[string]*zvuk.Track{
			trackID: {ID: 42, Title: "Енот Жарит Котлеты", ReleaseID: 1984, Position: 1},
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
	mockClient.EXPECT().GetAlbumsMetadata(gomock.Any(), albumIDs, gomock.Any()).Return(getAlbumsMetadataResponse, nil)
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

	ctx := context.Background()
	// This should not panic and should handle Cyrillic characters like a champ.
	service.DownloadURLs(ctx, urls)
}

// TestDownloadURLs_SpeedLimiting tests speed limiting with a large mock stream.
func TestDownloadURLs_SpeedLimiting(t *testing.T) {
	t.Parallel()

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
	mockTemplateManager := NewTemplateManager(context.Background(), cfg)
	mockTagProcessor := NewTagProcessor()

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
		Tracks: map[string]*zvuk.Track{
			trackID: {ID: 1488, Title: "Waiting For The Download Bar", ReleaseID: 2517, Position: 1},
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
	mockClient.EXPECT().GetAlbumsMetadata(gomock.Any(), albumIDs, gomock.Any()).Return(getAlbumsMetadataResponse, nil)
	mockClient.EXPECT().GetLabelsMetadata(gomock.Any(), labelIDs).Return(getLabelsMetadataResponse, nil)
	mockClient.EXPECT().GetStreamMetadata(gomock.Any(), trackID, gomock.Any()).Return(getStreamMetadataResponse, nil)

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

	ctx := context.Background()
	start := time.Now()

	service.DownloadURLs(ctx, urls)

	duration := time.Since(start)

	// At 512KB/s, 1MB should take ~2s, assuming the universe cooperates.
	// Note: Actual timing may vary because mocks are fast, but the throttling logic should still execute.
	assert.GreaterOrEqual(t, duration, 1*time.Second, "Download should show some evidence of throttling")
}
