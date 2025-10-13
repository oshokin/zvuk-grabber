package zvuk

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TestDownloadTracks_PartFileHandling tests that .part files are used for atomic downloads.
func TestDownloadTracks_PartFileHandling(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	tempDir := t.TempDir()

	testConfig := &config.Config{
		OutputPath:             tempDir,
		MaxConcurrentDownloads: 1,
		ReplaceTracks:          false,
		Quality:                3,
		ParsedLogLevel:         logger.Level(),
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
	trackIDs := []int64{701}
	tracksMetadata := map[string]*zvuk.Track{
		"701": {ID: 701, Title: "Track 1", ReleaseID: 7, Position: 1, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"7": {
			ID:          7,
			Title:       "Part File Test Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"7": {"albumTitle": "Part File Test Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Setup mock expectations.
	streamMetadata := &zvuk.StreamMetadata{Stream: "/stream/701"}

	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), "701", TrackQualityFLACString).
		Return(streamMetadata, nil)

	// Create fake audio data.
	fakeAudioData := []byte("complete audio file content")

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       io.NopCloser(bytes.NewReader(fakeAudioData)),
		TotalBytes: int64(len(fakeAudioData)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/stream/701").
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

	// Verify that .part file does NOT exist (was renamed to final file).
	partFiles := findPartFiles(t, tempDir)
	assert.Empty(t, partFiles, ".part files should be cleaned up after successful download")

	// Verify that final file DOES exist and content is correct.
	audioFiles := findAudioFiles(t, tempDir)
	assert.NotEmpty(t, audioFiles, "Final track file should exist after download")

	if len(audioFiles) > 0 {
		content, err := os.ReadFile(audioFiles[0])
		require.NoError(t, err, "Failed to read downloaded file")
		assert.Equal(t, fakeAudioData, content, "Downloaded file content should match source data")
	}
}

// TestDownloadTracks_PartFileCleanupOnFailure tests that .part files are cleaned up when download fails.
func TestDownloadTracks_PartFileCleanupOnFailure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	tempDir := t.TempDir()

	testConfig := &config.Config{
		OutputPath:             tempDir,
		MaxConcurrentDownloads: 1,
		ReplaceTracks:          false,
		Quality:                3,
		ParsedLogLevel:         logger.Level(),
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
	trackIDs := []int64{801}
	tracksMetadata := map[string]*zvuk.Track{
		"801": {ID: 801, Title: "Failed Track", ReleaseID: 8, Position: 1, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"8": {
			ID:          8,
			Title:       "Failed Download Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"8": {"albumTitle": "Failed Download Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Setup mock expectations for a failed download.
	streamMetadata := &zvuk.StreamMetadata{Stream: "/stream/801"}

	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), "801", TrackQualityFLACString).
		Return(streamMetadata, nil)

	// Mock returns partial data (50% of expected size).
	fullContent := []byte("this is supposed to be 100 bytes of audio data but network failed")
	partialReader := &partialReadCloser{Reader: bytes.NewReader(fullContent[:len(fullContent)/2])}

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       partialReader,
		TotalBytes: int64(len(fullContent)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/stream/801").
		Return(fetchTrackResult, nil) // Expects full size but only returns half.

	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        trackIDs,
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  albumsMetadata,
		albumsTags:      albumsTags,
		labelsMetadata:  labelsMetadata,
		audioCollection: nil,
	}

	// Execute download (should fail due to incomplete data).
	ctx := context.Background()

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	impl.downloadTracks(ctx, metadata)

	// Small delay to ensure defer cleanup has completed (especially on Windows).
	time.Sleep(50 * time.Millisecond)

	// Verify that NO .part files remain (cleaned up on failure).
	partFiles := findPartFiles(t, tempDir)
	assert.Empty(t, partFiles, ".part files should be cleaned up after failed download")

	// Verify that NO final files exist either (incomplete download was rejected).
	audioFiles := findAudioFiles(t, tempDir)
	assert.Empty(t, audioFiles, "No audio files should exist after failed download")
}
