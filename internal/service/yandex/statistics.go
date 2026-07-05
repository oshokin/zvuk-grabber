package yandex

import (
	"context"
	"errors"

	"github.com/oshokin/zvuk-grabber/internal/service/stats"
)

const (
	// retryCommandBase is the command prefix printed for retry hints.
	retryCommandBase = "zvuk-grabber"
	// dryRunSuggestion is the short command hint printed after dry-run previews.
	dryRunSuggestion = "zvuk-grabber <same command without --dry-run>"
)

// yandexSummaryAssets defines the asset sections shown in Yandex download summaries.
var yandexSummaryAssets = []stats.AssetSpec{
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

// recordProcessed increments the processed track counter.
func (s *ServiceImpl) recordProcessed() {
	s.updateStats(func(session *stats.Session) { session.RecordProcessed() })
}

// recordDownloaded increments downloaded track and byte counters.
func (s *ServiceImpl) recordDownloaded(bytes int64) {
	s.updateStats(func(session *stats.Session) { session.RecordDownloaded(bytes) })
}

// recordSkipped increments skipped counters by skip reason.
func (s *ServiceImpl) recordSkipped(reason yandexSkipReason) {
	s.updateStats(func(session *stats.Session) { session.RecordSkipped(yandexStatsSkipReason(reason)) })
}

// recordFailure appends a generic failure and increments the failed counter.
func (s *ServiceImpl) recordFailure(err error) {
	s.recordDownloadError(&stats.Error{Err: err})
}

// recordSourceFailure records a source URL failure, for example URL parsing or metadata resolving.
func (s *ServiceImpl) recordSourceFailure(rawURL, phase string, err error) {
	s.recordDownloadError(&stats.Error{
		Category:  "source",
		ItemURL:   rawURL,
		ItemTitle: rawURL,
		Phase:     phase,
		Err:       err,
	})
}

// recordTrackFailure records a track-scoped failure with enough context for the shared summary printer.
func (s *ServiceImpl) recordTrackFailure(job *trackJob, phase string, err error) {
	if job == nil || job.track == nil {
		s.recordDownloadError(&stats.Error{
			Category: "track",
			IsTrack:  true,
			Phase:    phase,
			Err:      err,
		})

		return
	}

	s.recordDownloadError(&stats.Error{
		Category:       collectionTrack,
		IsTrack:        true,
		ItemID:         yandexJobTrackID(job),
		ItemTitle:      job.track.FullTitle(),
		ItemURL:        job.sourceURL,
		ParentCategory: job.kind,
		ParentID:       job.collectionID,
		ParentTitle:    job.collectionTitle,
		Phase:          phase,
		Err:            err,
	})
}

// recordDownloadError appends a failure and increments the failed counter.
func (s *ServiceImpl) recordDownloadError(err *stats.Error) {
	if err == nil || err.Err == nil || errors.Is(err.Err, context.Canceled) {
		return
	}

	s.updateStats(func(session *stats.Session) {
		session.RecordFailed()
		session.AddError(err)
	})
}

// recordLyricsDownloaded increments the downloaded lyrics counter.
func (s *ServiceImpl) recordLyricsDownloaded() {
	s.updateStats(func(session *stats.Session) { session.RecordAssetDownloaded(stats.AssetLyrics) })
}

// recordLyricsSkipped increments the skipped lyrics counter.
func (s *ServiceImpl) recordLyricsSkipped() {
	s.updateStats(func(session *stats.Session) { session.RecordAssetSkipped(stats.AssetLyrics) })
}

// recordCoverDownloaded increments the downloaded cover counter.
func (s *ServiceImpl) recordCoverDownloaded() {
	s.updateStats(func(session *stats.Session) { session.RecordAssetDownloaded(stats.AssetCoverArt) })
}

// recordCoverSkipped increments the skipped cover counter.
func (s *ServiceImpl) recordCoverSkipped() {
	s.updateStats(func(session *stats.Session) { session.RecordAssetSkipped(stats.AssetCoverArt) })
}

// recordDescriptionSaved increments the saved description counter.
func (s *ServiceImpl) recordDescriptionSaved() {
	s.updateStats(func(session *stats.Session) { session.RecordAssetDownloaded(stats.AssetDescription) })
}

// recordDescriptionSkipped increments the skipped description counter.
func (s *ServiceImpl) recordDescriptionSkipped() {
	s.updateStats(func(session *stats.Session) { session.RecordAssetSkipped(stats.AssetDescription) })
}

// updateStats applies a statistics mutation under the service statistics lock.
func (s *ServiceImpl) updateStats(update func(*stats.Session)) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	update(s.sessionStats)
}

// statsSnapshot returns a copy of the current session statistics.
func (s *ServiceImpl) statsSnapshot() *stats.Session {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	return s.sessionStats.Clone()
}

// PrintDownloadSummary logs a formatted end-of-session download summary.
func (s *ServiceImpl) PrintDownloadSummary(ctx context.Context) {
	snapshot := s.statsSnapshot()
	if !snapshot.HasWork() {
		return
	}

	stats.Print(ctx, yandexStatsReport(ctx, snapshot))
}

// yandexStatsReport adapts Yandex-specific settings to the shared summary renderer.
func yandexStatsReport(ctx context.Context, snapshot *stats.Session) *stats.Report {
	return snapshot.Report(&stats.ReportConfig{
		Provider:         providerName,
		WasInterrupted:   ctx.Err() != nil,
		Assets:           yandexSummaryAssets,
		RetryCommandBase: retryCommandBase,
		DryRunSuggestion: dryRunSuggestion,
	})
}

// yandexStatsSkipReason converts provider-local skip reasons to shared statistics reasons.
func yandexStatsSkipReason(reason yandexSkipReason) stats.SkipReason {
	switch reason {
	case yandexSkipReasonExists:
		return stats.SkipReasonExists
	case yandexSkipReasonQuality:
		return stats.SkipReasonQuality
	case yandexSkipReasonDuration:
		return stats.SkipReasonDuration
	default:
		return stats.SkipReasonUnknown
	}
}
