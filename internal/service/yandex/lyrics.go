package yandex

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// lyricsExtension is the file extension used for downloaded lyrics sidecars.
const lyricsExtension = ".lrc"

// downloadAndSaveLyrics fetches lyrics for a track and writes them next to the audio file.
func (s *ServiceImpl) downloadAndSaveLyrics(ctx context.Context, targetPath string, job *trackJob) string {
	if s == nil || s.cfg == nil || !s.cfg.DownloadLyrics || job == nil || job.track == nil || isYandexChapterJob(job) {
		return ""
	}

	if !job.track.LyricsAvailable {
		s.recordLyricsSkipped()
		logger.Infof(ctx, "No lyrics available for Yandex track %q", yandexTrackTitle(job))

		return ""
	}

	logger.Infof(ctx, "Downloading lyrics for track: %s", yandexTrackTitle(job))

	if s.client == nil {
		s.recordLyricsSkipped()
		logger.Errorf(ctx, "Yandex download client is not configured for lyrics download")

		return ""
	}

	text := s.fetchTrackLyricsText(ctx, job)
	if text == "" {
		return ""
	}

	lyricsPath := utils.SetFileExtension(targetPath, lyricsExtension, true)
	if lyricsPath == "" {
		s.recordLyricsSkipped()
		return text
	}

	unlock := s.pathLocks.Lock(lyricsPath)
	defer unlock()

	if s.useExistingLyrics(ctx, lyricsPath) {
		return text
	}

	if err := writeSidecarFile(lyricsPath, text, s.cfg.ReplaceLyrics); err != nil {
		s.recordLyricsSkipped()
		logger.Errorf(ctx, "Failed to save Yandex lyrics for %q: %v", job.track.FullTitle(), err)

		return text
	}

	s.recordLyricsDownloaded()
	logger.Infof(ctx, "Lyrics saved to file: %s", lyricsPath)

	return text
}

// fetchTrackLyricsText retrieves and normalizes lyrics text for a download job.
func (s *ServiceImpl) fetchTrackLyricsText(ctx context.Context, job *trackJob) string {
	trackID := yandexJobTrackID(job)
	if trackID == "" {
		s.recordLyricsSkipped()
		logger.Warnf(ctx, "Track ID is empty, skipping lyrics download for %q", yandexTrackTitle(job))

		return ""
	}

	lyrics, err := s.trackLyricsWithRetry(ctx, trackID)
	if err != nil {
		s.recordLyricsSkipped()

		if isYandexLyricsMissing(err) {
			logger.Infof(ctx, "No lyrics found for Yandex track %q", yandexTrackTitle(job))

			return ""
		}

		logger.Errorf(ctx, "Failed to get lyrics: %v", err)

		return ""
	}

	if lyrics == nil {
		s.recordLyricsSkipped()
		logger.Info(ctx, "Lyrics is empty")

		return ""
	}

	text := strings.TrimSpace(lyrics.Text())
	if text == "" {
		s.recordLyricsSkipped()
		logger.Info(ctx, "Lyrics is empty")

		return ""
	}

	return text
}

// useExistingLyrics skips writing when a lyrics file already exists.
func (s *ServiceImpl) useExistingLyrics(ctx context.Context, lyricsPath string) bool {
	return s.useExistingSidecar(
		ctx,
		lyricsPath,
		s.cfg.ReplaceLyrics,
		s.recordLyricsSkipped,
		"Yandex lyrics file check failed: path=%s err=%v",
		fmt.Sprintf("File '%s' already exists, skipping download", lyricsPath),
	)
}

// isYandexChapterJob reports whether the job represents an audiobook or podcast chapter.
func isYandexChapterJob(job *trackJob) bool {
	return job != nil && (job.kind == collectionAudiobook || job.kind == collectionPodcast)
}

// isYandexLyricsMissing reports whether the API returned a terminal missing-lyrics response.
func isYandexLyricsMissing(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *model.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.IsNotFound() {
		return true
	}

	lowerErr := strings.ToLower(err.Error())

	return strings.Contains(lowerErr, "no lyrics") ||
		strings.Contains(lowerErr, "track lyrics not found") ||
		strings.Contains(lowerErr, "lyrics text is empty")
}
