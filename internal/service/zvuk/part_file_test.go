package zvuk

import (
	"bytes"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// TestDownloadTracks_PartFileHandling tests that .part files are used for atomic downloads.
func TestDownloadTracks_PartFileHandling(t *testing.T) {
	t.Parallel()

	setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(1))
	defer setup.cleanup()

	trackID := int64(701)
	audioData := []byte("complete audio file content")
	metadata := newTestMetadata([]int64{trackID}, 7).withAlbumTitle("Part File Test Album").build()

	expectSuccessfulTrackDownload(setup.mockClient, trackID, "/stream/701", audioData, nil)

	setup.impl(t).downloadTracks(t.Context(), metadata)

	assert.Empty(t, findPartFiles(t, setup.tempDir), ".part files should be cleaned up after successful download")

	audioFiles := findAudioFiles(t, setup.tempDir)
	assert.NotEmpty(t, audioFiles, "Final track file should exist after download")

	if len(audioFiles) > 0 {
		content, err := os.ReadFile(audioFiles[0])
		require.NoError(t, err, "Failed to read downloaded file")
		assert.Equal(t, audioData, content, "Downloaded file content should match source data")
	}
}

// TestDownloadTracks_PartFileCleanupOnFailure tests that .part files are cleaned up when download fails.
func TestDownloadTracks_PartFileCleanupOnFailure(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(1))
		defer setup.cleanup()

		trackID := int64(801)
		metadata := newTestMetadata([]int64{trackID}, 8).withAlbumTitle("Failed Download Album").build()

		setup.mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), "801", TrackQualityFLACString).
			Return(&zvuk.StreamMetadata{Stream: "/stream/801"}, nil)

		fullContent := []byte("this is supposed to be 100 bytes of audio data but network failed")
		partialReader := &partialReadCloser{Reader: bytes.NewReader(fullContent[:len(fullContent)/2])}

		setup.mockClient.EXPECT().
			FetchTrack(gomock.Any(), "/stream/801").
			Return(&zvuk.FetchTrackResult{
				Body:       partialReader,
				TotalBytes: int64(len(fullContent)),
			}, nil)

		setup.impl(t).downloadTracks(t.Context(), metadata)

		// Allow deferred cleanup paths to complete before assertions.
		time.Sleep(50 * time.Millisecond)

		assert.Empty(t, findPartFiles(t, setup.tempDir), ".part files should be cleaned up after failed download")
		assert.Empty(t, findAudioFiles(t, setup.tempDir), "No audio files should exist after failed download")
	})
}
