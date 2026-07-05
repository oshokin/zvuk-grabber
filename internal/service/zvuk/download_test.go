package zvuk

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeAudioData is sample payload used by download concurrency tests.
var fakeAudioData = []byte("fake audio data")

// TestDownloadTracks_Sequential tests that MaxConcurrentDownloads = 1 uses sequential download.
func TestDownloadTracks_Sequential(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(1))
		defer setup.cleanup()

		trackIDs := []int64{101, 102, 103}
		metadata := newTestMetadata(trackIDs, 1).build()

		var (
			executionOrder []int64
			executionMutex sync.Mutex
		)

		for _, trackID := range trackIDs {
			expectSuccessfulTrackDownload(
				setup.mockClient,
				trackID,
				"/stream/"+trackIDText(trackID),
				fakeAudioData,
				func(trackID int64) {
					executionMutex.Lock()
					defer executionMutex.Unlock()

					executionOrder = append(executionOrder, trackID)

					time.Sleep(10 * time.Millisecond) // Simulate API delay.
				},
			)
		}

		setup.impl(t).downloadTracks(context.Background(), metadata)

		assert.Equal(t, trackIDs, executionOrder, "Tracks should be downloaded sequentially")
	})
}

// TestDownloadTracks_Concurrent tests that MaxConcurrentDownloads > 1 downloads tracks concurrently.
func TestDownloadTracks_Concurrent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(3))
		defer setup.cleanup()

		trackIDs := []int64{201, 202, 203, 204, 205}
		metadata := newTestMetadata(trackIDs, 2).withAlbumTitle("Concurrent Album").build()

		var (
			activeDownloads atomic.Int32
			maxObserved     atomic.Int32
		)

		for _, trackID := range trackIDs {
			expectSuccessfulTrackDownload(
				setup.mockClient,
				trackID,
				"/stream/"+trackIDText(trackID),
				fakeAudioData,
				func(_ int64) {
					current := activeDownloads.Add(1)
					updateMaxInt32(&maxObserved, current)
					time.Sleep(50 * time.Millisecond)
					activeDownloads.Add(-1)
				},
			)
		}

		setup.impl(t).downloadTracks(context.Background(), metadata)

		assert.GreaterOrEqual(t, maxObserved.Load(), int32(2),
			"At least 2 tracks should have been downloading concurrently")
		assert.LessOrEqual(t, maxObserved.Load(), int32(3),
			"No more than 3 tracks should download concurrently")
	})
}

// TestDownloadTracks_ConcurrentLimitRespected tests that concurrent download limit is respected.
func TestDownloadTracks_ConcurrentLimitRespected(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		const maxConcurrent int64 = 2

		setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(maxConcurrent))
		defer setup.cleanup()

		trackIDs := []int64{301, 302, 303, 304, 305, 306}
		metadata := newTestMetadata(trackIDs, 3).withAlbumTitle("Limit Test Album").build()

		var (
			activeDownloads atomic.Int32
			maxObserved     atomic.Int32
		)

		for _, trackID := range trackIDs {
			expectSuccessfulTrackDownload(
				setup.mockClient,
				trackID,
				"/stream/"+trackIDText(trackID),
				fakeAudioData,
				func(_ int64) {
					current := activeDownloads.Add(1)
					updateMaxInt32(&maxObserved, current)
					time.Sleep(30 * time.Millisecond)
					activeDownloads.Add(-1)
				},
			)
		}

		setup.impl(t).downloadTracks(context.Background(), metadata)

		assert.LessOrEqual(t, maxObserved.Load(), int32(maxConcurrent),
			"Maximum concurrent downloads should not exceed configured limit")
		assert.GreaterOrEqual(t, maxObserved.Load(), int32(1),
			"At least one download should have occurred")
	})
}

// TestDownloadTracks_ConcurrentWithFewerTracks tests concurrent mode with fewer tracks than workers.
func TestDownloadTracks_ConcurrentWithFewerTracks(t *testing.T) {
	t.Parallel()

	setup := newTestDownloadSetup(t, withMaxConcurrentDownloads(5))
	defer setup.cleanup()

	trackIDs := []int64{401, 402}
	metadata := newTestMetadata(trackIDs, 4).withAlbumTitle("Small Album").build()

	var downloadCount atomic.Int32

	for _, trackID := range trackIDs {
		expectSuccessfulTrackDownload(
			setup.mockClient,
			trackID,
			"/stream/"+trackIDText(trackID),
			fakeAudioData,
			func(_ int64) { downloadCount.Add(1) },
		)
	}

	setup.impl(t).downloadTracks(context.Background(), metadata)

	assert.Equal(t, int32(len(trackIDs)), downloadCount.Load(), "All tracks should have been downloaded")
}
