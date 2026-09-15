package stats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// errorDetail describes one optional error detail line.
type errorDetail struct {
	// format is the printf-style template for the detail line.
	format string
	// value is the rendered detail text.
	value string
}

const (
	// Separator separates visually important sections in the download summary.
	Separator = "═══════════════════════════════════════════════════════════════"
	// unknownParentKey is the grouping key for track errors without parent metadata.
	unknownParentKey = "unknown"
)

// Print renders a provider-independent download summary.
func Print(ctx context.Context, r *Report) {
	if r == nil || !r.HasWork() {
		return
	}

	printHeader(ctx, r.WasInterrupted, r.IsDryRun)
	logger.Infof(ctx, "Provider:         %s", r.Provider)

	if r.IsDryRun {
		printTracksDryRun(ctx, &r.Tracks)
	} else {
		printTracksRegular(ctx, &r.Tracks)
	}

	printDataTransfer(ctx, r)

	for _, asset := range r.Assets {
		if asset != nil {
			printAsset(ctx, asset)
		}
	}

	logger.Info(ctx, Separator)
	printErrors(ctx, r)
	printFinalMessage(ctx, r)
	printDryRunSuggestion(ctx, r)
}

// FormatDuration formats a duration into a compact human-readable string.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	h, m, s := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}

	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}

	return fmt.Sprintf("%ds", s)
}

// printHeader writes the summary title banner for regular, dry-run, or interrupted sessions.
func printHeader(ctx context.Context, interrupted, dryRun bool) {
	title := "                     DOWNLOAD SUMMARY"

	switch {
	case dryRun:
		title = "                  DRY-RUN PREVIEW"
	case interrupted:
		title = "           DOWNLOAD SUMMARY (Interrupted)"
	}

	logger.Info(ctx, "")
	logger.Info(ctx, Separator)
	logger.Info(ctx, title)
	logger.Info(ctx, Separator)
}

// printTracksDryRun writes track counters using dry-run wording.
func printTracksDryRun(ctx context.Context, t *TrackCounters) {
	logger.Infof(ctx, "Tracks:           %d total", t.TotalProcessed)
	positive(ctx, "  Would Download: %d", t.Downloaded)

	if t.Skipped > 0 {
		logger.Infof(ctx, "  Already Have:    %d", t.SkippedExists)
		positive(ctx, "  Quality Filter:  %d", t.SkippedQuality)
		positive(ctx, "  Duration Filter: %d", t.SkippedDuration)
	}

	positive(ctx, "  Unavailable:     %d", t.Failed)
}

// printTracksRegular writes track counters for completed download sessions.
func printTracksRegular(ctx context.Context, t *TrackCounters) {
	logger.Infof(ctx, "Tracks:           %d total processed", t.TotalProcessed)
	positive(ctx, "  Downloaded:      %d", t.Downloaded)

	if t.Skipped > 0 {
		logger.Infof(ctx, "  Skipped:         %d total", t.Skipped)
		positive(ctx, "    Already Exist: %d", t.SkippedExists)
		positive(ctx, "    Quality:       %d", t.SkippedQuality)
		positive(ctx, "    Duration:      %d", t.SkippedDuration)
	}

	positive(ctx, "  Failed:          %d", t.Failed)

	if t.TotalProcessed > 0 {
		success := t.Downloaded + t.Skipped
		logger.Infof(ctx, "  Success Rate:    %.1f%%", float64(success)/float64(t.TotalProcessed)*100)
	}
}

// positive logs a formatted counter line only when the value is greater than zero.
func positive(ctx context.Context, format string, value int64) {
	if value > 0 {
		logger.Infof(ctx, format, value)
	}
}

// printDataTransfer writes downloaded size, duration, and average speed when available.
func printDataTransfer(ctx context.Context, r *Report) {
	if r.BytesDownloaded > 0 {
		logger.Info(ctx, "")

		label := "Data Downloaded: "
		if r.IsDryRun {
			label = "Estimated Size:  "
		}

		logger.Infof(ctx, "%s %s", label, humanize.IBytes(uint64(r.BytesDownloaded)))
	}

	d, ok := r.Duration()
	if !ok {
		return
	}

	logger.Infof(ctx, "Duration:         %s", FormatDuration(d))

	if r.BytesDownloaded > 0 {
		bps := float64(r.BytesDownloaded) / d.Seconds()
		logger.Infof(ctx, "Average Speed:    %s/s", humanize.IBytes(uint64(bps)))
	}
}

// printAsset writes one sidecar asset section such as lyrics or cover art.
func printAsset(ctx context.Context, a *AssetCounters) {
	total := a.Downloaded + a.Skipped
	if total == 0 {
		return
	}

	if a.LeadingBlankLine {
		logger.Info(ctx, "")
	}

	logger.Infof(ctx, "%-18s%d total", a.Title+":", total)
	positive(ctx, "  Downloaded:     %d", a.Downloaded)
	positive(ctx, "  Skipped:        %d", a.Skipped)
}

// printErrors writes the error summary and optional retry command.
func printErrors(ctx context.Context, r *Report) {
	if len(r.Errors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "ERRORS ENCOUNTERED: %d", len(r.Errors))

	track, item, generic := splitErrors(r.Errors)
	printItemErrors(ctx, item)
	printTrackErrors(ctx, track)
	printGenericErrors(ctx, generic)

	logger.Info(ctx, "")
	logger.Info(ctx, Separator)
	printRetryCommand(ctx, r)
}

// splitErrors partitions summary errors into track, item, and generic groups.
func splitErrors(errors []*Error) (track, item, generic []*Error) {
	for _, err := range errors {
		if err == nil {
			continue
		}

		switch {
		case err.IsTrack:
			track = append(track, err)
		case err.HasDetails():
			item = append(item, err)
		default:
			generic = append(generic, err)
		}
	}

	return track, item, generic
}

// printItemErrors writes structured failures for albums, playlists, and other items.
func printItemErrors(ctx context.Context, errors []*Error) {
	if len(errors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "ITEM ERRORS:")

	for i, err := range errors {
		if err == nil {
			continue
		}

		logger.Info(ctx, "")
		logger.Errorf(ctx, "  [%d] %s: %s", i+1, fallback(err.Category, "item"), fallback(err.ItemTitle, err.ItemURL))
		fields(ctx,
			errorDetail{format: "      URL: %s", value: err.ItemURL},
			errorDetail{format: "      ID: %s", value: err.ItemID},
			errorDetail{format: "      Phase: %s", value: err.Phase},
			errorDetail{format: "      Error: %s", value: err.ErrorMessage()},
		)
	}
}

// printTrackErrors writes track failures grouped by parent collection.
func printTrackErrors(ctx context.Context, errors []*Error) {
	if len(errors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "TRACK ERRORS:")

	for _, group := range groupTrackErrors(errors) {
		printParentGroupErrors(ctx, group)
	}
}

// printGenericErrors writes failures that lack structured item metadata.
func printGenericErrors(ctx context.Context, errors []*Error) {
	if len(errors) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Errorf(ctx, "GENERAL ERRORS:")

	for i, err := range errors {
		if err == nil {
			continue
		}

		logger.Errorf(ctx, "  [%d] %s", i+1, err.ErrorMessage())
	}
}

// groupTrackErrors groups track failures by parent collection ID.
func groupTrackErrors(errors []*Error) map[string][]*Error {
	groups := make(map[string][]*Error)

	for _, err := range errors {
		if err == nil {
			continue
		}

		key := err.ParentID
		if key == "" {
			key = unknownParentKey
		}

		groups[key] = append(groups[key], err)
	}

	return groups
}

// printParentGroupErrors writes one parent collection and its track failures.
func printParentGroupErrors(ctx context.Context, errors []*Error) {
	if len(errors) == 0 {
		return
	}

	first := errors[0]
	if first == nil {
		return
	}

	logger.Info(ctx, "")

	if first.ParentTitle != "" {
		logger.Errorf(ctx, "  From %s: %s (ID: %s)", first.ParentCategory, first.ParentTitle, first.ParentID)
	} else {
		logger.Errorf(ctx, "  From unknown collection:")
	}

	for i, err := range errors {
		if err == nil {
			continue
		}

		logger.Info(ctx, "")
		logger.Errorf(ctx, "    [%d] %s", i+1, fallback(err.ItemTitle, err.ItemURL))
		fields(ctx,
			errorDetail{format: "        Track ID: %s", value: err.ItemID},
			errorDetail{format: "        Phase: %s", value: err.Phase},
			errorDetail{format: "        Error: %s", value: err.ErrorMessage()},
		)
	}
}

// fields logs optional error detail lines when their values are non-empty.
func fields(ctx context.Context, details ...errorDetail) {
	for _, detail := range details {
		if detail.value != "" {
			logger.Errorf(ctx, detail.format, detail.value)
		}
	}
}

// printRetryCommand writes a retry command for failed non-track URLs.
func printRetryCommand(ctx context.Context, r *Report) {
	urls := retryURLs(r.Errors)
	if r.RetryCommandBase == "" || len(urls) == 0 {
		return
	}

	logger.Info(ctx, "")
	logger.Info(ctx, "To retry only failed downloads, run:")
	logger.Info(ctx, "")
	logger.Infof(ctx, "  %s %s", r.RetryCommandBase, strings.Join(urls, " "))
}

// retryURLs collects unique item URLs suitable for a retry command.
func retryURLs(errors []*Error) []string {
	seen := make(map[string]bool)

	var urls []string

	for _, err := range errors {
		if err == nil || err.IsTrack || err.ItemURL == "" || seen[err.ItemURL] {
			continue
		}

		seen[err.ItemURL] = true
		urls = append(urls, err.ItemURL)
	}

	return urls
}

// printDryRunSuggestion writes the follow-up command hint after a dry-run preview.
func printDryRunSuggestion(ctx context.Context, r *Report) {
	if !r.IsDryRun || r.Tracks.Downloaded == 0 || r.DryRunSuggestion == "" {
		return
	}

	logger.Info(ctx, "")
	logger.Info(ctx, "To proceed with actual download, remove the --dry-run flag:")
	logger.Infof(ctx, "  %s", r.DryRunSuggestion)
}

// printFinalMessage writes the closing status line for the summary.
func printFinalMessage(ctx context.Context, r *Report) {
	if r.IsDryRun {
		if r.Tracks.Downloaded == 0 && r.Tracks.Skipped > 0 {
			logger.Info(ctx, "")
			logger.Info(ctx, "All tracks already exist - nothing to download.")
		}

		return
	}

	switch {
	case r.WasInterrupted:
		logger.Info(ctx, "")
		logger.Warn(ctx, "Download interrupted by user (CTRL+C).")
		positive(ctx, "Successfully downloaded %d track(s) before interruption.", r.Tracks.Downloaded)
	case len(r.Errors) > 0:
		logger.Info(ctx, "")
		logger.Warnf(ctx, "%d error(s) occurred during download. See detailed error log above.", len(r.Errors))
	case r.Tracks.Downloaded > 0:
		logger.Info(ctx, "")
		logger.Info(ctx, "All downloads completed successfully!")
	case r.Tracks.Skipped > 0:
		logger.Info(ctx, "")
		logger.Info(ctx, "All tracks already exist in the output directory.")
	}
}

// fallback returns the first non-empty string or "unknown".
func fallback(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return "unknown"
}
