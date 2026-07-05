package stats

import "time"

// TrackCounters contains audio-track counters collected during one download session.
type TrackCounters struct {
	// TotalProcessed is the number of tracks that entered the download pipeline.
	TotalProcessed int64
	// Downloaded is the number of tracks successfully written or marked for download.
	Downloaded int64
	// Skipped is the total number of tracks skipped for any reason.
	Skipped int64
	// SkippedExists is the number of tracks skipped because the target file already exists.
	SkippedExists int64
	// SkippedQuality is the number of tracks skipped for below-minimum quality.
	SkippedQuality int64
	// SkippedDuration is the number of tracks skipped for duration constraints.
	SkippedDuration int64
	// Failed is the number of tracks that failed during download or write.
	Failed int64
}

// AssetCounters contains counters for sidecar files such as lyrics and covers.
type AssetCounters struct {
	// Title is the summary section label, for example "Lyrics" or "Cover Art".
	Title string
	// Downloaded is the number of sidecar files saved during the session.
	Downloaded int64
	// Skipped is the number of sidecar files skipped or reused.
	Skipped int64
	// LeadingBlankLine inserts a blank line before the asset section when true.
	LeadingBlankLine bool
}

// Error contains provider-independent error details for summary rendering.
type Error struct {
	// Category is the item type label shown in the summary, for example "album" or "track".
	Category string
	// IsTrack reports whether the failure belongs to a track inside a collection.
	IsTrack bool
	// ItemID is the unique identifier of the failed item.
	ItemID string
	// ItemTitle is the human-readable title of the failed item.
	ItemTitle string
	// ItemURL is the source URL of the failed item.
	ItemURL string
	// ParentCategory is the collection type for track-scoped failures.
	ParentCategory string
	// ParentID is the identifier of the parent collection.
	ParentID string
	// ParentTitle is the display title of the parent collection.
	ParentTitle string
	// Phase is the pipeline step where the error occurred.
	Phase string
	// Err is the underlying failure.
	Err error
}

// ReportConfig contains provider-level fields that are stable for one rendered summary.
type ReportConfig struct {
	// Provider is the display name of the music provider.
	Provider string
	// WasInterrupted reports whether the session ended because the context was canceled.
	WasInterrupted bool
	// Assets defines the sidecar sections rendered for this provider.
	Assets []AssetSpec
	// RetryCommandBase is the command prefix printed for retry hints.
	RetryCommandBase string
	// DryRunSuggestion is the short command hint printed after dry-run previews.
	DryRunSuggestion string
}

// ReportSnapshot contains per-session counters to be rendered by the shared summary printer.
type ReportSnapshot struct {
	// IsDryRun reports whether the session ran as a preview without writing files.
	IsDryRun bool
	// StartTime is when the download session began.
	StartTime time.Time
	// EndTime is when the download session completed.
	EndTime time.Time
	// Tracks holds audio-track counters for the session.
	Tracks TrackCounters
	// BytesDownloaded is the total audio bytes written or estimated during the session.
	BytesDownloaded int64
	// Assets holds sidecar file counters such as lyrics and covers.
	Assets []*AssetCounters
	// Errors collects structured failures for the summary report.
	Errors []*Error
}

// Report contains all data required to render one provider summary.
type Report struct {
	// Provider is the display name of the music provider.
	Provider string
	// IsDryRun reports whether the session ran as a preview without writing files.
	IsDryRun bool
	// WasInterrupted reports whether the session ended because the context was canceled.
	WasInterrupted bool
	// StartTime is when the download session began.
	StartTime time.Time
	// EndTime is when the download session completed.
	EndTime time.Time
	// Tracks holds audio-track counters for the session.
	Tracks TrackCounters
	// BytesDownloaded is the total audio bytes written or estimated during the session.
	BytesDownloaded int64
	// Assets holds sidecar file counters such as lyrics and covers.
	Assets []*AssetCounters
	// Errors collects structured failures for the summary report.
	Errors []*Error
	// RetryCommandBase is the command prefix printed for retry hints.
	RetryCommandBase string
	// DryRunSuggestion is the short command hint printed after dry-run previews.
	DryRunSuggestion string
}

// NewReport combines provider config and session counters into a printable report.
func NewReport(cfg *ReportConfig, snapshot *ReportSnapshot) *Report {
	return &Report{
		Provider:         cfg.Provider,
		IsDryRun:         snapshot.IsDryRun,
		WasInterrupted:   cfg.WasInterrupted,
		StartTime:        snapshot.StartTime,
		EndTime:          snapshot.EndTime,
		Tracks:           snapshot.Tracks,
		BytesDownloaded:  snapshot.BytesDownloaded,
		Assets:           compactAssets(snapshot.Assets),
		Errors:           append([]*Error(nil), snapshot.Errors...),
		RetryCommandBase: cfg.RetryCommandBase,
		DryRunSuggestion: cfg.DryRunSuggestion,
	}
}

// compactAssets drops nil asset entries while preserving provider-defined order.
func compactAssets(assets []*AssetCounters) []*AssetCounters {
	if len(assets) == 0 {
		return nil
	}

	result := make([]*AssetCounters, 0, len(assets))
	for _, asset := range assets {
		if asset != nil {
			clone := *asset
			result = append(result, &clone)
		}
	}

	return result
}

// HasDetails reports whether the error can be rendered as a structured item.
func (e *Error) HasDetails() bool {
	if e == nil {
		return false
	}

	return e.Category != "" || e.ItemID != "" || e.ItemTitle != "" || e.ItemURL != "" || e.Phase != ""
}

// ErrorMessage returns the underlying error text.
func (e *Error) ErrorMessage() string {
	if e == nil || e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

// HasWork reports whether the summary contains anything useful to print.
func (r *Report) HasWork() bool {
	if r == nil {
		return false
	}

	return r.Tracks.TotalProcessed > 0 || len(r.Errors) > 0
}

// Duration returns a printable completed duration.
func (r *Report) Duration() (time.Duration, bool) {
	if r == nil || r.IsDryRun || r.StartTime.IsZero() || r.EndTime.IsZero() {
		return 0, false
	}

	d := r.EndTime.Sub(r.StartTime)

	return d, d > 100*time.Millisecond
}
