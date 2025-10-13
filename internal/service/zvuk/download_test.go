package zvuk

import (
	"context"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	mock_zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk/mocks"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TestDownloadTracks_Sequential tests that MaxConcurrentDownloads = 1 uses sequential download.
func TestDownloadTracks_Sequential(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	testConfig := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 1, // Sequential mode.
		ReplaceTracks:          false,
		Quality:                3,
		ParsedLogLevel:         logger.Level(),
		ParsedMaxDownloadPause: 100 * time.Millisecond,
	}

	// Create service instance.
	service := NewService(
		testConfig,
		mockClient,
		new(mockURLProcessor),
		new(mockTemplateManager),
		new(mockTagProcessor),
	)

	// Create test metadata with 3 tracks.
	trackIDs := []int64{101, 102, 103}
	tracksMetadata := map[string]*zvuk.Track{
		"101": {ID: 101, Title: "Track 1", ReleaseID: 1, Position: 1, HighestQuality: "flac", HasFLAC: true},
		"102": {ID: 102, Title: "Track 2", ReleaseID: 1, Position: 2, HighestQuality: "flac", HasFLAC: true},
		"103": {ID: 103, Title: "Track 3", ReleaseID: 1, Position: 3, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"1": {
			ID:          1,
			Title:       "Test Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"1": {"albumTitle": "Test Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Track the order of API calls to verify sequential execution.
	var (
		executionOrder []int64
		executionMutex sync.Mutex
	)

	// Setup mock expectations for each track.
	fakeAudioContent := "fake audio data"

	for _, trackID := range trackIDs {
		trackIDString := "101"

		switch trackID {
		case 102:
			trackIDString = "102"
		case 103:
			trackIDString = "103"
		}

		// Prepare stream metadata response before using in mock.
		streamURL := "/stream/" + trackIDString
		streamMetadata := &zvuk.StreamMetadata{Stream: streamURL}

		mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), trackIDString, TrackQualityFLACString).
			DoAndReturn(func(_ context.Context, _ string, _ string) (*zvuk.StreamMetadata, error) {
				executionMutex.Lock()

				executionOrder = append(executionOrder, trackID)

				executionMutex.Unlock()
				time.Sleep(10 * time.Millisecond) // Simulate API delay.

				return streamMetadata, nil
			})

		// Prepare fetch result before using in mock.
		fetchTrackResult := &zvuk.FetchTrackResult{
			Body:       io.NopCloser(strings.NewReader(fakeAudioContent)),
			TotalBytes: 100,
		}

		mockClient.EXPECT().
			FetchTrack(gomock.Any(), streamURL).
			Return(fetchTrackResult, nil)
	}

	// Create download metadata.
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

	// Verify sequential execution (tracks downloaded in order).
	assert.Equal(t, []int64{101, 102, 103}, executionOrder, "Tracks should be downloaded sequentially")
}

// TestDownloadTracks_Concurrent tests that MaxConcurrentDownloads > 1 downloads tracks concurrently.
func TestDownloadTracks_Concurrent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	testConfig := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 3, // Concurrent mode.
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

	// Create test metadata with 5 tracks.
	trackIDs := []int64{201, 202, 203, 204, 205}
	tracksMetadata := map[string]*zvuk.Track{
		"201": {ID: 201, Title: "Track 1", ReleaseID: 2, Position: 1, HighestQuality: "flac", HasFLAC: true},
		"202": {ID: 202, Title: "Track 2", ReleaseID: 2, Position: 2, HighestQuality: "flac", HasFLAC: true},
		"203": {ID: 203, Title: "Track 3", ReleaseID: 2, Position: 3, HighestQuality: "flac", HasFLAC: true},
		"204": {ID: 204, Title: "Track 4", ReleaseID: 2, Position: 4, HighestQuality: "flac", HasFLAC: true},
		"205": {ID: 205, Title: "Track 5", ReleaseID: 2, Position: 5, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"2": {
			ID:          2,
			Title:       "Concurrent Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"2": {"albumTitle": "Concurrent Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Track concurrent execution metrics.
	var (
		activeConcurrentCount int32
		maxConcurrent         int32
		concurrentMutex       sync.Mutex
	)

	// Setup mock expectations for each track.
	fakeAudioContent := "fake audio data"

	for _, trackID := range trackIDs {
		trackIDString := ""

		switch trackID {
		case 201:
			trackIDString = "201"
		case 202:
			trackIDString = "202"
		case 203:
			trackIDString = "203"
		case 204:
			trackIDString = "204"
		case 205:
			trackIDString = "205"
		}

		// Prepare stream metadata response before using in mock.
		streamURL := "/stream/" + trackIDString
		streamMetadata := &zvuk.StreamMetadata{Stream: streamURL}

		mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), trackIDString, TrackQualityFLACString).
			DoAndReturn(func(_ context.Context, _ string, _ string) (*zvuk.StreamMetadata, error) {
				// Increment active count.
				current := atomic.AddInt32(&activeConcurrentCount, 1)

				// Track maximum concurrent downloads.
				concurrentMutex.Lock()

				if current > maxConcurrent {
					maxConcurrent = current
				}

				concurrentMutex.Unlock()

				// Simulate API delay.
				time.Sleep(50 * time.Millisecond)

				// Decrement active count.
				atomic.AddInt32(&activeConcurrentCount, -1)

				return streamMetadata, nil
			})

		// Prepare fetch result before using in mock.
		fetchTrackResult := &zvuk.FetchTrackResult{
			Body:       io.NopCloser(strings.NewReader(fakeAudioContent)),
			TotalBytes: 100,
		}

		mockClient.EXPECT().
			FetchTrack(gomock.Any(), streamURL).
			Return(fetchTrackResult, nil)
	}

	// Create download metadata.
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

	// Verify concurrent execution (at least 2 tracks were downloading simultaneously).
	assert.GreaterOrEqual(t, maxConcurrent, int32(2),
		"At least 2 tracks should have been downloading concurrently")
	assert.LessOrEqual(t, maxConcurrent, int32(3),
		"No more than 3 tracks should download concurrently (MaxConcurrentDownloads=3)")
}

// TestDownloadTracks_ConcurrentLimitRespected tests that concurrent download limit is respected.
func TestDownloadTracks_ConcurrentLimitRespected(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	maxConcurrent := int64(2)
	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	testConfig := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: maxConcurrent,
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

	// Create test metadata with 6 tracks.
	trackIDs := []int64{301, 302, 303, 304, 305, 306}
	tracksMetadata := make(map[string]*zvuk.Track)

	for i, tid := range trackIDs {
		tidStr := ""

		switch tid {
		case 301:
			tidStr = "301"
		case 302:
			tidStr = "302"
		case 303:
			tidStr = "303"
		case 304:
			tidStr = "304"
		case 305:
			tidStr = "305"
		case 306:
			tidStr = "306"
		}

		tracksMetadata[tidStr] = &zvuk.Track{
			ID:             tid,
			Title:          "Track " + tidStr,
			ReleaseID:      3,
			Position:       int64(i + 1),
			HighestQuality: "flac",
			HasFLAC:        true,
		}
	}

	albumsMetadata := map[string]*zvuk.Release{
		"3": {
			ID:          3,
			Title:       "Limit Test Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"3": {"albumTitle": "Limit Test Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	// Track maximum concurrent downloads.
	var (
		activeConcurrentCount int32
		maxConcurrentObserved int32
	)

	// Setup mock expectations.
	fakeAudioContent := "fake audio data"

	for _, trackID := range trackIDs {
		trackIDString := ""

		switch trackID {
		case 301:
			trackIDString = "301"
		case 302:
			trackIDString = "302"
		case 303:
			trackIDString = "303"
		case 304:
			trackIDString = "304"
		case 305:
			trackIDString = "305"
		case 306:
			trackIDString = "306"
		}

		// Prepare stream metadata response before using in mock.
		streamURL := "/stream/" + trackIDString
		streamMetadata := &zvuk.StreamMetadata{Stream: streamURL}

		mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), trackIDString, TrackQualityFLACString).
			DoAndReturn(func(_ context.Context, _ string, _ string) (*zvuk.StreamMetadata, error) {
				current := atomic.AddInt32(&activeConcurrentCount, 1)

				// Track maximum.
				for {
					currentMax := atomic.LoadInt32(&maxConcurrentObserved)
					if current <= currentMax ||
						atomic.CompareAndSwapInt32(&maxConcurrentObserved, currentMax, current) {
						break
					}
				}

				// Hold for a bit to ensure overlapping execution.
				time.Sleep(30 * time.Millisecond)

				atomic.AddInt32(&activeConcurrentCount, -1)

				return streamMetadata, nil
			})

		// Prepare fetch result before using in mock.
		fetchTrackResult := &zvuk.FetchTrackResult{
			Body:       io.NopCloser(strings.NewReader(fakeAudioContent)),
			TotalBytes: 100,
		}

		mockClient.EXPECT().
			FetchTrack(gomock.Any(), streamURL).
			Return(fetchTrackResult, nil)
	}

	// Create download metadata.
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

	// Verify the concurrent limit was respected.
	assert.LessOrEqual(t, maxConcurrentObserved, int32(maxConcurrent),
		"Maximum concurrent downloads should not exceed configured limit")
	assert.GreaterOrEqual(t, maxConcurrentObserved, int32(1),
		"At least one download should have occurred")
}

// TestDownloadTracks_ConcurrentWithFewerTracks tests concurrent mode with fewer tracks than workers.
func TestDownloadTracks_ConcurrentWithFewerTracks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_zvuk_client.NewMockClient(ctrl)
	testConfig := &config.Config{
		OutputPath:             t.TempDir(),
		MaxConcurrentDownloads: 5, // More workers than tracks.
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

	// Create test metadata with only 2 tracks.
	trackIDs := []int64{401, 402}
	tracksMetadata := map[string]*zvuk.Track{
		"401": {ID: 401, Title: "Track 1", ReleaseID: 4, Position: 1, HighestQuality: "flac", HasFLAC: true},
		"402": {ID: 402, Title: "Track 2", ReleaseID: 4, Position: 2, HighestQuality: "flac", HasFLAC: true},
	}
	albumsMetadata := map[string]*zvuk.Release{
		"4": {
			ID:          4,
			Title:       "Small Album",
			ArtistNames: []string{"Test Artist"},
			Date:        20240101,
			LabelID:     999,
			TrackIDs:    trackIDs,
		},
	}
	albumsTags := map[string]map[string]string{
		"4": {"albumTitle": "Small Album"},
	}
	labelsMetadata := map[string]*zvuk.Label{
		"999": {Title: "Test Label"},
	}

	var downloadCount int32

	// Setup mock expectations.
	fakeAudioContent := "fake audio data"

	for _, trackID := range trackIDs {
		trackIDString := "401"
		if trackID == 402 {
			trackIDString = "402"
		}

		// Prepare stream metadata response before using in mock.
		streamURL := "/stream/" + trackIDString
		streamMetadata := &zvuk.StreamMetadata{Stream: streamURL}

		mockClient.EXPECT().
			GetStreamMetadata(gomock.Any(), trackIDString, TrackQualityFLACString).
			DoAndReturn(func(_ context.Context, _ string, _ string) (*zvuk.StreamMetadata, error) {
				atomic.AddInt32(&downloadCount, 1)

				return streamMetadata, nil
			})

		// Prepare fetch result before using in mock.
		fetchTrackResult := &zvuk.FetchTrackResult{
			Body:       io.NopCloser(strings.NewReader(fakeAudioContent)),
			TotalBytes: 100,
		}

		mockClient.EXPECT().
			FetchTrack(gomock.Any(), streamURL).
			Return(fetchTrackResult, nil)
	}

	// Create download metadata.
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

	// Verify all tracks were downloaded.
	assert.Equal(t, int32(2), downloadCount, "All 2 tracks should have been downloaded")
}
