package stats

import "time"

// SkipReason describes why a track was skipped by provider-specific policy.
type SkipReason uint8

// AssetKey identifies a provider-independent sidecar asset counter.
type AssetKey string

// AssetSpec maps an internal asset key to a rendered summary section.
type AssetSpec struct {
	// Key identifies the asset counter inside a Session.
	Key AssetKey
	// Title is the summary section label, for example "Lyrics" or "Cover Art".
	Title string
	// LeadingBlankLine inserts a blank line before the section when true.
	LeadingBlankLine bool
}

// Session accumulates provider-independent download counters for one run.
type Session struct {
	// StartTime is when the download session began.
	StartTime time.Time
	// EndTime is when the download session completed.
	EndTime time.Time
	// IsDryRun reports whether the session ran as a preview without writing files.
	IsDryRun bool
	// Tracks holds audio-track counters for the session.
	Tracks TrackCounters
	// BytesDownloaded is the total audio bytes written or estimated during the session.
	BytesDownloaded int64
	// Assets holds sidecar counters by provider-independent asset key.
	Assets map[AssetKey]*AssetCounters
	// Errors collects structured failures for the summary report.
	Errors []*Error
}

const (
	// SkipReasonUnknown means the provider did not classify the skip.
	SkipReasonUnknown SkipReason = iota
	// SkipReasonExists means the output file already exists.
	SkipReasonExists
	// SkipReasonQuality means the resolved track quality is below the configured minimum.
	SkipReasonQuality
	// SkipReasonDuration means the track duration is outside configured bounds.
	SkipReasonDuration
)

const (
	// AssetLyrics is the shared key for lyrics sidecar files.
	AssetLyrics AssetKey = "lyrics"
	// AssetCoverArt is the shared key for cover images.
	AssetCoverArt AssetKey = "cover_art"
	// AssetDescription is the shared key for collection description files.
	AssetDescription AssetKey = "description"
)

// NewSession creates a fresh statistics session.
func NewSession(start time.Time, dryRun bool) *Session {
	return &Session{
		StartTime: start,
		IsDryRun:  dryRun,
		Assets:    make(map[AssetKey]*AssetCounters),
	}
}

// Finish records the session completion time.
func (s *Session) Finish(end time.Time) {
	if s == nil {
		return
	}

	s.EndTime = end
}

// RecordProcessed increments the number of tracks that entered the pipeline.
func (s *Session) RecordProcessed() {
	if s == nil {
		return
	}

	s.Tracks.TotalProcessed++
}

// RecordDownloaded increments downloaded tracks and downloaded bytes.
func (s *Session) RecordDownloaded(bytes int64) {
	if s == nil {
		return
	}

	s.Tracks.Downloaded++
	s.BytesDownloaded += bytes
}

// RecordSkipped increments skipped counters by a normalized reason.
func (s *Session) RecordSkipped(reason SkipReason) {
	if s == nil {
		return
	}

	s.Tracks.Skipped++

	switch reason {
	case SkipReasonExists:
		s.Tracks.SkippedExists++
	case SkipReasonQuality:
		s.Tracks.SkippedQuality++
	case SkipReasonDuration:
		s.Tracks.SkippedDuration++
	}
}

// RecordFailed increments the failed track counter.
func (s *Session) RecordFailed() {
	if s == nil {
		return
	}

	s.Tracks.Failed++
}

// RecordAssetDownloaded increments a sidecar downloaded counter.
func (s *Session) RecordAssetDownloaded(key AssetKey) {
	if s == nil {
		return
	}

	s.ensureAssets()

	asset := s.asset(key)
	asset.Downloaded++
	s.Assets[key] = asset
}

// RecordAssetSkipped increments a sidecar skipped counter.
func (s *Session) RecordAssetSkipped(key AssetKey) {
	if s == nil {
		return
	}

	s.ensureAssets()

	asset := s.asset(key)
	asset.Skipped++
	s.Assets[key] = asset
}

// AddError appends a structured download error.
func (s *Session) AddError(err *Error) {
	if s == nil || err == nil {
		return
	}

	s.Errors = append(s.Errors, err)
}

// Clone returns a detached copy safe for rendering outside the service lock.
func (s *Session) Clone() *Session {
	if s == nil {
		return nil
	}

	cloned := make(map[AssetKey]*AssetCounters, len(s.Assets))
	for key, asset := range s.Assets {
		cloned[key] = asset.clone()
	}

	return &Session{
		StartTime:       s.StartTime,
		EndTime:         s.EndTime,
		IsDryRun:        s.IsDryRun,
		Tracks:          s.Tracks,
		BytesDownloaded: s.BytesDownloaded,
		Assets:          cloned,
		Errors:          append([]*Error(nil), s.Errors...),
	}
}

// HasWork reports whether the session contains anything useful to print.
func (s *Session) HasWork() bool {
	if s == nil {
		return false
	}

	return s.Tracks.TotalProcessed > 0 || len(s.Errors) > 0
}

// Report converts the session snapshot into a printable summary report.
func (s *Session) Report(cfg *ReportConfig) *Report {
	if s == nil {
		return nil
	}

	assets := make([]*AssetCounters, 0, len(cfg.Assets))
	for _, spec := range cfg.Assets {
		counterCopy := s.asset(spec.Key).clone()
		counterCopy.Title = spec.Title
		counterCopy.LeadingBlankLine = spec.LeadingBlankLine
		assets = append(assets, counterCopy)
	}

	return NewReport(cfg, &ReportSnapshot{
		IsDryRun:        s.IsDryRun,
		StartTime:       s.StartTime,
		EndTime:         s.EndTime,
		Tracks:          s.Tracks,
		BytesDownloaded: s.BytesDownloaded,
		Assets:          assets,
		Errors:          append([]*Error(nil), s.Errors...),
	})
}

// ensureAssets lazily initializes sidecar counters for zero-value sessions.
func (s *Session) ensureAssets() {
	if s.Assets == nil {
		s.Assets = make(map[AssetKey]*AssetCounters)
	}
}

// asset returns an initialized sidecar counter for key.
func (s *Session) asset(key AssetKey) *AssetCounters {
	if s == nil || s.Assets == nil {
		return &AssetCounters{}
	}

	asset := s.Assets[key]
	if asset != nil {
		return asset
	}

	return &AssetCounters{}
}
