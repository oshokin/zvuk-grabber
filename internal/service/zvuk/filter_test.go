package zvuk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestDownloadTracks_MinQualityFilter tests that tracks below minimum quality are skipped.
func TestDownloadTracks_MinQualityFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		minQuality            uint8
		trackHighestQuality   string
		trackHasFLAC          bool
		expectedSkipped       int64
		expectedDownloaded    int64
		expectedErrorContains string
	}{
		{
			name:                "No filtering (min_quality=0) - download MP3 128",
			minQuality:          0,
			trackHighestQuality: TrackQualityMP3MidString,
			trackHasFLAC:        false,
			expectedSkipped:     0,
			expectedDownloaded:  1,
		},
		{
			name:                  "Min quality MP3 320 - skip MP3 128",
			minQuality:            2,
			trackHighestQuality:   TrackQualityMP3MidString,
			trackHasFLAC:          false,
			expectedSkipped:       1,
			expectedDownloaded:    0,
			expectedErrorContains: "quality below minimum threshold",
		},
		{
			name:                "Min quality MP3 320 - accept MP3 320",
			minQuality:          2,
			trackHighestQuality: TrackQualityMP3HighString,
			trackHasFLAC:        false,
			expectedSkipped:     0,
			expectedDownloaded:  1,
		},
		{
			name:                "Min quality MP3 320 - accept FLAC",
			minQuality:          2,
			trackHighestQuality: TrackQualityFLACString,
			trackHasFLAC:        true,
			expectedSkipped:     0,
			expectedDownloaded:  1,
		},
		{
			name:                  "Min quality FLAC - skip MP3 320",
			minQuality:            3,
			trackHighestQuality:   TrackQualityMP3HighString,
			trackHasFLAC:          false,
			expectedSkipped:       1,
			expectedDownloaded:    0,
			expectedErrorContains: "quality below minimum threshold",
		},
		{
			name:                "Min quality FLAC - accept FLAC",
			minQuality:          3,
			trackHighestQuality: TrackQualityFLACString,
			trackHasFLAC:        true,
			expectedSkipped:     0,
			expectedDownloaded:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			setup := newTestDownloadSetup(t, func(cfg *config.Config) {
				cfg.MinQuality = tc.minQuality
			})
			defer setup.cleanup()

			// Create test metadata with specific quality settings.
			trackID := int64(1000)
			metadata := newTestMetadata([]int64{trackID}, 100).
				withTrackQuality(trackID, tc.trackHighestQuality, tc.trackHasFLAC).
				withAlbumTitle("Quality Test Album").
				build()

			// Setup mock expectations only if track should be downloaded.
			if tc.expectedDownloaded > 0 {
				// Determine stream URL based on quality.
				trackIDString := "1000"
				streamURL := "/stream?id=" + trackIDString

				switch tc.trackHighestQuality {
				case TrackQualityMP3HighString:
					streamURL = "/streamhq?id=" + trackIDString
				case TrackQualityFLACString:
					streamURL = "/streamfl?id=" + trackIDString
				}

				setupMockStreamMetadata(setup.mockClient, trackIDString, tc.trackHighestQuality, streamURL)
				setupMockFetchTrack(setup.mockClient, streamURL, []byte("test audio data"))
			}

			// Execute download.
			ctx := context.Background()
			impl, ok := setup.service.(*ServiceImpl)
			require.True(t, ok, "service must be of type *ServiceImpl")

			impl.downloadTracks(ctx, metadata)

			// Verify statistics.
			assert.Equal(t, tc.expectedSkipped, impl.stats.TracksSkipped,
				"Expected %d tracks skipped", tc.expectedSkipped)
			assert.Equal(t, tc.expectedDownloaded, impl.stats.TracksDownloaded,
				"Expected %d tracks downloaded", tc.expectedDownloaded)

			// Verify error message if track was skipped.
			if tc.expectedSkipped > 0 && tc.expectedErrorContains != "" {
				require.NotEmpty(t, impl.stats.Errors, "Should have recorded an error")
				assert.Contains(t, impl.stats.Errors[0].ErrorMessage, tc.expectedErrorContains,
					"Error message should mention quality threshold")
				assert.Equal(t, "quality check", impl.stats.Errors[0].Phase,
					"Error phase should be 'quality check'")
			}

			// Verify files.
			audioFiles := findAudioFiles(t, setup.tempDir)
			if tc.expectedDownloaded > 0 {
				assert.NotEmpty(t, audioFiles, "Audio file should exist when track is downloaded")
			} else {
				assert.Empty(t, audioFiles, "No audio files should exist when track is skipped")
			}
		})
	}
}

// TestDownloadTracks_MinDurationFilter tests that tracks below minimum duration are skipped.
func TestDownloadTracks_MinDurationFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		minDuration           string
		trackDuration         int64 // Duration in seconds.
		expectedSkipped       int64
		expectedDownloaded    int64
		expectedErrorContains string
	}{
		{
			name:               "No filtering (empty min_duration) - download short track",
			minDuration:        "",
			trackDuration:      15, // 15 seconds.
			expectedSkipped:    0,
			expectedDownloaded: 1,
		},
		{
			name:                  "Min duration 30s - skip 15s track",
			minDuration:           "30s",
			trackDuration:         15, // 15 seconds.
			expectedSkipped:       1,
			expectedDownloaded:    0,
			expectedErrorContains: "duration below minimum threshold",
		},
		{
			name:               "Min duration 30s - accept 30s track (boundary)",
			minDuration:        "30s",
			trackDuration:      30, // 30 seconds.
			expectedSkipped:    0,
			expectedDownloaded: 1,
		},
		{
			name:               "Min duration 30s - accept 60s track",
			minDuration:        "30s",
			trackDuration:      60, // 1 minute.
			expectedSkipped:    0,
			expectedDownloaded: 1,
		},
		{
			name:                  "Min duration 1m - skip 45s track",
			minDuration:           "1m",
			trackDuration:         45, // 45 seconds.
			expectedSkipped:       1,
			expectedDownloaded:    0,
			expectedErrorContains: "duration below minimum threshold",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			setup := newTestDownloadSetup(t, func(cfg *config.Config) {
				cfg.MinDuration = tc.minDuration
				// Parse min duration if set.
				if tc.minDuration != "" {
					var err error

					cfg.ParsedMinDuration, err = time.ParseDuration(tc.minDuration)
					require.NoError(t, err, "Failed to parse min duration")
				}
			})
			defer setup.cleanup()

			// Create test metadata with specific duration.
			trackID := int64(2000)
			metadata := newTestMetadata([]int64{trackID}, 200).
				withTrackDuration(trackID, tc.trackDuration).
				withAlbumTitle("Duration Test Album").
				build()

			// Setup mock expectations only if track should be downloaded.
			if tc.expectedDownloaded > 0 {
				trackIDString := "2000"
				streamURL := "/streamfl?id=" + trackIDString
				setupMockStreamMetadata(setup.mockClient, trackIDString, TrackQualityFLACString, streamURL)
				setupMockFetchTrack(setup.mockClient, streamURL, []byte("test audio data"))
			}

			// Execute download.
			ctx := context.Background()
			impl, ok := setup.service.(*ServiceImpl)
			require.True(t, ok, "service must be of type *ServiceImpl")

			impl.downloadTracks(ctx, metadata)

			// Verify statistics.
			assert.Equal(t, tc.expectedSkipped, impl.stats.TracksSkipped,
				"Expected %d tracks skipped", tc.expectedSkipped)
			assert.Equal(t, tc.expectedDownloaded, impl.stats.TracksDownloaded,
				"Expected %d tracks downloaded", tc.expectedDownloaded)

			// Verify skip reason breakdown.
			if tc.expectedSkipped > 0 {
				assert.Equal(t, tc.expectedSkipped, impl.stats.TracksSkippedDuration,
					"All skipped tracks should be due to duration filter")
			}

			// Verify error message if track was skipped.
			if tc.expectedSkipped > 0 && tc.expectedErrorContains != "" {
				require.NotEmpty(t, impl.stats.Errors, "Should have recorded an error")
				assert.Contains(t, impl.stats.Errors[0].ErrorMessage, tc.expectedErrorContains,
					"Error message should mention duration threshold")
				assert.Equal(t, "duration check", impl.stats.Errors[0].Phase,
					"Error phase should be 'duration check'")
			}

			// Verify files.
			audioFiles := findAudioFiles(t, setup.tempDir)
			if tc.expectedDownloaded > 0 {
				assert.NotEmpty(t, audioFiles, "Audio file should exist when track is downloaded")
			} else {
				assert.Empty(t, audioFiles, "No audio files should exist when track is skipped")
			}
		})
	}
}

// TestDownloadTracks_MaxDurationFilter tests that tracks above maximum duration are skipped.
func TestDownloadTracks_MaxDurationFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		maxDuration           string
		trackDuration         int64 // Duration in seconds.
		expectedSkipped       int64
		expectedDownloaded    int64
		expectedErrorContains string
	}{
		{
			name:               "No filtering (empty max_duration) - download long track",
			maxDuration:        "",
			trackDuration:      900, // 15 minutes.
			expectedSkipped:    0,
			expectedDownloaded: 1,
		},
		{
			name:               "Max duration 10m - accept 5m track",
			maxDuration:        "10m",
			trackDuration:      300, // 5 minutes.
			expectedSkipped:    0,
			expectedDownloaded: 1,
		},
		{
			name:                  "Max duration 10m - skip 15m track",
			maxDuration:           "10m",
			trackDuration:         900, // 15 minutes.
			expectedSkipped:       1,
			expectedDownloaded:    0,
			expectedErrorContains: "duration above maximum threshold",
		},
		{
			name:                  "Max duration 10m - skip 10m 1s track",
			maxDuration:           "10m",
			trackDuration:         601, // 10 minutes 1 second.
			expectedSkipped:       1,
			expectedDownloaded:    0,
			expectedErrorContains: "duration above maximum threshold",
		},
		{
			name:               "Max duration 1h - accept 45m track",
			maxDuration:        "1h",
			trackDuration:      2700, // 45 minutes.
			expectedSkipped:    0,
			expectedDownloaded: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			setup := newTestDownloadSetup(t, func(cfg *config.Config) {
				cfg.MaxDuration = tc.maxDuration
				// Parse max duration if set.
				if tc.maxDuration != "" {
					var err error

					cfg.ParsedMaxDuration, err = time.ParseDuration(tc.maxDuration)
					require.NoError(t, err, "Failed to parse max duration")
				}
			})
			defer setup.cleanup()

			// Create test metadata with specific duration.
			trackID := int64(3000)
			metadata := newTestMetadata([]int64{trackID}, 300).
				withTrackDuration(trackID, tc.trackDuration).
				withAlbumTitle("Duration Test Album").
				build()

			// Setup mock expectations only if track should be downloaded.
			if tc.expectedDownloaded > 0 {
				trackIDString := "3000"
				streamURL := "/streamfl?id=" + trackIDString
				setupMockStreamMetadata(setup.mockClient, trackIDString, TrackQualityFLACString, streamURL)
				setupMockFetchTrack(setup.mockClient, streamURL, []byte("test audio data"))
			}

			// Execute download.
			ctx := context.Background()
			impl, ok := setup.service.(*ServiceImpl)
			require.True(t, ok, "service must be of type *ServiceImpl")

			impl.downloadTracks(ctx, metadata)

			// Verify statistics.
			assert.Equal(t, tc.expectedSkipped, impl.stats.TracksSkipped,
				"Expected %d tracks skipped", tc.expectedSkipped)
			assert.Equal(t, tc.expectedDownloaded, impl.stats.TracksDownloaded,
				"Expected %d tracks downloaded", tc.expectedDownloaded)

			// Verify skip reason breakdown.
			if tc.expectedSkipped > 0 {
				assert.Equal(t, tc.expectedSkipped, impl.stats.TracksSkippedDuration,
					"All skipped tracks should be due to duration filter")
			}

			// Verify error message if track was skipped.
			if tc.expectedSkipped > 0 && tc.expectedErrorContains != "" {
				require.NotEmpty(t, impl.stats.Errors, "Should have recorded an error")
				assert.Contains(t, impl.stats.Errors[0].ErrorMessage, tc.expectedErrorContains,
					"Error message should mention duration threshold")
				assert.Equal(t, "duration check", impl.stats.Errors[0].Phase,
					"Error phase should be 'duration check'")
			}

			// Verify files.
			audioFiles := findAudioFiles(t, setup.tempDir)
			if tc.expectedDownloaded > 0 {
				assert.NotEmpty(t, audioFiles, "Audio file should exist when track is downloaded")
			} else {
				assert.Empty(t, audioFiles, "No audio files should exist when track is skipped")
			}
		})
	}
}
