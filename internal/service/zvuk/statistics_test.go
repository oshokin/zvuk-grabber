package zvuk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/service/stats"
)

// TestDownloadStatistics_InitialState verifies download statistics start at zero.
func TestDownloadStatistics_InitialState(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	want := new(DownloadStatistics)
	assert.Equal(t, want, impl.stats)
}

// TestDownloadStatistics_Clone detaches error slices from the live counters.
func TestDownloadStatistics_Clone(t *testing.T) {
	t.Parallel()

	original := &DownloadStatistics{
		TracksDownloaded: 2,
		Errors:           []*DownloadError{{ItemID: "1"}},
	}
	cloned := original.Clone()
	assert.Equal(t, original.TracksDownloaded, cloned.TracksDownloaded)

	cloned.TracksDownloaded = 9
	cloned.Errors[0] = &DownloadError{ItemID: "2"}
	cloned.Errors = append(cloned.Errors, &DownloadError{ItemID: "3"})

	assert.Equal(t, int64(2), original.TracksDownloaded)
	assert.Equal(t, "1", original.Errors[0].ItemID)
	assert.Len(t, original.Errors, 1)

	assert.Nil(t, (*DownloadStatistics)(nil).Clone())
}

// TestDownloadStatistics_IncrementTrackDownloaded verifies downloaded track and byte counters increment correctly.
func TestDownloadStatistics_IncrementTrackDownloaded(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackDownloaded(1024)
	impl.incrementTrackDownloaded(2048)

	want := &DownloadStatistics{
		TotalTracksProcessed: 2,
		TracksDownloaded:     2,
		TotalBytesDownloaded: 3072,
	}
	assert.Equal(t, want, impl.stats)
}

// TestDownloadStatistics_IncrementTrackSkipped verifies skipped track counters increment by reason.
func TestDownloadStatistics_IncrementTrackSkipped(t *testing.T) {
	t.Parallel()

	impl := newStatisticsService(t, nil)
	impl.incrementTrackSkipped(SkipReasonExists)
	impl.incrementTrackSkipped(SkipReasonQuality)

	want := &DownloadStatistics{
		TotalTracksProcessed: 2,
		TracksSkipped:        2,
		TracksSkippedExists:  1,
		TracksSkippedQuality: 1,
	}
	assert.Equal(t, want, impl.stats)
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

	want := &DownloadStatistics{
		TotalTracksProcessed:  4,
		TracksDownloaded:      2,
		TracksSkipped:         1,
		TracksSkippedDuration: 1,
		TracksFailed:          1,
		TotalBytesDownloaded:  3000,
		LyricsDownloaded:      1,
		LyricsSkipped:         1,
		CoversDownloaded:      1,
		CoversSkipped:         1,
	}
	assert.Equal(t, want, impl.stats)
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

	want := &DownloadStatistics{
		TotalTracksProcessed: 1,
		TracksDownloaded:     1,
		TotalBytesDownloaded: 36860019,
		LyricsDownloaded:     1,
		CoversDownloaded:     1,
	}
	assert.Equal(t, want, impl.stats)
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

	want := &DownloadStatistics{
		TotalTracksProcessed: 10,
		TracksDownloaded:     10,
		TotalBytesDownloaded: 10000,
		LyricsDownloaded:     10,
		CoversDownloaded:     10,
	}
	assert.Equal(t, want, impl.stats)
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

	want := &DownloadStatistics{
		TotalTracksProcessed: 2,
		TracksDownloaded:     2,
		TotalBytesDownloaded: 15000000,
		CoversDownloaded:     1,
	}
	assert.Equal(t, want, impl.stats)
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

	require.Len(t, impl.stats.Errors, 3)

	want := &DownloadError{
		Category:       DownloadCategoryTrack,
		ItemID:         "12345",
		ItemTitle:      "Test Track 1",
		Phase:          "downloading file",
		ParentCategory: DownloadCategoryAlbum,
		ParentID:       "99999",
		ParentTitle:    "Parent Album",
		Error:          assert.AnError,
	}
	assert.Equal(t, want, impl.stats.Errors[0])

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
			assert.Equal(t, tc.expected, stats.FormatDuration(tc.duration))
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
