package zvuk

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
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

// TestDownloadTracks_DryRunMode tests that dry-run mode previews without downloading files.
func TestDownloadTracks_DryRunMode(t *testing.T) {
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
		DownloadLyrics:         true,
		DryRun:                 true, // Enable dry-run mode.
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

	// Create test metadata.
	trackID := int64(950)
	trackIDString := "950"
	tracksMetadata := map[string]*zvuk.Track{
		trackIDString: {
			ID:             trackID,
			Title:          "Dry-Run Test Track",
			ReleaseID:      95,
			Position:       1,
			HighestQuality: "flac",
			HasFLAC:        true,
			Lyrics:         true,
		},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"95": {
			ID:          95,
			Title:       "Dry-Run Test Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20250101,
			LabelID:     999,
			TrackIDs:    []int64{trackID},
		},
	}
	albumsTags := map[string]map[string]string{
		"95": {"albumTitle": "Dry-Run Test Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Setup mock expectations.
	streamMetadata := &zvuk.StreamMetadata{Stream: "/streamfl?id=" + trackIDString}

	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), trackIDString, "flac").
		Return(streamMetadata, nil)

	// Create fake audio data.
	fakeAudioData := makeFakeAudioData(10 * 1024) // 10 MB.

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       io.NopCloser(bytes.NewReader(fakeAudioData)),
		TotalBytes: int64(len(fakeAudioData)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/streamfl?id="+trackIDString).
		Return(fetchTrackResult, nil)

	// Mock lyrics fetch.
	lyricsData := &zvuk.Lyrics{
		Lyrics: "[Verse 1]\nDry-run test lyrics\n",
	}

	mockClient.EXPECT().
		GetTrackLyrics(gomock.Any(), trackIDString).
		Return(lyricsData, nil)

	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        []int64{trackID},
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  albumsMetadata,
		albumsTags:      albumsTags,
		labelsMetadata:  labelsMetadata,
		audioCollection: nil,
	}

	// Set dry-run flag in statistics (normally set by DownloadURLs).
	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	impl.stats.IsDryRun = true

	// Execute dry-run download.
	ctx := context.Background()
	impl.downloadTracks(ctx, metadata)

	// Verify NO files were created (dry-run doesn't save anything).
	audioFiles := findAudioFiles(t, tempDir)
	assert.Empty(t, audioFiles, "No audio files should be created in dry-run mode")

	partFiles := findPartFiles(t, tempDir)
	assert.Empty(t, partFiles, "No .part files should be created in dry-run mode")

	// Verify NO lyrics files were created.
	lyricsFiles, err := filepath.Glob(filepath.Join(tempDir, "**", "*.lrc"))
	require.NoError(t, err)
	assert.Empty(t, lyricsFiles, "No lyrics files should be created in dry-run mode")

	// Verify statistics show correct counts.
	assert.Equal(t, int64(1), impl.stats.TracksDownloaded, "Should count track as 'would download'")
	assert.Equal(t, int64(len(fakeAudioData)), impl.stats.TotalBytesDownloaded, "Should estimate file size")
	assert.Equal(t, int64(1), impl.stats.LyricsDownloaded, "Should count lyrics as 'would download'")
	assert.True(t, impl.stats.IsDryRun, "Statistics should be marked as dry-run")

	// Print summary to verify dry-run output.
	impl.PrintDownloadSummary(ctx)
}

// TestDownloadTracks_DryRunSkipsExistingFiles tests that dry-run respects existing files.
func TestDownloadTracks_DryRunSkipsExistingFiles(t *testing.T) {
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
		DryRun:                 true, // Enable dry-run mode.
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

	// First, download one track to create the actual file.
	// Then run dry-run and verify it's skipped.
	trackID := int64(960)
	trackIDString := "960"
	tracksMetadata := map[string]*zvuk.Track{
		trackIDString: {
			ID:             trackID,
			Title:          "Test Track",
			ReleaseID:      96,
			Position:       1,
			HighestQuality: "flac",
			HasFLAC:        true,
		},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"96": {
			ID:          96,
			Title:       "Test Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20250101,
			LabelID:     999,
			TrackIDs:    []int64{trackID},
		},
	}
	albumsTags := map[string]map[string]string{
		"96": {"albumTitle": "Test Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// First, create the file in NON-dry-run mode.
	normalConfig := &config.Config{
		OutputPath:             tempDir,
		MaxConcurrentDownloads: 1,
		ReplaceTracks:          false,
		Quality:                3,
		DryRun:                 false, // Normal mode.
		ParsedLogLevel:         logger.Level(),
		ParsedMaxDownloadPause: 100 * time.Millisecond,
	}

	normalService := NewService(
		normalConfig,
		mockClient,
		new(mockURLProcessor),
		new(mockTemplateManager),
		new(mockTagProcessor),
	)

	// Setup mocks for actual download.
	streamMetadata := &zvuk.StreamMetadata{Stream: "/streamfl?id=" + trackIDString}

	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), trackIDString, "flac").
		Return(streamMetadata, nil)

	fakeAudioData := []byte("test audio data")

	fetchTrackResult := &zvuk.FetchTrackResult{
		Body:       io.NopCloser(bytes.NewReader(fakeAudioData)),
		TotalBytes: int64(len(fakeAudioData)),
	}

	mockClient.EXPECT().
		FetchTrack(gomock.Any(), "/streamfl?id="+trackIDString).
		Return(fetchTrackResult, nil)

	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        []int64{trackID},
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  albumsMetadata,
		albumsTags:      albumsTags,
		labelsMetadata:  labelsMetadata,
		audioCollection: nil,
	}

	// Download the file in normal mode first.
	ctx := context.Background()
	//nolint:errcheck // downloadTracks does not return an error.
	normalService.(*ServiceImpl).downloadTracks(ctx, metadata)

	// Verify file was created.
	audioFiles := findAudioFiles(t, tempDir)
	require.NotEmpty(t, audioFiles, "File should be created in normal mode")

	// Now run dry-run mode with the file already existing.
	mockClient.EXPECT().
		GetStreamMetadata(gomock.Any(), trackIDString, "flac").
		Return(streamMetadata, nil)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	impl.stats.IsDryRun = true

	// Execute dry-run download.
	impl.downloadTracks(ctx, metadata)

	// Verify statistics show track was skipped.
	assert.Equal(t, int64(1), impl.stats.TracksSkipped, "Should count existing track as skipped in dry-run")
	assert.Equal(t, int64(0), impl.stats.TracksDownloaded, "Should not count existing track as download")
	assert.Equal(t, int64(0), impl.stats.TotalBytesDownloaded, "Should not estimate size for skipped tracks")

	// Verify only one audio file exists (not duplicated).
	audioFilesAfter := findAudioFiles(t, tempDir)
	assert.Len(t, audioFilesAfter, 1, "Should still have only one file after dry-run skip")

	// Print summary.
	impl.PrintDownloadSummary(ctx)
}
