package zvuk

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDownloadTracks_ProgressBarWithSequential tests that progress bars work with sequential downloads.
func TestDownloadTracks_ProgressBarWithSequential(t *testing.T) {
	t.Parallel()

	setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(1))
	defer setup.cleanup()

	trackIDs := []int64{501}
	metadata := newTestMetadata(trackIDs, 5).withAlbumTitle("Progress Test Album").build()

	expectSuccessfulTrackDownload(
		setup.mockClient,
		trackIDs[0],
		"/stream/501",
		makeFakeAudioData(100),
		nil,
	)

	setup.impl(t).downloadTracks(t.Context(), metadata)

	assert.Equal(t, int64(1), setup.config.MaxConcurrentDownloads,
		"Sequential mode should enable progress bars")
}

// TestDownloadTracks_NoProgressBarWithConcurrent tests that progress bars are disabled in concurrent mode.
func TestDownloadTracks_NoProgressBarWithConcurrent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(2))
		defer setup.cleanup()

		trackIDs := []int64{601, 602}
		metadata := newTestMetadata(trackIDs, 6).withAlbumTitle("Concurrent Progress Test").build()

		for _, trackID := range trackIDs {
			expectSuccessfulTrackDownload(
				setup.mockClient,
				trackID,
				"/streamfl?id="+trackIDText(trackID),
				makeFakeAudioData(50),
				func(_ int64) { time.Sleep(10 * time.Millisecond) },
			)
		}

		setup.impl(t).downloadTracks(t.Context(), metadata)

		assert.Greater(t, setup.config.MaxConcurrentDownloads, int64(1),
			"Concurrent mode disables progress bars to prevent terminal output conflicts")
	})
}
