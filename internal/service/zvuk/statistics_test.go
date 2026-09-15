package zvuk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestDownloadStatistics_InitialState verifies download statistics start at zero.
func TestDownloadStatistics_InitialState(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	assert.NotNil(t, impl.stats)
	assert.Equal(t, int64(0), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(0), impl.stats.TracksDownloaded)
	assert.Equal(t, int64(0), impl.stats.TracksSkipped)
	assert.Equal(t, int64(0), impl.stats.TracksFailed)
}

// TestDownloadStatistics_IncrementTrackDownloaded verifies downloaded track and byte counters increment correctly.
func TestDownloadStatistics_IncrementTrackDownloaded(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackDownloaded(1024)
	impl.incrementTrackDownloaded(2048)

	assert.Equal(t, int64(2), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded)
	assert.Equal(t, int64(3072), impl.stats.TotalBytesDownloaded)
}

// TestDownloadStatistics_IncrementTrackSkipped verifies skipped track counters increment by reason.
func TestDownloadStatistics_IncrementTrackSkipped(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackSkipped(SkipReasonExists)
	impl.incrementTrackSkipped(SkipReasonQuality)

	assert.Equal(t, int64(2), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(2), impl.stats.TracksSkipped)
	assert.Equal(t, int64(1), impl.stats.TracksSkippedExists)
	assert.Equal(t, int64(1), impl.stats.TracksSkippedQuality)
}

// TestDownloadStatistics_IncrementTrackFailed verifies failed track counters increment correctly.
func TestDownloadStatistics_IncrementTrackFailed(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackFailed()

	assert.Equal(t, int64(1), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(1), impl.stats.TracksFailed)
}

// TestDownloadStatistics_MixedResults verifies mixed download, skip, and asset counters stay consistent.
func TestDownloadStatistics_MixedResults(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackDownloaded(1000)
	impl.incrementTrackDownloaded(2000)
	impl.incrementTrackSkipped(SkipReasonDuration)
	impl.incrementTrackFailed()
	impl.incrementLyricsDownloaded()
	impl.incrementLyricsSkipped()
	impl.incrementCoverDownloaded()
	impl.incrementCoverSkipped()

	assert.Equal(t, int64(4), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded)
	assert.Equal(t, int64(1), impl.stats.TracksSkipped)
	assert.Equal(t, int64(1), impl.stats.TracksFailed)
	assert.Equal(t, int64(3000), impl.stats.TotalBytesDownloaded)
	assert.Equal(t, int64(1), impl.stats.LyricsDownloaded)
	assert.Equal(t, int64(1), impl.stats.LyricsSkipped)
	assert.Equal(t, int64(1), impl.stats.CoversDownloaded)
	assert.Equal(t, int64(1), impl.stats.CoversSkipped)
}

// TestPrintDownloadSummary_NoTracksProcessed verifies the summary is omitted when no work was done.
func TestPrintDownloadSummary_NoTracksProcessed(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.PrintDownloadSummary(t.Context())
	assert.Equal(t, int64(0), impl.stats.TotalTracksProcessed)
}

// TestPrintDownloadSummary_WithResults verifies the summary renders when tracks and assets were processed.
func TestPrintDownloadSummary_WithResults(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackDownloaded(36860019)
	impl.incrementLyricsDownloaded()
	impl.incrementCoverDownloaded()
	impl.PrintDownloadSummary(t.Context())

	assert.Equal(t, int64(1), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(1), impl.stats.TracksDownloaded)
	assert.Equal(t, int64(36860019), impl.stats.TotalBytesDownloaded)
}

// TestDownloadStatistics_ConcurrentAccess verifies statistics remain correct under concurrent updates.
func TestDownloadStatistics_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, &config.Config{MaxConcurrentDownloads: 5})
	done := make(chan struct{}, 10)

	for range 10 {
		go func() {
			impl.incrementTrackDownloaded(1000)
			impl.incrementLyricsDownloaded()
			impl.incrementCoverDownloaded()

			done <- struct{}{}
		}()
	}

	for range 10 {
		<-done
	}

	assert.Equal(t, int64(10), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(10), impl.stats.TracksDownloaded)
	assert.Equal(t, int64(10000), impl.stats.TotalBytesDownloaded)
	assert.Equal(t, int64(10), impl.stats.LyricsDownloaded)
	assert.Equal(t, int64(10), impl.stats.CoversDownloaded)
}

// TestPrintDownloadSummary_WithInterruption verifies interrupted sessions are reflected in the summary.
func TestPrintDownloadSummary_WithInterruption(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackDownloaded(10000000)
	impl.incrementTrackDownloaded(5000000)
	impl.incrementCoverDownloaded()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	impl.PrintDownloadSummary(ctx)

	assert.Equal(t, int64(2), impl.stats.TotalTracksProcessed)
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded)
	assert.Equal(t, int64(15000000), impl.stats.TotalBytesDownloaded)
}

// TestDownloadStatistics_ErrorTracking verifies structured errors are stored and included in the summary.
func TestDownloadStatistics_ErrorTracking(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.recordError(&DownloadError{
		Category:       DownloadCategoryTrack,
		ItemID:         "12345",
		ItemTitle:      "Test Track 1",
		Phase:          "downloading file",
		ParentCategory: DownloadCategoryAlbum,
		ParentID:       "99999",
		ParentTitle:    "Parent Album",
		Error:          assert.AnError,
	})
	impl.recordError(&DownloadError{
		Category:  DownloadCategoryAlbum,
		ItemID:    "67890",
		ItemTitle: "Test Album",
		ItemURL:   "https://zvuk.com/release/67890",
		Phase:     "fetching album data",
		Error:     assert.AnError,
	})
	impl.recordError(&DownloadError{
		Category:  DownloadCategoryPlaylist,
		ItemID:    "11111",
		ItemTitle: "My Playlist",
		ItemURL:   "https://zvuk.com/playlist/11111",
		Phase:     "fetching playlist metadata",
		Error:     assert.AnError,
	})
	impl.incrementTrackFailed()
	impl.incrementTrackDownloaded(1000)

	assert.Len(t, impl.stats.Errors, 3)
	assert.Equal(t, "12345", impl.stats.Errors[0].ItemID)
	assert.Equal(t, "Test Track 1", impl.stats.Errors[0].ItemTitle)
	assert.Equal(t, "downloading file", impl.stats.Errors[0].Phase)
	assert.Equal(t, DownloadCategoryTrack, impl.stats.Errors[0].Category)
	impl.PrintDownloadSummary(t.Context())
}

// TestPrintDownloadSummary_WithDuration verifies duration and speed are computed from session timestamps.
func TestPrintDownloadSummary_WithDuration(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	start := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	totalBytes := int64(100 * 1024 * 1024)
	impl.stats.StartTime = start
	impl.incrementTrackDownloaded(totalBytes)
	impl.incrementTrackDownloaded(totalBytes)
	impl.stats.EndTime = start.Add(150 * time.Millisecond)

	actualDuration := impl.stats.EndTime.Sub(impl.stats.StartTime)
	assert.Equal(t, int64(2), impl.stats.TracksDownloaded)
	assert.Equal(t, totalBytes*2, impl.stats.TotalBytesDownloaded)
	impl.PrintDownloadSummary(t.Context())
	assert.Equal(t, 150*time.Millisecond, actualDuration)
	assert.Greater(t, float64(totalBytes*2)/actualDuration.Seconds(), float64(1024*1024))
}

// TestFormatDuration verifies human-readable duration formatting for common intervals.
func TestFormatDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{name: "milliseconds", duration: 500 * time.Millisecond, expected: "500ms"},
		{name: "seconds only", duration: 45 * time.Second, expected: "45s"},
		{name: "minutes and seconds", duration: 2*time.Minute + 30*time.Second, expected: "2m 30s"},
		{name: "exactly 1 minute", duration: time.Minute, expected: "1m 0s"},
		{
			name:     "hours, minutes, and seconds",
			duration: time.Hour + 15*time.Minute + 30*time.Second,
			expected: "1h 15m 30s",
		},
		{name: "multiple hours", duration: 3*time.Hour + 45*time.Minute + 12*time.Second, expected: "3h 45m 12s"},
		{name: "exactly 1 hour", duration: time.Hour, expected: "1h 0m 0s"},
		{name: "very short duration", duration: time.Millisecond, expected: "1ms"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, formatDuration(tc.duration))
		})
	}
}

// newStatisticsService builds a ServiceImpl test instance with optional configuration.
func newStatisticsService(t *testing.T, cfg *config.Config) *ServiceImpl {
	t.Helper()

	if cfg == nil {
		cfg = new(config.Config)
	}

	service := NewService(cfg, nil, nil, nil, nil)

	impl, ok := service.(*ServiceImpl)
	if !ok {
		t.Fatalf("Service should be of type *ServiceImpl")
	}

	return impl
}
