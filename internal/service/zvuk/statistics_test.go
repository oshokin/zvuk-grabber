package zvuk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
)

func TestDownloadStatistics_InitialState(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")
	assert.NotNil(t, impl.stats, "Statistics should be initialized")
	assert.Equal(t, int64(0), impl.stats.TotalTracksProcessed, "Initial tracks processed should be 0")
	assert.Equal(t, int64(0), impl.stats.TracksDownloaded, "Initial tracks downloaded should be 0")
	assert.Equal(t, int64(0), impl.stats.TracksSkipped, "Initial tracks skipped should be 0")
	assert.Equal(t, int64(0), impl.stats.TracksFailed, "Initial tracks failed should be 0")
}

func TestDownloadStatistics_IncrementTrackDownloaded(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Increment downloaded tracks.
	impl.incrementTrackDownloaded(1024)
	impl.incrementTrackDownloaded(2048)

	assert.Equal(t, int64(2), impl.stats.TotalTracksProcessed, "Should have 2 tracks processed")
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded, "Should have 2 tracks downloaded")
	assert.Equal(t, int64(3072), impl.stats.TotalBytesDownloaded, "Should have 3072 bytes downloaded")
}

func TestDownloadStatistics_IncrementTrackSkipped(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Increment skipped tracks.
	impl.incrementTrackSkipped(SkipReasonExists)
	impl.incrementTrackSkipped(SkipReasonQuality)

	assert.Equal(t, int64(2), impl.stats.TotalTracksProcessed, "Should have 2 tracks processed")
	assert.Equal(t, int64(2), impl.stats.TracksSkipped, "Should have 2 tracks skipped")
	assert.Equal(t, int64(1), impl.stats.TracksSkippedExists, "Should have 1 track skipped (exists)")
	assert.Equal(t, int64(1), impl.stats.TracksSkippedQuality, "Should have 1 track skipped (quality)")
}

func TestDownloadStatistics_IncrementTrackFailed(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Increment failed tracks.
	impl.incrementTrackFailed()

	assert.Equal(t, int64(1), impl.stats.TotalTracksProcessed, "Should have 1 track processed")
	assert.Equal(t, int64(1), impl.stats.TracksFailed, "Should have 1 track failed")
}

func TestDownloadStatistics_MixedResults(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Simulate mixed download results.
	impl.incrementTrackDownloaded(1000)
	impl.incrementTrackDownloaded(2000)
	impl.incrementTrackSkipped(SkipReasonDuration)
	impl.incrementTrackFailed()
	impl.incrementLyricsDownloaded()
	impl.incrementLyricsSkipped()
	impl.incrementCoverDownloaded()
	impl.incrementCoverSkipped()

	assert.Equal(t, int64(4), impl.stats.TotalTracksProcessed, "Should have 4 tracks processed")
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded, "Should have 2 tracks downloaded")
	assert.Equal(t, int64(1), impl.stats.TracksSkipped, "Should have 1 track skipped")
	assert.Equal(t, int64(1), impl.stats.TracksFailed, "Should have 1 track failed")
	assert.Equal(t, int64(3000), impl.stats.TotalBytesDownloaded, "Should have 3000 bytes downloaded")
	assert.Equal(t, int64(1), impl.stats.LyricsDownloaded, "Should have 1 lyrics downloaded")
	assert.Equal(t, int64(1), impl.stats.LyricsSkipped, "Should have 1 lyrics skipped")
	assert.Equal(t, int64(1), impl.stats.CoversDownloaded, "Should have 1 cover downloaded")
	assert.Equal(t, int64(1), impl.stats.CoversSkipped, "Should have 1 cover skipped")
}

func TestPrintDownloadSummary_NoTracksProcessed(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Should not panic when no tracks processed.
	ctx := context.Background()
	impl.PrintDownloadSummary(ctx)

	// Verify no changes to stats.
	assert.Equal(t, int64(0), impl.stats.TotalTracksProcessed, "Should still have 0 tracks processed")
}

func TestPrintDownloadSummary_WithResults(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Simulate some downloads.
	impl.incrementTrackDownloaded(36860019) // ~37 MB (from the example).
	impl.incrementLyricsDownloaded()
	impl.incrementCoverDownloaded()

	// Should not panic when printing summary.
	ctx := context.Background()
	impl.PrintDownloadSummary(ctx)

	// Verify stats are correct.
	assert.Equal(t, int64(1), impl.stats.TotalTracksProcessed, "Should have 1 track processed")
	assert.Equal(t, int64(1), impl.stats.TracksDownloaded, "Should have 1 track downloaded")
	assert.Equal(t, int64(36860019), impl.stats.TotalBytesDownloaded, "Should have correct bytes")
}

func TestDownloadStatistics_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	service := NewService(
		&config.Config{
			MaxConcurrentDownloads: 5,
		},
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Simulate concurrent downloads.
	done := make(chan bool)

	for range 10 {
		go func() {
			impl.incrementTrackDownloaded(1000)
			impl.incrementLyricsDownloaded()
			impl.incrementCoverDownloaded()

			done <- true
		}()
	}

	// Wait for all goroutines to finish.
	for range 10 {
		<-done
	}

	// Verify all increments were recorded.
	assert.Equal(t, int64(10), impl.stats.TotalTracksProcessed, "Should have 10 tracks processed")
	assert.Equal(t, int64(10), impl.stats.TracksDownloaded, "Should have 10 tracks downloaded")
	assert.Equal(t, int64(10000), impl.stats.TotalBytesDownloaded, "Should have 10000 bytes downloaded")
	assert.Equal(t, int64(10), impl.stats.LyricsDownloaded, "Should have 10 lyrics downloaded")
	assert.Equal(t, int64(10), impl.stats.CoversDownloaded, "Should have 10 covers downloaded")
}

func TestPrintDownloadSummary_WithInterruption(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Simulate partial download before interruption.
	impl.incrementTrackDownloaded(10000000) // 10 MB.
	impl.incrementTrackDownloaded(5000000)  // 5 MB.
	impl.incrementCoverDownloaded()

	// Create a canceled context to simulate CTRL+C.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel to simulate interruption.

	// Should not panic when printing summary with interrupted context.
	impl.PrintDownloadSummary(ctx)

	// Verify stats are correct.
	assert.Equal(t, int64(2), impl.stats.TotalTracksProcessed, "Should have 2 tracks processed")
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded, "Should have 2 tracks downloaded")
	assert.Equal(t, int64(15000000), impl.stats.TotalBytesDownloaded, "Should have 15 MB downloaded")
}

func TestDownloadStatistics_ErrorTracking(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Simulate various errors during download.
	impl.recordError(&ErrorContext{
		Category:       DownloadCategoryTrack,
		ItemID:         "12345",
		ItemTitle:      "Test Track 1",
		Phase:          "downloading file",
		ParentCategory: DownloadCategoryAlbum,
		ParentID:       "99999",
		ParentTitle:    "Parent Album",
	}, assert.AnError)

	impl.recordError(&ErrorContext{
		Category:  DownloadCategoryAlbum,
		ItemID:    "67890",
		ItemTitle: "Test Album",
		ItemURL:   "https://zvuk.com/release/67890",
		Phase:     "fetching album data",
	}, assert.AnError)

	impl.recordError(&ErrorContext{
		Category:  DownloadCategoryPlaylist,
		ItemID:    "11111",
		ItemTitle: "My Playlist",
		ItemURL:   "https://zvuk.com/playlist/11111",
		Phase:     "fetching playlist metadata",
	}, assert.AnError)

	impl.incrementTrackFailed()
	impl.incrementTrackDownloaded(1000)

	// Verify errors were recorded.
	assert.Len(t, impl.stats.Errors, 3, "Should have 3 errors recorded")
	assert.Equal(t, "12345", impl.stats.Errors[0].ItemID)
	assert.Equal(t, "Test Track 1", impl.stats.Errors[0].ItemTitle)
	assert.Equal(t, "downloading file", impl.stats.Errors[0].Phase)
	assert.Equal(t, DownloadCategoryTrack, impl.stats.Errors[0].Category)

	// Print summary with errors (should not panic).
	ctx := context.Background()
	impl.PrintDownloadSummary(ctx)
}

// Example of what the stats might look like for a real download.
func ExampleServiceImpl_PrintDownloadSummary() {
	service := NewService(
		new(config.Config),
		new(zvuk.ClientImpl),
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	if !ok {
		panic("failed to cast to ServiceImpl")
	}

	// Simulate a typical download session.
	impl.incrementTrackDownloaded(36860019)
	impl.incrementCoverDownloaded()

	ctx := context.Background()
	impl.PrintDownloadSummary(ctx)
}

// TestPrintDownloadSummary_WithDuration tests that duration and speed are displayed correctly.
func TestPrintDownloadSummary_WithDuration(t *testing.T) {
	t.Parallel()

	service := NewService(
		new(config.Config),
		nil,
		nil,
		nil,
		nil,
	)

	impl, ok := service.(*ServiceImpl)
	assert.True(t, ok, "Service should be of type *ServiceImpl")

	// Record actual start time.
	impl.stats.StartTime = time.Now()

	// Simulate some download work with controlled timing.
	totalBytes := int64(100 * 1024 * 1024)
	impl.incrementTrackDownloaded(totalBytes)

	// Sleep to ensure measurable duration (at least 100ms for test reliability).
	time.Sleep(150 * time.Millisecond)

	impl.incrementTrackDownloaded(totalBytes)

	// Record actual end time.
	impl.stats.EndTime = time.Now()

	// Calculate actual duration.
	actualDuration := impl.stats.EndTime.Sub(impl.stats.StartTime)

	// Verify stats.
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded)
	assert.Equal(t, totalBytes*2, impl.stats.TotalBytesDownloaded)

	// Print summary (should show duration and average speed).
	ctx := context.Background()
	impl.PrintDownloadSummary(ctx)

	// Verify duration is at least what we slept for.
	assert.GreaterOrEqual(t, actualDuration, 150*time.Millisecond,
		"Duration should be at least the sleep time")

	// Verify average speed calculation is reasonable.
	if actualDuration > 0 {
		expectedSpeed := float64(totalBytes*2) / actualDuration.Seconds()
		assert.Greater(t, expectedSpeed, float64(0), "Average speed should be positive")
		// Speed should be huge since we downloaded 200MB in ~150ms.
		assert.Greater(t, expectedSpeed, float64(1024*1024), "Speed should be > 1 MB/s in test")
	}
}

// TestFormatDuration tests the formatDuration helper function.
func TestFormatDuration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "exactly 1 minute",
			duration: 1 * time.Minute,
			expected: "1m 0s",
		},
		{
			name:     "hours, minutes, and seconds",
			duration: 1*time.Hour + 15*time.Minute + 30*time.Second,
			expected: "1h 15m 30s",
		},
		{
			name:     "multiple hours",
			duration: 3*time.Hour + 45*time.Minute + 12*time.Second,
			expected: "3h 45m 12s",
		},
		{
			name:     "exactly 1 hour",
			duration: 1 * time.Hour,
			expected: "1h 0m 0s",
		},
		{
			name:     "very short duration",
			duration: 1 * time.Millisecond,
			expected: "1ms",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := formatDuration(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}
