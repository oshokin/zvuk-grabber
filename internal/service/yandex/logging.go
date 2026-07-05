package yandex

import (
	"context"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// logDownloadStart logs user-facing and debug download start messages for a track.
func (s *ServiceImpl) logDownloadStart(ctx context.Context, job *trackJob, targetPath string, payload *audioPayload) {
	if payload == nil {
		return
	}

	trackTitle := yandexTrackTitle(job)

	logger.Infof(
		ctx,
		"Downloading %s %d of %d: %s (ID: %s, Quality: %s)",
		job.kind,
		job.trackNumber,
		job.trackCount,
		trackTitle,
		yandexJobTrackID(job),
		qualityDescriptionForLog(payload),
	)

	logger.Debugf(
		ctx,
		"Yandex track download started: kind=%s track_id=%s collection=%q quality=%s bitrate_kbps=%d path=%s",
		job.kind,
		yandexJobTrackID(job),
		job.collectionTitle,
		qualityTokenForLog(payload),
		payload.bitrate,
		targetPath,
	)
}

// logDownloadDone logs user-facing and debug download completion messages for a track.
func (s *ServiceImpl) logDownloadDone(
	ctx context.Context,
	job *trackJob,
	targetPath string,
	payload *audioPayload,
	bytes int64,
) {
	if payload == nil {
		return
	}

	trackTitle := yandexTrackTitle(job)

	size := uint64(0)
	if bytes > 0 {
		size = uint64(bytes)
	}

	logger.Infof(
		ctx,
		"Downloaded %s %d of %d: %s (%s)",
		job.kind,
		job.trackNumber,
		job.trackCount,
		trackTitle,
		humanize.IBytes(size),
	)

	logger.Debugf(
		ctx,
		"Yandex track download completed: kind=%s track_id=%s collection=%q quality=%s bitrate_kbps=%d size=%s path=%s codec=%s",
		job.kind,
		yandexJobTrackID(job),
		job.collectionTitle,
		qualityTokenForLog(payload),
		payload.bitrate,
		humanize.IBytes(size),
		targetPath,
		media.ParseCodec(payload.codec).String(),
	)
}

// logCollectionStart logs a summary line when downloading multi-track collections.
func (s *ServiceImpl) logCollectionStart(ctx context.Context, jobs []*trackJob) {
	if len(jobs) == 0 {
		return
	}

	first := jobs[0]
	if first == nil || first.kind == collectionTrack {
		return
	}

	tags := s.buildTags(first)

	switch first.kind {
	case collectionAlbum:
		logger.Infof(
			ctx,
			"Downloading %s: %s - %s (%s)",
			first.kind,
			tags[media.TagAlbumArtist],
			tags[media.TagAlbumTitle],
			tags[media.TagReleaseYear],
		)
	case collectionPlaylist:
		logger.Infof(ctx, "Downloading %s: %s", yandexCollectionTitle(first.kind), first.collectionTitle)
	case collectionAudiobook:
		logger.Infof(
			ctx,
			"Downloading %s: %s by %s",
			yandexCollectionTitle(first.kind),
			tags[media.TagAudiobookTitle],
			tags[media.TagAudiobookAuthors],
		)
	case collectionPodcast:
		logger.Infof(
			ctx,
			"Downloading %s: %s by %s",
			yandexCollectionTitle(first.kind),
			tags[media.TagPodcastTitle],
			tags[media.TagPodcastAuthors],
		)
	default:
		logger.Infof(
			ctx,
			"Downloading %s: %s",
			yandexCollectionTitle(first.kind),
			first.collectionTitle,
		)
	}
}

// warnLosslessFallbackOnce logs a FLAC fallback warning once per collection and reason.
func (s *ServiceImpl) warnLosslessFallbackOnce(
	ctx context.Context,
	job *trackJob,
	reason error,
) {
	if job == nil || job.track == nil {
		logger.Warnf(ctx, "Yandex FLAC is unavailable: %v. Falling back to MP3 when allowed by min_quality.", reason)
		return
	}

	if job.collectionID == "" || job.trackCount <= 1 {
		logger.Warnf(
			ctx,
			"Yandex FLAC is unavailable for track %q: %v. Falling back to MP3 when allowed by min_quality.",
			job.track.FullTitle(),
			reason,
		)

		return
	}

	key := fmt.Sprintf("%s:%s:%v", job.kind, job.collectionID, reason)
	if _, loaded := s.fallbackWarnings.LoadOrStore(key, struct{}{}); loaded {
		logger.Debugf(
			ctx,
			"Yandex FLAC fallback already reported: kind=%s collection=%q reason=%v",
			job.kind,
			job.collectionTitle,
			reason,
		)

		return
	}

	logger.Warnf(
		ctx,
		"Yandex FLAC is unavailable for %s %q: %v. Falling back to MP3 when allowed by min_quality.",
		job.kind,
		job.collectionTitle,
		reason,
	)
}

// yandexCollectionTitle returns a human-readable title for a collection kind.
func yandexCollectionTitle(kind string) string {
	switch kind {
	case collectionAlbum:
		return "Album"
	case collectionPlaylist:
		return "Playlist"
	case collectionAudiobook:
		return "Audiobook"
	case collectionPodcast:
		return "Podcast"
	default:
		trimmed := strings.TrimSpace(kind)
		if trimmed == "" {
			return "Collection"
		}

		return strings.ToUpper(trimmed[:1]) + strings.ToLower(trimmed[1:])
	}
}

// yandexTrackTitle returns a display title for a download job track.
func yandexTrackTitle(job *trackJob) string {
	if job == nil || job.track == nil {
		return ""
	}

	if title := strings.TrimSpace(job.track.Title); title != "" {
		return title
	}

	return job.track.FullTitle()
}

// qualityDescriptionForLog returns a human-readable quality label for info logs.
func qualityDescriptionForLog(payload *audioPayload) string {
	if label, ok := mp3QualityLabel(payload, "MP3, %d Kbps"); ok {
		return label
	}

	if payload == nil {
		return media.QualityUnknown.Description()
	}

	return payload.quality.Description()
}

// qualityTokenForLog returns a compact quality token for debug logs.
func qualityTokenForLog(payload *audioPayload) string {
	if label, ok := mp3QualityLabel(payload, "mp3_%d"); ok {
		return label
	}

	if payload == nil {
		return media.QualityUnknown.String()
	}

	return payload.quality.String()
}

// mp3QualityLabel formats the actual MP3 bitrate when it is known.
func mp3QualityLabel(payload *audioPayload, format string) (string, bool) {
	if payload == nil || !strings.EqualFold(payload.codec, media.CodecMP3.String()) || payload.bitrate <= 0 {
		return "", false
	}

	return fmt.Sprintf(format, payload.bitrate), true
}
