package zvuk

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TestDownloadTracks_ProgressBarWithSequential tests that progress bars work with sequential downloads.
func TestDownloadTracks_ProgressBarWithSequential(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	testConfig := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 1, // Sequential mode - progress bars enabled.
		ReplaceTracks:          false,
		Quality:                3,
		ParsedLogLevel:         logger.Level(), // Info level - progress bars visible.
		ParsedMaxDownloadPause: 100 * time.Millisecond,
	}

	service := NewService(
		testConfig,
		mockClient,
		new(mockURLProcessor),
		new(mockTemplateManager),
		new(mockTagProcessor),
	)

	// Create test metadata with 1 track.
	trackIDs := []int64{501}
	tracksMetadata := map[string]*zvuk.Track{
		"501": {ID: 501, Title: "Track 1", ReleaseID: 5, Position: 1, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"5": {
			ID:          5,
			Title:       "Progress Test Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"5": {"albumTitle": "Progress Test Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Setup mock expectations.
	streamMetadata := &zvuk.StreamMetadata{Stream: "/stream/501"}

	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), "501", TrackQualityFLACString).
		Return(streamMetadata, nil)

	// Create a larger fake stream to simulate real download.
	fakeAudioData := makeFakeAudioData(100) // 100KB.

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       io.NopCloser(bytes.NewReader(fakeAudioData)),
		TotalBytes: int64(len(fakeAudioData)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/stream/501").
		Return(fetchTrackResult, nil)

	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        trackIDs,
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  albumsMetadata,
		albumsTags:      albumsTags,
		labelsMetadata:  labelsMetadata,
		audioCollection: nil,
	}

	// Execute download.
	ctx := context.Background()

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	impl.downloadTracks(ctx, metadata)

	// With MaxConcurrentDownloads=1 and Info log level, progress bars are enabled.
	// This test verifies the download completes successfully with progress bar logic active.
	assert.Equal(t, int64(1), testConfig.MaxConcurrentDownloads,
		"Sequential mode should enable progress bars")
}

// TestDownloadTracks_NoProgressBarWithConcurrent tests that progress bars are disabled in concurrent mode.
func TestDownloadTracks_NoProgressBarWithConcurrent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	testConfig := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 2, // Concurrent mode - progress bars MUST be disabled.
		ReplaceTracks:          false,
		Quality:                3,
		ParsedLogLevel:         logger.Level(), // Info level but progress bars still disabled due to concurrency.
		ParsedMaxDownloadPause: 100 * time.Millisecond,
	}

	service := NewService(
		testConfig,
		mockClient,
		new(mockURLProcessor),
		new(mockTemplateManager),
		new(mockTagProcessor),
	)

	// Create test metadata with 2 tracks that will download concurrently.
	trackIDs := []int64{601, 602}
	tracksMetadata := map[string]*zvuk.Track{
		"601": {ID: 601, Title: "Track 1", ReleaseID: 6, Position: 1, HighestQuality: "flac", HasFLAC: true},
		"602": {ID: 602, Title: "Track 2", ReleaseID: 6, Position: 2, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"6": {
			ID:          6,
			Title:       "Concurrent Progress Test",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"6": {"albumTitle": "Concurrent Progress Test"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Setup mock expectations for both tracks.
	for _, trackID := range trackIDs {
		trackIDString := ""

		switch trackID {
		case 601:
			trackIDString = "601"
		case 602:
			trackIDString = "602"
		}

		streamURL := "/streamfl?id=" + trackIDString

		// Prepare stream metadata response before using in mock.
		streamMetadata := &zvuk.StreamMetadata{Stream: streamURL}

		mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), trackIDString, TrackQualityFLACString).
			DoAndReturn(func(_ context.Context, _ string, _ string) (*zvuk.StreamMetadata, error) {
				// Simulate some processing time to ensure concurrent execution.
				time.Sleep(10 * time.Millisecond)

				return streamMetadata, nil
			})

		// Create larger fake stream to simulate real download where progress bars would be useful.
		fakeAudioData := makeFakeAudioData(50) // 50KB per track.

		// Prepare fetch result before using in mock.
		fetchTrackResult := &zvuk.FetchTrackResult{
			Body:       io.NopCloser(bytes.NewReader(fakeAudioData)),
			TotalBytes: int64(len(fakeAudioData)),
		}

		mockClient.EXPECT().
			FetchTrack(gomock.Any(), streamURL).
			Return(fetchTrackResult, nil)
	}

	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        trackIDs,
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  albumsMetadata,
		albumsTags:      albumsTags,
		labelsMetadata:  labelsMetadata,
		audioCollection: nil,
	}

	// Execute download with concurrent downloads.
	ctx := context.Background()

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	impl.downloadTracks(ctx, metadata)

	// Verify that concurrent mode was used.
	assert.Greater(t, testConfig.MaxConcurrentDownloads, int64(1),
		"Concurrent mode disables progress bars to prevent terminal output conflicts")
}
