package zvuk

import (
	"context"
	"errors"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/service/stats"
)

const (
	// summaryProviderName is the display name used in download summaries.
	summaryProviderName = "Zvuk"
	// retryCommandBase is the command prefix printed for retry hints.
	retryCommandBase = "zvuk-grabber"
	// dryRunSuggestion is the short command hint printed after dry-run previews.
	dryRunSuggestion = "zvuk-grabber <same command without --dry-run>"
)

// zvukSummaryAssets defines the asset sections shown in Zvuk download summaries.
var zvukSummaryAssets = []*stats.AssetSpec{
	{
		Key:              stats.AssetLyrics,
		Title:            "Lyrics",
		LeadingBlankLine: true,
	},
	{
		Key:              stats.AssetCoverArt,
		Title:            "Cover Art",
		LeadingBlankLine: true,
	},
	{
		Key:   stats.AssetDescription,
		Title: "Description",
	},
}

// formatDuration keeps the old package-level helper available for tests and delegates to the shared renderer.
func formatDuration(d time.Duration) string {
	return stats.FormatDuration(d)
}

// incrementTrackDownloaded atomically increments the downloaded tracks counter and adds bytes.
func (s *ServiceImpl) incrementTrackDownloaded(bytes int64) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.TracksDownloaded++
	s.stats.TotalTracksProcessed++
	s.stats.TotalBytesDownloaded += bytes
}

// incrementTrackSkipped atomically increments the skipped tracks counter with reason.
func (s *ServiceImpl) incrementTrackSkipped(reason SkipReason) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.TracksSkipped++
	s.stats.TotalTracksProcessed++

	switch reason {
	case SkipReasonExists:
		s.stats.TracksSkippedExists++
	case SkipReasonQuality:
		s.stats.TracksSkippedQuality++
	case SkipReasonDuration:
		s.stats.TracksSkippedDuration++
	}
}

// incrementTrackFailed atomically increments the failed tracks counter.
func (s *ServiceImpl) incrementTrackFailed() {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.TracksFailed++
	s.stats.TotalTracksProcessed++
}

// incrementLyricsDownloaded safely increments the downloaded lyrics counter.
func (s *ServiceImpl) incrementLyricsDownloaded() {
	s.incrementStat(func(stats *DownloadStatistics) { stats.LyricsDownloaded++ })
}

// incrementLyricsSkipped safely increments the skipped lyrics counter.
func (s *ServiceImpl) incrementLyricsSkipped() {
	s.incrementStat(func(stats *DownloadStatistics) { stats.LyricsSkipped++ })
}

// incrementCoverDownloaded safely increments the downloaded covers counter.
func (s *ServiceImpl) incrementCoverDownloaded() {
	s.incrementStat(func(stats *DownloadStatistics) { stats.CoversDownloaded++ })
}

// incrementCoverSkipped safely increments the skipped covers counter.
func (s *ServiceImpl) incrementCoverSkipped() {
	s.incrementStat(func(stats *DownloadStatistics) { stats.CoversSkipped++ })
}

// incrementDescriptionSaved safely increments the saved descriptions counter.
func (s *ServiceImpl) incrementDescriptionSaved() {
	s.incrementStat(func(stats *DownloadStatistics) { stats.DescriptionsSaved++ })
}

// incrementDescriptionSkipped safely increments the skipped descriptions counter.
func (s *ServiceImpl) incrementDescriptionSkipped() {
	s.incrementStat(func(stats *DownloadStatistics) { stats.DescriptionsSkipped++ })
}

// incrementStat applies a small statistics mutation under the service statistics lock.
func (s *ServiceImpl) incrementStat(update func(*DownloadStatistics)) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	update(s.stats)
}

// PrintDownloadSummary prints a formatted summary of download statistics.
func (s *ServiceImpl) PrintDownloadSummary(ctx context.Context) {
	snapshot := s.statsSnapshot()
	if snapshot.TotalTracksProcessed == 0 && len(snapshot.Errors) == 0 {
		return
	}

	stats.Print(ctx, s.downloadStatsReport(ctx, snapshot))
}

// statsSnapshot returns a detached copy so summary rendering does not hold the stats mutex while logging.
func (s *ServiceImpl) statsSnapshot() *DownloadStatistics {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	snapshot := *s.stats
	snapshot.Errors = append([]*DownloadError(nil), s.stats.Errors...)

	return &snapshot
}

// downloadStatsReport adapts Zvuk-specific statistics to the shared summary renderer.
func (s *ServiceImpl) downloadStatsReport(ctx context.Context, snapshot *DownloadStatistics) *stats.Report {
	return stats.NewReport(&stats.ReportConfig{
		Provider:         summaryProviderName,
		WasInterrupted:   ctx.Err() != nil,
		RetryCommandBase: retryCommandBase,
		DryRunSuggestion: dryRunSuggestion,
	}, &stats.ReportSnapshot{
		IsDryRun:        snapshot.IsDryRun,
		StartTime:       snapshot.StartTime,
		EndTime:         snapshot.EndTime,
		Tracks:          snapshot.TrackCounters(),
		BytesDownloaded: snapshot.TotalBytesDownloaded,
		Assets:          snapshot.AssetCounters(zvukSummaryAssets),
		Errors:          zvukErrors(snapshot.Errors),
	})
}

// TrackCounters returns provider-independent track counters for the shared renderer.
func (d *DownloadStatistics) TrackCounters() stats.TrackCounters {
	if d == nil {
		return stats.TrackCounters{}
	}

	return stats.TrackCounters{
		TotalProcessed:  d.TotalTracksProcessed,
		Downloaded:      d.TracksDownloaded,
		Skipped:         d.TracksSkipped,
		SkippedExists:   d.TracksSkippedExists,
		SkippedQuality:  d.TracksSkippedQuality,
		SkippedDuration: d.TracksSkippedDuration,
		Failed:          d.TracksFailed,
	}
}

// AssetCounters returns provider-independent sidecar counters in configured rendering order.
func (d *DownloadStatistics) AssetCounters(specs []*stats.AssetSpec) []*stats.AssetCounters {
	if d == nil {
		return nil
	}

	assets := make([]*stats.AssetCounters, 0, len(specs))
	for _, spec := range specs {
		asset := d.assetCounter(spec)
		assets = append(assets, asset)
	}

	return assets
}

// assetCounter converts one normalized asset key to Zvuk-specific counters.
func (d *DownloadStatistics) assetCounter(spec *stats.AssetSpec) *stats.AssetCounters {
	asset := &stats.AssetCounters{
		Title:            spec.Title,
		LeadingBlankLine: spec.LeadingBlankLine,
	}

	switch spec.Key {
	case stats.AssetLyrics:
		asset.Downloaded = d.LyricsDownloaded
		asset.Skipped = d.LyricsSkipped
	case stats.AssetCoverArt:
		asset.Downloaded = d.CoversDownloaded
		asset.Skipped = d.CoversSkipped
	case stats.AssetDescription:
		asset.Downloaded = d.DescriptionsSaved
		asset.Skipped = d.DescriptionsSkipped
	}

	return asset
}

// zvukErrors converts rich Zvuk errors to provider-independent summary errors.
func zvukErrors(errors []*DownloadError) []*stats.Error {
	result := make([]*stats.Error, 0, len(errors))
	for _, err := range errors {
		if err == nil {
			continue
		}

		result = append(result, &stats.Error{
			Category:       err.Category.String(),
			IsTrack:        err.Category == DownloadCategoryTrack,
			ItemID:         err.ItemID,
			ItemTitle:      err.ItemTitle,
			ItemURL:        err.ItemURL,
			ParentCategory: err.ParentCategory.String(),
			ParentID:       err.ParentID,
			ParentTitle:    err.ParentTitle,
			Phase:          err.Phase,
			Err:            err.Error,
		})
	}

	return result
}

// recordError records an error in the statistics with proper context.
// Context cancellation errors are ignored as they are expected during graceful shutdown.
func (s *ServiceImpl) recordError(e *DownloadError) {
	if e == nil || e.Error == nil {
		return
	}

	if errors.Is(e.Error, context.Canceled) {
		return
	}

	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.Errors = append(s.stats.Errors, e)
}
