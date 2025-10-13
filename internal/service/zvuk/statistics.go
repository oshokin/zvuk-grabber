package zvuk

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

const (
	// unknownParentKey is used as a fallback key when parent collection is unknown.
	unknownParentKey = "unknown"
)

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}

	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	return fmt.Sprintf("%ds", seconds)
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

	// Track specific skip reason.
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

// incrementLyricsDownloaded atomically increments the downloaded lyrics counter.
func (s *ServiceImpl) incrementLyricsDownloaded() {
	atomic.AddInt64(&s.stats.LyricsDownloaded, 1)
}

// incrementLyricsSkipped atomically increments the skipped lyrics counter.
func (s *ServiceImpl) incrementLyricsSkipped() {
	atomic.AddInt64(&s.stats.LyricsSkipped, 1)
}

// incrementCoverDownloaded atomically increments the downloaded covers counter.
func (s *ServiceImpl) incrementCoverDownloaded() {
	atomic.AddInt64(&s.stats.CoversDownloaded, 1)
}

// incrementCoverSkipped atomically increments the skipped covers counter.
func (s *ServiceImpl) incrementCoverSkipped() {
	atomic.AddInt64(&s.stats.CoversSkipped, 1)
}

// groupErrors separates track errors from collection errors for better display organization.
func (s *ServiceImpl) groupErrors(errors []DownloadError) (trackErrors, collectionErrors []DownloadError) {
	for i := range errors {
		if errors[i].Category == DownloadCategoryTrack {
			trackErrors = append(trackErrors, errors[i])
		} else {
			collectionErrors = append(collectionErrors, errors[i])
		}
	}

	return trackErrors, collectionErrors
}

// PrintDownloadSummary prints a formatted summary of download statistics.
func (s *ServiceImpl) PrintDownloadSummary(ctx context.Context) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	stats := s.stats

	// If nothing was processed, don't print summary.
	if stats.TotalTracksProcessed == 0 {
		return
	}

	// Check if the context was canceled (CTRL+C or timeout).
	wasInterrupted := ctx.Err() != nil

	s.printSummaryHeader(ctx, wasInterrupted, stats.IsDryRun)
	s.printTrackStatistics(ctx, stats)
	s.printDataTransferStatistics(ctx, stats)
	s.printLyricsStatistics(ctx, stats)
	s.printCoverArtStatistics(ctx, stats)
	s.printSummaryFooter(ctx)
	s.printErrorDetails(ctx, stats)
	s.printFinalMessage(ctx, wasInterrupted, stats)
	s.printDryRunSuggestion(ctx, stats)
}

// printSummaryHeader prints the summary header.
func (s *ServiceImpl) printSummaryHeader(ctx context.Context, wasInterrupted, isDryRun bool) {
	logger.Info(ctx, "")

	switch {
	case isDryRun:
		logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
		logger.Info(ctx, "                  DRY-RUN PREVIEW")
		logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	case wasInterrupted:
		logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
		logger.Info(ctx, "           DOWNLOAD SUMMARY (Interrupted)")
		logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	default:
		logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
		logger.Info(ctx, "                     DOWNLOAD SUMMARY")
		logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	}
}

// printTrackStatistics prints track download statistics.
func (s *ServiceImpl) printTrackStatistics(ctx context.Context, stats *DownloadStatistics) {
	if stats.IsDryRun {
		s.printTrackStatisticsDryRun(ctx, stats)
	} else {
		s.printTrackStatisticsRegular(ctx, stats)
	}
}

// printTrackStatisticsDryRun prints track statistics for dry-run preview mode.
func (s *ServiceImpl) printTrackStatisticsDryRun(ctx context.Context, stats *DownloadStatistics) {
	logger.Infof(ctx, "Tracks:           %d total", stats.TotalTracksProcessed)

	if stats.TracksDownloaded > 0 {
		logger.Infof(ctx, "  Would Download: %d", stats.TracksDownloaded)
	}

	if stats.TracksSkipped > 0 {
		logger.Infof(ctx, "  Already Have:    %d", stats.TracksSkippedExists)

		if stats.TracksSkippedQuality > 0 {
			logger.Infof(ctx, "  Quality Filter:  %d", stats.TracksSkippedQuality)
		}

		if stats.TracksSkippedDuration > 0 {
			logger.Infof(ctx, "  Duration Filter: %d", stats.TracksSkippedDuration)
		}
	}

	if stats.TracksFailed > 0 {
		logger.Infof(ctx, "  Unavailable:     %d", stats.TracksFailed)
	}
}

// printTrackStatisticsRegular prints track statistics for regular download mode.
func (s *ServiceImpl) printTrackStatisticsRegular(ctx context.Context, stats *DownloadStatistics) {
	logger.Infof(ctx, "Tracks:           %d total processed", stats.TotalTracksProcessed)

	if stats.TracksDownloaded > 0 {
		logger.Infof(ctx, "  Downloaded:      %d", stats.TracksDownloaded)
	}

	if stats.TracksSkipped > 0 {
		logger.Infof(ctx, "  Skipped:         %d total", stats.TracksSkipped)

		if stats.TracksSkippedExists > 0 {
			logger.Infof(ctx, "    Already Exist: %d", stats.TracksSkippedExists)
		}

		if stats.TracksSkippedQuality > 0 {
			logger.Infof(ctx, "    Quality:       %d", stats.TracksSkippedQuality)
		}

		if stats.TracksSkippedDuration > 0 {
			logger.Infof(ctx, "    Duration:      %d", stats.TracksSkippedDuration)
		}
	}

	if stats.TracksFailed > 0 {
		logger.Infof(ctx, "  Failed:          %d", stats.TracksFailed)
	}

	// Success rate.
	if stats.TotalTracksProcessed > 0 {
		successCount := stats.TracksDownloaded + stats.TracksSkipped
		successRate := float64(successCount) / float64(stats.TotalTracksProcessed) * 100
		logger.Infof(ctx, "  Success Rate:    %.1f%%", successRate)
	}
}

// printDataTransferStatistics prints data transfer statistics.
func (s *ServiceImpl) printDataTransferStatistics(ctx context.Context, stats *DownloadStatistics) {
	if stats.TotalBytesDownloaded > 0 {
		logger.Info(ctx, "")

		if stats.IsDryRun {
			//nolint:gosec // TotalBytesDownloaded is always positive, no overflow risk.
			logger.Infof(ctx, "Estimated Size:   %s", humanize.Bytes(uint64(stats.TotalBytesDownloaded)))
		} else {
			//nolint:gosec // TotalBytesDownloaded is always positive, no overflow risk.
			logger.Infof(ctx, "Data Downloaded:  %s", humanize.Bytes(uint64(stats.TotalBytesDownloaded)))
		}
	}

	// Print duration if we have both start and end times (skip for dry-run).
	if !stats.IsDryRun && !stats.StartTime.IsZero() && !stats.EndTime.IsZero() {
		duration := stats.EndTime.Sub(stats.StartTime)

		// Only show if duration is meaningful (> 100ms).
		if duration > 100*time.Millisecond {
			logger.Infof(ctx, "Duration:         %s", formatDuration(duration))

			// Calculate and show average speed if we downloaded anything.
			if stats.TotalBytesDownloaded > 0 {
				bytesPerSecond := float64(stats.TotalBytesDownloaded) / duration.Seconds()
				logger.Infof(ctx, "Average Speed:    %s/s", humanize.Bytes(uint64(bytesPerSecond)))
			}
		}
	}
}

// printLyricsStatistics prints lyrics download statistics.
func (s *ServiceImpl) printLyricsStatistics(ctx context.Context, stats *DownloadStatistics) {
	totalLyrics := stats.LyricsDownloaded + stats.LyricsSkipped
	if totalLyrics == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Infof(ctx, "Lyrics:           %d total", totalLyrics)

	if stats.LyricsDownloaded > 0 {
		logger.Infof(ctx, "  Downloaded:     %d", stats.LyricsDownloaded)
	}

	if stats.LyricsSkipped > 0 {
		logger.Infof(ctx, "  Skipped:        %d", stats.LyricsSkipped)
	}
}

// printCoverArtStatistics prints cover art download statistics.
func (s *ServiceImpl) printCoverArtStatistics(ctx context.Context, stats *DownloadStatistics) {
	totalCovers := stats.CoversDownloaded + stats.CoversSkipped
	if totalCovers == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Infof(ctx, "Cover Art:        %d total", totalCovers)

	if stats.CoversDownloaded > 0 {
		logger.Infof(ctx, "  Downloaded:     %d", stats.CoversDownloaded)
	}

	if stats.CoversSkipped > 0 {
		logger.Infof(ctx, "  Skipped:        %d", stats.CoversSkipped)
	}
}

// printSummaryFooter prints the summary footer separator.
func (s *ServiceImpl) printSummaryFooter(ctx context.Context) {
	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
}

// printErrorDetails prints detailed error information if any errors occurred.
func (s *ServiceImpl) printErrorDetails(ctx context.Context, stats *DownloadStatistics) {
	if len(stats.Errors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "ERRORS ENCOUNTERED: %d", len(stats.Errors))

	// Group errors by parent collection for better readability.
	trackErrors, collectionErrors := s.groupErrors(stats.Errors)

	s.printCollectionErrors(ctx, collectionErrors)
	s.printTrackErrors(ctx, trackErrors)

	logger.Info(ctx, "")
	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")

	// Print retry command for failed items.
	s.printRetryCommand(ctx, stats.Errors)
}

// printCollectionErrors prints collection-level errors (albums, playlists, artists).
func (s *ServiceImpl) printCollectionErrors(ctx context.Context, collectionErrors []DownloadError) {
	if len(collectionErrors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "COLLECTION ERRORS:")

	for i := range collectionErrors {
		logger.Info(ctx, "")
		logger.Errorf(ctx, "  [%d] %s: %s", i+1, collectionErrors[i].Category, collectionErrors[i].ItemTitle)

		if collectionErrors[i].ItemURL != "" {
			logger.Errorf(ctx, "      URL: %s", collectionErrors[i].ItemURL)
		}

		logger.Errorf(ctx, "      ID: %s", collectionErrors[i].ItemID)
		logger.Errorf(ctx, "      Phase: %s", collectionErrors[i].Phase)
		logger.Errorf(ctx, "      Error: %s", collectionErrors[i].ErrorMessage)
	}
}

// printTrackErrors prints track-level errors grouped by parent collection.
func (s *ServiceImpl) printTrackErrors(ctx context.Context, trackErrors []DownloadError) {
	if len(trackErrors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "TRACK ERRORS:")

	// Group by parent.
	parentGroups := s.groupTrackErrorsByParent(trackErrors)

	// Print grouped errors.
	for _, errs := range parentGroups {
		if len(errs) == 0 {
			continue
		}

		s.printParentGroupErrors(ctx, errs)
	}
}

// groupTrackErrorsByParent groups track errors by their parent collection.
func (s *ServiceImpl) groupTrackErrorsByParent(trackErrors []DownloadError) map[string][]DownloadError {
	parentGroups := make(map[string][]DownloadError)

	for i := range trackErrors {
		key := trackErrors[i].ParentID
		if key == "" {
			key = unknownParentKey
		}

		parentGroups[key] = append(parentGroups[key], trackErrors[i])
	}

	return parentGroups
}

// printParentGroupErrors prints errors for tracks from a specific parent collection.
func (s *ServiceImpl) printParentGroupErrors(ctx context.Context, errs []DownloadError) {
	firstErr := errs[0]

	logger.Info(ctx, "")

	if firstErr.ParentTitle != "" {
		logger.Errorf(ctx, "  From %s: %s (ID: %s)",
			firstErr.ParentCategory, firstErr.ParentTitle, firstErr.ParentID)
	} else {
		logger.Errorf(ctx, "  From unknown collection:")
	}

	for i := range errs {
		logger.Info(ctx, "")
		logger.Errorf(ctx, "    [%d] %s", i+1, errs[i].ItemTitle)
		logger.Errorf(ctx, "        Track ID: %s", errs[i].ItemID)
		logger.Errorf(ctx, "        Phase: %s", errs[i].Phase)
		logger.Errorf(ctx, "        Error: %s", errs[i].ErrorMessage)
	}
}

// printRetryCommand generates and prints a command to retry failed downloads.
func (s *ServiceImpl) printRetryCommand(ctx context.Context, errors []DownloadError) {
	if len(errors) == 0 {
		return
	}

	// Collect unique URLs from failed items (only collections, not individual tracks).
	var (
		urlsMap = make(map[string]bool)
		urls    []string
	)

	for i := range errors {
		// Only include collection-level errors (albums, playlists, artists).
		// Individual track errors are part of their parent collection.
		if errors[i].Category != DownloadCategoryTrack && errors[i].ItemURL != "" {
			if !urlsMap[errors[i].ItemURL] {
				urlsMap[errors[i].ItemURL] = true
				urls = append(urls, errors[i].ItemURL)
			}
		}
	}

	// If we have any failed collections, show the retry command.
	if len(urls) > 0 {
		logger.Info(ctx, "")
		logger.Info(ctx, "To retry only failed downloads, run:")
		logger.Info(ctx, "")

		// Build command line.
		command := "zvuk-grabber download"
		for _, url := range urls {
			command += " " + url
		}

		logger.Infof(ctx, "  %s", command)
	}
}

// printDryRunSuggestion prints a suggestion to proceed with actual download after dry-run.
func (s *ServiceImpl) printDryRunSuggestion(ctx context.Context, stats *DownloadStatistics) {
	if !stats.IsDryRun || stats.TracksDownloaded == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Info(ctx, "To proceed with actual download, remove the --dry-run flag:")
	logger.Info(ctx, "  zvuk-grabber <same command without --dry-run>")
}

// printFinalMessage prints a helpful message based on download results.
func (s *ServiceImpl) printFinalMessage(ctx context.Context, wasInterrupted bool, stats *DownloadStatistics) {
	// Dry-run specific messages.
	if stats.IsDryRun {
		if stats.TracksDownloaded == 0 && stats.TracksSkipped > 0 {
			logger.Info(ctx, "")
			logger.Info(ctx, "All tracks already exist - nothing to download.")
		}

		return
	}

	// Regular download messages.
	switch {
	case wasInterrupted:
		logger.Info(ctx, "")
		logger.Warn(ctx, "Download interrupted by user (CTRL+C).")

		if stats.TracksDownloaded > 0 {
			logger.Infof(ctx, "Successfully downloaded %d track(s) before interruption.", stats.TracksDownloaded)
		}
	case len(stats.Errors) > 0:
		logger.Info(ctx, "")
		logger.Warnf(ctx, "%d error(s) occurred during download. See detailed error log above.", len(stats.Errors))
	case stats.TracksDownloaded > 0:
		logger.Info(ctx, "")
		logger.Info(ctx, "All downloads completed successfully!")
	case stats.TracksSkipped > 0 && stats.TracksDownloaded == 0:
		logger.Info(ctx, "")
		logger.Info(ctx, "All tracks already exist in the output directory.")
	}
}
