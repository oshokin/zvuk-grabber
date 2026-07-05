//nolint:gocognit,err113,funlen // Yandex pipeline keeps step-by-step orchestration grouped for readability and observability.
package yandex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	yandexclient "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/retry"
	"github.com/oshokin/zvuk-grabber/internal/service/stats"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// musicClient groups Yandex API and download operations used by the service.
type musicClient interface {
	// TrackInfo fetches metadata for a single track by ID.
	TrackInfo(ctx context.Context, id string) (*model.Track, error)
	// AlbumWithTracks fetches album metadata including track volumes.
	AlbumWithTracks(ctx context.Context, id string) (*model.Album, error)
	// UsersPlaylist fetches a user-owned playlist by username and playlist ID.
	UsersPlaylist(ctx context.Context, id string, username string) (*model.Playlist, error)
	// PlaylistByUUID fetches a playlist by its UUID.
	PlaylistByUUID(ctx context.Context, id string) (*model.Playlist, error)
	// DownloadFLACBytes tries to download a real FLAC stream for a track.
	DownloadFLACBytes(ctx context.Context, trackID string) ([]byte, error)
	// ResolveMP3Link resolves a direct MP3 URL for a track.
	ResolveMP3Link(ctx context.Context, trackID string, preferredQuality uint8) (string, int, error)
	// OpenAudio opens a remote audio stream for direct file writing.
	OpenAudio(ctx context.Context, rawURL string) (*yandexclient.RemoteFile, error)
	// DownloadCoverBytes downloads cover image bytes from the given URL.
	DownloadCoverBytes(ctx context.Context, rawURL string) ([]byte, error)
	// TrackLyrics fetches lyrics metadata and text for a single track.
	TrackLyrics(ctx context.Context, id string) (*model.TrackLyrics, error)
}

// Service downloads Yandex Music URLs using shared template and tag components.
type Service interface {
	// DownloadURLs resolves and downloads tracks from the given Yandex Music URLs.
	DownloadURLs(ctx context.Context, urls []string)
	// PrintDownloadSummary logs aggregated download statistics for the session.
	PrintDownloadSummary(ctx context.Context)
}

// yandexSkipReason classifies why a track download was skipped.
type yandexSkipReason uint8

// ServiceImpl implements the Yandex Music download service.
type ServiceImpl struct {
	// cfg holds application configuration and quality preferences.
	cfg *config.Config
	// client performs Yandex Music API and download operations.
	client musicClient
	// templateManager renders output folder and filename templates.
	templateManager media.TemplateManager
	// tagProcessor writes metadata tags into downloaded audio files.
	tagProcessor media.TagProcessor

	// statsMu protects stats updates from concurrent workers.
	statsMu sync.Mutex
	// sessionStats holds running download counters for the current session.
	sessionStats *stats.Session

	// pathLocks serializes writes to the same output path.
	pathLocks *files.PathLocks

	// retryEngine handles reusable retry orchestration for Yandex API operations.
	retryEngine *retry.Engine

	// fallbackWarnings deduplicates FLAC-to-MP3 fallback warnings per collection.
	fallbackWarnings sync.Map
}

// trackJob describes one track queued for download from a Yandex Music source.
type trackJob struct {
	// kind is the collection type: album, playlist, track, audiobook, or podcast.
	kind string
	// sourceURL is the original user-provided Yandex Music URL.
	sourceURL string
	// collectionID identifies the parent album, playlist, or release.
	collectionID string
	// collectionTitle is the display title of the parent collection.
	collectionTitle string

	// track holds the Yandex track metadata to download.
	track *model.Track
	// album holds album context when the track belongs to an album release.
	album *model.Album
	// playlist holds playlist context when the track comes from a playlist.
	playlist *model.Playlist

	// trackNumber is the one-based index within the collection.
	trackNumber int
	// trackCount is the total number of tracks in the collection.
	trackCount int
}

// audioPayload holds resolved audio data or stream reference for writing.
type audioPayload struct {
	// quality is the resolved output quality tier.
	quality media.Quality
	// data contains in-memory audio bytes when pre-downloaded (e.g. FLAC).
	data []byte
	// streamURL is the direct URL for streaming MP3 downloads.
	streamURL string
	// contentLength is the expected stream or buffer size in bytes.
	contentLength int64
	// bitrate is the MP3 bitrate in Kbps when applicable.
	bitrate int
	// codec identifies the audio codec of the payload.
	codec string
}

// coverResult holds the local path to a downloaded or reused cover image.
type coverResult struct {
	// path is the filesystem path to the cover image, or empty when absent.
	path string
}

// existingTrackCheck describes a pre-download existence check for a target path.
type existingTrackCheck struct {
	// path is the output file path to check.
	path string
	// stage labels the pipeline stage performing the check.
	stage string
	// debugLabel is the structured log message for skip diagnostics.
	debugLabel string
}

const (
	// providerName is the display name used in logs and summaries.
	providerName = "Yandex Music"
	// maxCoverSize is the cover image size placeholder in Yandex cover URIs.
	maxCoverSize = "m1000x1000"
	// tempFilePattern is the suffix pattern for temporary partial download files.
	tempFilePattern = "*.part"
	// coverExtension is the file extension used for downloaded cover images.
	coverExtension = ".jpg"
	// defaultCoverName is the shared cover filename inside collection folders.
	defaultCoverName = "cover" + coverExtension
	// collectionAlbum is the tag type for standard music albums.
	collectionAlbum = "album"
	// collectionAudiobook is the tag type for audiobook releases.
	collectionAudiobook = "audiobook"
	// collectionPodcast is the tag type for podcast releases.
	collectionPodcast = "podcast"
	// collectionPlaylist is the tag type for playlist downloads.
	collectionPlaylist = "playlist"
	// collectionTrack is the tag type for single-track downloads.
	collectionTrack = "track"
	// unknownReleaseYear is the fallback release year when metadata is missing.
	unknownReleaseYear = "0000"
	// albumTypeAudiobook is the Yandex album type value for audiobooks.
	albumTypeAudiobook = "audiobook"
	// albumTypePodcast is the Yandex album type value for podcasts.
	albumTypePodcast = "podcast"
	// albumMetaTypePodcast is the Yandex album meta-type value for podcasts.
	albumMetaTypePodcast = "podcast"
)

const (
	// yandexSkipReasonUnknown means the skip reason could not be classified.
	yandexSkipReasonUnknown yandexSkipReason = iota
	// yandexSkipReasonExists means the output file already exists.
	yandexSkipReasonExists
	// yandexSkipReasonQuality means the resolved quality is below min_quality.
	yandexSkipReasonQuality
	// yandexSkipReasonDuration means the track duration is outside configured bounds.
	yandexSkipReasonDuration
)

// NewService creates a Yandex Music download service with the given dependencies.
func NewService(
	cfg *config.Config,
	client *yandexclient.Client,
	templateManager media.TemplateManager,
	tagProcessor media.TagProcessor,
) Service {
	retryEngine, err := buildYandexRetryEngine(cfg)
	if err != nil {
		retryEngine = nil
	}

	service := &ServiceImpl{
		cfg:             cfg,
		client:          client,
		templateManager: templateManager,
		tagProcessor:    tagProcessor,
		sessionStats:    stats.NewSession(time.Time{}, false),
		pathLocks:       files.NewPathLocks(),
		retryEngine:     retryEngine,
	}

	return service
}

// DownloadURLs resolves each URL into track jobs and downloads them sequentially or concurrently.
func (s *ServiceImpl) DownloadURLs(ctx context.Context, urls []string) {
	s.statsMu.Lock()
	s.sessionStats = stats.NewSession(time.Now(), s.cfg.DryRun)
	s.statsMu.Unlock()

	var (
		interrupted bool
		outputPath  = s.outputPath()
	)

	if !s.cfg.DryRun {
		if err := os.MkdirAll(outputPath, files.DefaultFolderPermissions); err != nil {
			logger.Errorf(ctx, "Failed to create output path: %v", err)
			return
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create output directory: %s", outputPath)
	}

	logger.Infof(ctx, "Starting %s download process", providerName)

	for _, rawURL := range urls {
		if ctx.Err() != nil {
			interrupted = true
			break
		}

		resolvedJobs, err := s.resolveJobs(ctx, rawURL)
		if err != nil {
			if ctx.Err() != nil {
				interrupted = true
				break
			}

			s.recordSourceFailure(rawURL, "resolving Yandex Music URL", err)
			logger.Errorf(ctx, "Failed to resolve Yandex Music URL %q: %v", rawURL, err)

			continue
		}

		if ctx.Err() != nil {
			interrupted = true
			break
		}

		s.logCollectionStart(ctx, resolvedJobs)
		s.saveCollectionDescription(ctx, resolvedJobs)
		s.downloadJobs(ctx, resolvedJobs)
	}

	if ctx.Err() != nil {
		interrupted = true
	}

	s.statsMu.Lock()
	s.sessionStats.Finish(time.Now())
	s.statsMu.Unlock()

	logger.Infof(ctx, "%s download process completed", providerName)

	if interrupted {
		logger.Debugf(ctx, "Yandex download loop interrupted: %v", ctx.Err())
	}
}

// outputPath returns the effective output folder for Yandex Music downloads.
func (s *ServiceImpl) outputPath() string {
	return s.cfg.ResolveOutputPath(config.ProviderYandex)
}

// downloadJobs runs track jobs sequentially or with bounded concurrency.
func (s *ServiceImpl) downloadJobs(ctx context.Context, jobs []*trackJob) {
	s.executeDownloadJobs(ctx, jobs, s.downloadJobWithPause)
}

// downloadJobWithPause applies the provider-level inter-track pause after each job.
func (s *ServiceImpl) downloadJobWithPause(ctx context.Context, job *trackJob) {
	s.downloadJob(ctx, job)

	if ctx.Err() != nil || s.cfg == nil {
		return
	}

	utils.RandomPause(0, s.cfg.ParsedMaxDownloadPause)
}

// executeDownloadJobs runs track jobs sequentially or concurrently using the provided handler.
func (s *ServiceImpl) executeDownloadJobs(
	ctx context.Context,
	jobs []*trackJob,
	handler func(context.Context, *trackJob),
) {
	if len(jobs) == 0 {
		return
	}

	maxConcurrent := s.cfg.MaxConcurrentDownloads
	if maxConcurrent <= 1 {
		for _, job := range jobs {
			if ctx.Err() != nil {
				return
			}

			handler(ctx, job)
		}

		return
	}

	s.downloadJobsConcurrently(ctx, jobs, maxConcurrent, handler)
}

// downloadJobsConcurrently downloads track jobs using a worker pool.
func (s *ServiceImpl) downloadJobsConcurrently(
	ctx context.Context,
	jobs []*trackJob,
	maxConcurrent int64,
	handler func(context.Context, *trackJob),
) {
	workerCount := int(maxConcurrent)
	if workerCount <= 0 {
		workerCount = 1
	}

	jobsCh := make(chan *trackJob)

	var wg sync.WaitGroup

	for range workerCount {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobsCh:
					if !ok {
						return
					}

					if ctx.Err() != nil {
						return
					}

					handler(ctx, job)
				}
			}
		})
	}

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			close(jobsCh)
			wg.Wait()

			return
		case jobsCh <- job:
		}
	}

	close(jobsCh)
	wg.Wait()
}

// downloadJob validates, resolves, and writes a single track download.
func (s *ServiceImpl) downloadJob(ctx context.Context, job *trackJob) {
	if job == nil || job.track == nil {
		s.recordFailure(errors.New("invalid empty yandex track job"))
		return
	}

	s.recordProcessed()

	validation := s.validateTrackJob(job)
	if !validation.allowed {
		s.recordSkipped(validation.reason)

		logger.Warnf(ctx, "Track validation failed: %v", validation.err)
		logger.DebugKV(
			ctx,
			"Yandex track skipped by duration",
			"kind", job.kind,
			"track_id", yandexJobTrackID(job),
			"track_title", job.track.FullTitle(),
			"duration", time.Duration(job.track.DurationMs)*time.Millisecond,
			"min_duration", s.cfg.ParsedMinDuration,
			"max_duration", s.cfg.ParsedMaxDuration,
		)

		return
	}

	tags := s.buildTags(job)

	preferredTargetPath := s.buildTargetPath(ctx, job, tags, preferredQuality(s.cfg.Quality))
	if strings.TrimSpace(preferredTargetPath) == "" {
		s.recordTrackFailure(
			job,
			"building preferred target path",
			fmt.Errorf("failed to build target path for track %s", yandexJobTrackID(job)),
		)

		return
	}

	if s.cfg.DryRun {
		s.recordDownloaded(0)
		logger.Infof(ctx, "[DRY-RUN] Would download track to: %s", preferredTargetPath)
		logger.DebugKV(
			ctx,
			"Yandex dry-run track download planned",
			"kind", job.kind,
			"track_id", yandexJobTrackID(job),
			"track_title", job.track.FullTitle(),
			"track_number", job.trackNumber,
			"track_count", job.trackCount,
		)

		return
	}

	if s.skipExistingYandexTrack(ctx, job, &existingTrackCheck{
		path:       preferredTargetPath,
		stage:      "pre-resolve",
		debugLabel: "Yandex existing track skipped before audio resolve",
	}) {
		return
	}

	// FLAC may fallback to MP3; check MP3 path early to avoid unnecessary resolve/download calls.
	if preferredQuality(s.cfg.Quality) == media.QualityFLAC {
		mp3FallbackPath := s.buildTargetPath(ctx, job, tags, media.QualityMP3High)
		if strings.TrimSpace(mp3FallbackPath) != "" && mp3FallbackPath != preferredTargetPath {
			if s.skipExistingYandexTrack(ctx, job, &existingTrackCheck{
				path:       mp3FallbackPath,
				stage:      "fallback",
				debugLabel: "Yandex existing fallback track skipped before audio resolve",
			}) {
				return
			}
		}
	}

	result, err := s.resolveAudioQuality(ctx, job)
	if err != nil {
		s.recordTrackFailure(job, "resolving audio", err)
		logger.Errorf(ctx, "Failed to resolve audio for Yandex track %q: %v", job.track.FullTitle(), err)

		return
	}

	if result.shouldSkip {
		s.recordSkipped(yandexSkipReasonQuality)
		minQuality := media.FromConfig(s.cfg.MinQuality).Description()
		logger.Warnf(ctx, "%s quality is below minimum threshold %s, skipping", job.track.FullTitle(), minQuality)
		logger.DebugKV(
			ctx,
			"Yandex track skipped by quality",
			"kind", job.kind,
			"track_id", yandexJobTrackID(job),
			"track_title", job.track.FullTitle(),
			"min_quality", minQuality,
			"reason", result.skipReason,
		)

		return
	}

	payload := result.payload
	if payload == nil {
		s.recordTrackFailure(
			job,
			"resolving audio payload",
			fmt.Errorf("empty audio payload for track %s", yandexJobTrackID(job)),
		)

		return
	}

	targetPath := s.buildTargetPath(ctx, job, tags, payload.quality)
	if strings.TrimSpace(targetPath) == "" {
		s.recordTrackFailure(
			job,
			"building target path",
			fmt.Errorf("failed to build target path for track %s", yandexJobTrackID(job)),
		)

		return
	}

	unlock := s.pathLocks.Lock(targetPath)
	defer unlock()

	if s.skipExistingYandexTrack(ctx, job, &existingTrackCheck{
		path:       targetPath,
		stage:      "target",
		debugLabel: "Yandex existing track skipped",
	}) {
		return
	}

	if err = os.MkdirAll(filepath.Dir(targetPath), files.DefaultFolderPermissions); err != nil {
		s.recordTrackFailure(job, "creating target folder", err)
		return
	}

	lyrics := s.downloadAndSaveLyrics(ctx, targetPath, job)

	s.logDownloadStart(ctx, job, targetPath, payload)

	written, err := s.writeAudioFile(ctx, targetPath, payload, job, tags, lyrics)
	if err != nil {
		s.recordTrackFailure(job, "writing audio file", err)
		logger.Errorf(ctx, "Failed to write Yandex track %q: %v", job.track.FullTitle(), err)

		return
	}

	s.recordDownloaded(written)
	s.logDownloadDone(ctx, job, targetPath, payload, written)
}

// skipExistingYandexTrack skips download when the target file already exists.
func (s *ServiceImpl) skipExistingYandexTrack(ctx context.Context, job *trackJob, check *existingTrackCheck) bool {
	if s.cfg.ReplaceTracks || job == nil || job.track == nil || check == nil || strings.TrimSpace(check.path) == "" {
		return false
	}

	exists, err := utils.IsFileExist(check.path)
	if err != nil {
		logger.Debugf(
			ctx,
			"Yandex %s file check failed: kind=%s track_id=%s path=%s err=%v",
			check.stage,
			job.kind,
			yandexJobTrackID(job),
			check.path,
			err,
		)

		return false
	}

	if !exists {
		return false
	}

	s.recordSkipped(yandexSkipReasonExists)
	logger.Infof(ctx, "Track '%s' already exists, skipping download", check.path)
	logger.DebugKV(
		ctx,
		check.debugLabel,
		"kind", job.kind,
		"track_id", yandexJobTrackID(job),
		"track_title", job.track.FullTitle(),
		"path", check.path,
	)

	return true
}
