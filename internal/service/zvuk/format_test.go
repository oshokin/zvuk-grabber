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
	"github.com/oshokin/zvuk-grabber/internal/constants"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TestDownloadTracks_AllFormatsWithCoverEmbedding tests that FLAC, MP3, and fallback formats work correctly.
func TestDownloadTracks_AllFormatsWithCoverEmbedding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		quality           TrackQuality
		expectedExtension string
		highestQuality    string
		streamURL         string
	}{
		{
			name:              "FLAC format",
			quality:           TrackQualityFLAC,
			expectedExtension: constants.ExtensionFLAC,
			highestQuality:    "flac",
			streamURL:         "/streamfl?id=1001",
		},
		{
			name:              "MP3 High format",
			quality:           TrackQualityMP3High,
			expectedExtension: constants.ExtensionMP3,
			highestQuality:    "high",
			streamURL:         "/streamhq?id=1001",
		},
		{
			name:              "MP3 Mid format",
			quality:           TrackQualityMP3Mid,
			expectedExtension: constants.ExtensionMP3,
			highestQuality:    "mid",
			streamURL:         "/stream?id=1001",
		},
		{
			name:              "Unknown format (fallback to .bin)",
			quality:           TrackQualityUnknown,
			expectedExtension: constants.ExtensionBin,
			highestQuality:    "",
			streamURL:         "/unknown-stream/1001",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock_zvuk_client.NewMockClient(ctrl)
			tempDir := t.TempDir()

			testConfig := &config.Config{
				OutputPath:             tempDir,
				MaxConcurrentDownloads: 1,
				ReplaceTracks:          false,
				Quality:                uint8(tc.quality),
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

			// Create test metadata using realistic values based on actual API responses.
			trackID := int64(160456567)
			trackIDString := "160456567"
			releaseID := int64(42393651)
			releaseIDString := "42393651"
			labelID := int64(3493777)
			labelIDString := "3493777"

			tracksMetadata := map[string]*zvuk.Track{
				trackIDString: {
					ID:             trackID,
					Title:          "Tipping Point",
					ReleaseID:      releaseID,
					Position:       1,
					HighestQuality: tc.highestQuality,
					HasFLAC:        tc.quality == TrackQualityFLAC,
				},
			}
			albumsMetadata := map[string]*zvuk.Release{
				releaseIDString: {
					ID:          releaseID,
					Title:       "Tipping Point",
					ArtistNames: []string{"Megadeth"},
					Date:        20251003,
					LabelID:     labelID,
					TrackIDs:    []int64{trackID},
				},
			}
			albumsTags := map[string]map[string]string{
				releaseIDString: {
					"albumTitle":  "Tipping Point",
					"albumArtist": "Megadeth",
					"releaseYear": "2025",
					"label":       "BLKIIBLK",
				},
			}
			labelsMetadata := map[string]*zvuk.Label{
				labelIDString: {Title: "BLKIIBLK"},
			}

			// Setup mock expectations.
			streamMetadata := &zvuk.StreamMetadata{Stream: tc.streamURL}

			mockClient.EXPECT().
				GetStreamMetadata(gomock.Any(), trackIDString, gomock.Any()).
				Return(streamMetadata, nil)

			// Create realistic audio data for the format based on actual track sizes.
			// Real FLAC: ~37 MB for 4:28 track, MP3 320: ~10 MB, MP3 128: ~4 MB.
			var fakeAudioData []byte

			switch tc.quality {
			case TrackQualityFLAC:
				// Simulate FLAC file (~37 MB like the real Tipping Point track).
				fakeAudioData = make([]byte, 37*1024*1024)
			case TrackQualityMP3High:
				// Simulate MP3 320 Kbps file.
				fakeAudioData = make([]byte, 10*1024*1024)
			case TrackQualityMP3Mid:
				// Simulate MP3 128 Kbps file.
				fakeAudioData = make([]byte, 4*1024*1024)
			default:
				// Unknown format.
				fakeAudioData = []byte("unknown format binary data")
			}

			// Fill with pseudo-random but deterministic data.
			for i := range fakeAudioData {
				fakeAudioData[i] = byte(i % 256)
			}

			fetchTrackResult := &zvuk.FetchTrackResult{
				Body:       io.NopCloser(bytes.NewReader(fakeAudioData)),
				TotalBytes: int64(len(fakeAudioData)),
			}

			mockClient.EXPECT().
				FetchTrack(gomock.Any(), tc.streamURL).
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

			// Execute download.
			ctx := context.Background()

			impl, ok := service.(*ServiceImpl)
			assert.True(t, ok, "Service should be of type *ServiceImpl")

			impl.downloadTracks(ctx, metadata)

			// Verify that the file was created with the correct extension and content.
			_, foundFile := findFileWithExtension(t, tempDir, tc.expectedExtension, fakeAudioData)
			assert.True(t, foundFile, "File with extension %s should exist", tc.expectedExtension)
		})
	}
}
