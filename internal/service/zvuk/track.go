package zvuk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/constants"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// fetchAlbumsDataFromTracksResponse contains album data fetched from tracks.
type fetchAlbumsDataFromTracksResponse struct {
	// releases contains release metadata mapped by release ID.
	releases map[string]*zvuk.Release
	// releasesTags contains tag metadata for releases.
	releasesTags map[string]map[string]string
	// labels contains music label metadata mapped by label ID.
	labels map[string]*zvuk.Label
}

// downloadTracksMetadata contains all metadata needed for downloading tracks.
type downloadTracksMetadata struct {
	// audioCollection contains the collection structure for the download.
	// It's nil when downloading tracks individually, so duplication of category and trackIDs is necessary.
	audioCollection *audioCollection
	// category indicates the type of download (album, playlist, audiobook, podcast, track, etc.).
	category DownloadCategory
	// trackIDs is the list of track IDs to download.
	trackIDs []int64
	// tracksMetadata contains track metadata mapped by track ID.
	tracksMetadata map[string]*zvuk.Track
	// albumsMetadata contains album metadata mapped by album ID.
	albumsMetadata map[string]*zvuk.Release
	// albumsTags contains tag metadata for albums.
	albumsTags map[string]map[string]string
	// chapterStreamsMetadata contains stream metadata for audiobook chapters.
	chapterStreamsMetadata map[string]*zvuk.StreamQualities
	// labelsMetadata contains music label metadata mapped by label ID.
	labelsMetadata map[string]*zvuk.Label
}

// downloadTrackTask is a task for downloading a single track.
type downloadTrackTask struct {
	// trackIndex is the 1-based position of this track in the download batch.
	trackIndex int64
	// trackIDString is the string representation of the track ID.
	trackIDString string
	// track contains the fetched track metadata.
	track *zvuk.Track
	// trackPosition is the resolved display position within the collection.
	trackPosition int64
	// trackFilename is the output filename for the downloaded track.
	trackFilename string
	// trackPath is the full output path for the downloaded track.
	trackPath string
	// quality is the resolved audio quality for this download.
	quality TrackQuality
	// streamURL is the remote URL to download the track audio from.
	streamURL string
	// albumTags contains album-level metadata tags for the track.
	albumTags map[string]string
	// trackTags contains track-level metadata tags for tagging and templates.
	trackTags map[string]string
	// parentID is the ID of the parent collection (album, playlist, etc.).
	parentID string
	// parentTitle is the title of the parent collection.
	parentTitle string
	// audioCollection is the resolved collection context for this track.
	// For albums/playlists/audiobooks/podcasts it's shared across all tracks via metadata.
	// For standalone track downloads it is derived from the track's album.
	audioCollection *audioCollection
	// metadata contains the shared download context for the current batch.
	metadata *downloadTracksMetadata
}

// defaultLyricsExtension is the default file extension for lyrics files.
const defaultLyricsExtension = extensionLRC

// downloadError builds a DownloadError for this track task with the given phase.
func (t *downloadTrackTask) downloadError(phase string, err error) *DownloadError {
	return &DownloadError{
		Category:       DownloadCategoryTrack,
		ItemID:         t.trackIDString,
		ItemTitle:      t.track.Title,
		ParentCategory: t.metadata.category,
		ParentID:       t.parentID,
		ParentTitle:    t.parentTitle,
		Phase:          phase,
		Error:          err,
	}
}

// fetchAlbumsDataFromTracks fetches album and label data for a list of tracks.
func (s *ServiceImpl) fetchAlbumsDataFromTracks(
	ctx context.Context,
	tracks map[string]*zvuk.Track,
) (*fetchAlbumsDataFromTracksResponse, error) {
	// Collect unique album IDs from the tracks.
	uniqueAlbumIDs := make(map[int64]struct{}, len(tracks))
	for _, track := range tracks {
		uniqueAlbumIDs[track.ReleaseID] = struct{}{}
	}

	// Convert album IDs to strings for API request.
	albumIDs := utils.MapIterator(maps.Keys(uniqueAlbumIDs),
		func(v int64) string {
			return strconv.FormatInt(v, 10)
		})

	// Fetch album metadata.
	albumsMetadataResponse, err := s.zvukClient.GetAlbumsMetadata(ctx, albumIDs, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get albums metadata: %w", err)
	}

	// Generate tags for each album.
	releasesTags := make(map[string]map[string]string, len(albumsMetadataResponse.Releases))
	for _, album := range albumsMetadataResponse.Releases {
		releasesTags[strconv.FormatInt(album.ID, 10)] = s.albumHandler.FillTags(album)
	}

	// Collect label IDs from the albums.
	labelIDs := utils.MapIterator(maps.Values(albumsMetadataResponse.Releases),
		func(v *zvuk.Release) string {
			if v == nil {
				return ""
			}

			return strconv.FormatInt(v.LabelID, 10)
		})

	// Fetch label metadata.
	labelsMetadata, err := s.zvukClient.GetLabelsMetadata(ctx, labelIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get labels metadata: %w", err)
	}

	return &fetchAlbumsDataFromTracksResponse{
		releases:     albumsMetadataResponse.Releases,
		releasesTags: releasesTags,
		labels:       labelsMetadata,
	}, nil
}

// downloadTracks downloads a list of tracks, either sequentially or concurrently.
func (s *ServiceImpl) downloadTracks(ctx context.Context, metadata *downloadTracksMetadata) {
	maxConcurrent := s.cfg.MaxConcurrentDownloads

	// Sequential download (default behavior when maxConcurrent == 1).
	if maxConcurrent == 1 {
		s.downloadTracksSequentially(ctx, metadata)

		return
	}

	// Concurrent downloads with worker pool pattern.
	s.downloadTracksConcurrently(ctx, metadata, maxConcurrent)
}

// downloadTracksSequentially downloads tracks one by one (original behavior).
func (s *ServiceImpl) downloadTracksSequentially(ctx context.Context, metadata *downloadTracksMetadata) {
	for i, trackID := range metadata.trackIDs {
		// Stop between tracks on CTRL+C, but still finalize shared assets.
		if ctx.Err() != nil {
			break
		}

		s.downloadSingleTrack(ctx, i, trackID, metadata)
	}

	s.finalizeCollectionAssets(ctx, metadata)
}

// downloadTracksConcurrently downloads tracks using a worker pool for concurrent execution.
func (s *ServiceImpl) downloadTracksConcurrently(
	ctx context.Context,
	metadata *downloadTracksMetadata,
	maxConcurrent int64,
) {
	// Create a semaphore channel to limit concurrent downloads.
	semaphore := make(chan struct{}, maxConcurrent)

	var waitGroup sync.WaitGroup

	// Process each track in a separate goroutine.
queueTracks:
	for index, trackID := range metadata.trackIDs {
		// Check if context was canceled (CTRL+C pressed) - stop queueing new downloads.
		select {
		case <-ctx.Done():
			break queueTracks
		default:
		}

		waitGroup.Add(1)

		go func(trackIndex int, currentTrackID int64) {
			defer waitGroup.Done()

			// Acquire semaphore slot or stop immediately on cancellation.
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

			defer func() {
				// Release semaphore slot when done.
				<-semaphore
			}()

			// Avoid starting new work when cancellation arrives while waiting for a slot.
			if ctx.Err() != nil {
				return
			}

			s.downloadSingleTrack(ctx, trackIndex, currentTrackID, metadata)
		}(index, trackID)
	}

	// Wait for all in-flight downloads to complete.
	waitGroup.Wait()

	s.finalizeCollectionAssets(ctx, metadata)
}

// finalizeCollectionAssets finalizes cover and description files after all tracks complete.
func (s *ServiceImpl) finalizeCollectionAssets(ctx context.Context, metadata *downloadTracksMetadata) {
	if metadata == nil || metadata.audioCollection == nil {
		return
	}

	s.finalizeCover(ctx, metadata.audioCollection.tracksCount, metadata.audioCollection)
	s.finalizeDescription(ctx, metadata.audioCollection, metadata.audioCollection.tracksCount)
}

// downloadSingleTrack downloads one track by index and ID within the given metadata context.
func (s *ServiceImpl) downloadSingleTrack(
	ctx context.Context,
	trackIndex int,
	trackID int64,
	metadata *downloadTracksMetadata,
) {
	// Stop as early as possible on CTRL+C.
	if ctx.Err() != nil {
		return
	}

	// Create new download track task.
	task, err := s.newDownloadTrackTask(ctx, trackIndex, trackID, metadata)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	// Download track.
	s.downloadTrack(ctx, task)

	if ctx.Err() != nil {
		return
	}

	// Random pause.
	utils.RandomPause(0, s.cfg.ParsedMaxDownloadPause)
}

// newDownloadTrackTask builds a download task for the given track index and ID.
func (s *ServiceImpl) newDownloadTrackTask(
	ctx context.Context,
	trackIndex int,
	trackID int64,
	metadata *downloadTracksMetadata,
) (*downloadTrackTask, error) {
	// Create new download track task.
	t := &downloadTrackTask{
		trackIndex:      int64(trackIndex) + 1,
		trackIDString:   strconv.FormatInt(trackID, 10),
		audioCollection: metadata.audioCollection,
		metadata:        metadata,
	}

	// Fetch track metadata.
	track, ok := t.metadata.tracksMetadata[t.trackIDString]
	if !ok || track == nil {
		err := fmt.Errorf("track with ID '%s': %w", t.trackIDString, ErrTrackNotFound)
		logger.Errorf(ctx, "Track with ID '%s' is not found", t.trackIDString)
		s.recordError(&DownloadError{
			Category:  DownloadCategoryTrack,
			ItemID:    t.trackIDString,
			ItemTitle: "Unknown Track",
			Phase:     "fetching metadata",
			Error:     err,
		})

		return nil, err
	}

	t.track = track

	if t.metadata.category.IsChapterCollection() {
		// I did it for easy logging, since tracks metadata doesn't have audio collection.
		if t.audioCollection == nil {
			return nil, ErrAlbumContextFailed
		}

		t.parentID = t.audioCollection.id
		t.parentTitle = t.audioCollection.title
	} else if !s.prepareRegularTrackTask(ctx, t) {
		return nil, ErrAlbumContextFailed
	}

	return t, nil
}

// downloadTrack downloads a single track.
func (s *ServiceImpl) downloadTrack(
	ctx context.Context,
	task *downloadTrackTask,
) {
	if ctx.Err() != nil {
		return
	}

	// Validate track constraints before fetching stream (duration, etc.).
	// This avoids unnecessary API calls for tracks that will be skipped.
	if !s.validateTrackConstraints(ctx, task) {
		return // Validation failed, track skipped.
	}

	if ctx.Err() != nil {
		return
	}

	// Resolve quality and stream URL.
	if !s.resolveQualityAndStream(ctx, task) {
		return // Errors already handled.
	}

	if ctx.Err() != nil {
		return
	}

	// Generate file paths and tags.
	s.prepareTrackFiles(ctx, task)

	if ctx.Err() != nil {
		return
	}

	// Download and finalize.
	s.downloadAndFinalizeTrack(ctx, task)
}

// prepareRegularTrackTask prepares track context for regular tracks/albums/playlists.
func (s *ServiceImpl) prepareRegularTrackTask(
	ctx context.Context,
	t *downloadTrackTask,
) bool {
	albumIDString := strconv.FormatInt(t.track.ReleaseID, 10)

	// Retrieve album metadata.
	album, ok := t.metadata.albumsMetadata[albumIDString]
	if !ok || album == nil {
		err := fmt.Errorf("album with ID '%s': %w", albumIDString, ErrTrackAlbumNotFound)
		logger.Errorf(ctx, "Album with ID '%s' is not found", albumIDString)
		s.recordError(&DownloadError{
			Category:  DownloadCategoryTrack,
			ItemID:    t.trackIDString,
			ItemTitle: t.track.Title,
			Phase:     "fetching album metadata",
			Error:     err,
		})

		return false
	}

	// Retrieve album tags.
	albumTags, ok := t.metadata.albumsTags[albumIDString]
	if !ok || albumTags == nil {
		logger.Errorf(ctx, "Tags for album with ID '%s' are not found", albumIDString)
		return false
	}

	// Get or create audio collection.
	audioCollection := t.metadata.audioCollection
	if audioCollection == nil {
		audioCollection = s.getOrRegisterAudioCollection(ctx, album, t.metadata.tracksMetadata)
	}

	if audioCollection == nil {
		logger.Errorf(ctx, "Audio collection not found for track with ID '%s'", t.trackIDString)
		return false
	}

	t.audioCollection = audioCollection

	// Verify label exists.
	labelIDString := strconv.FormatInt(album.LabelID, 10)
	if _, labelExists := t.metadata.labelsMetadata[labelIDString]; !labelExists {
		logger.Errorf(ctx, "Label with ID '%s' is not found", labelIDString)
		return false
	}

	// Populate task fields.
	t.parentID = albumIDString
	t.parentTitle = album.Title
	t.albumTags = albumTags

	return true
}

// resolveQualityAndStream resolves the quality and stream URL for the track.
func (s *ServiceImpl) resolveQualityAndStream(
	ctx context.Context,
	t *downloadTrackTask,
) bool {
	qualityResult, err := s.resolveTrackQuality(ctx, t.trackIDString, t.track, t.metadata)
	if err != nil {
		s.handleError(ctx, t.downloadError("resolving quality", err), true)

		return false
	}

	// Check if track should be skipped due to quality constraints.
	if qualityResult.ShouldSkip {
		s.handleTrackSkipped(SkipReasonQuality, t.downloadError("quality check", qualityResult.SkipReason))

		return false
	}

	t.quality = qualityResult.Quality
	t.streamURL = qualityResult.StreamURL

	return true
}

// validateTrackConstraints validates duration and other constraints.
func (s *ServiceImpl) validateTrackConstraints(
	ctx context.Context,
	task *downloadTrackTask,
) bool {
	result := s.validator.Validate(ctx, task.track)

	if !result.IsValid {
		s.handleTrackSkipped(result.SkipReason, task.downloadError("duration check", result.Error))

		return false
	}

	return true
}

// prepareTrackFiles generates file paths and tags for the track.
func (s *ServiceImpl) prepareTrackFiles(
	ctx context.Context,
	task *downloadTrackTask,
) {
	// Calculate track position.
	task.trackPosition = resolveTrackPosition(task)

	task.trackTags = buildTrackTags(&trackTagContext{
		trackNumber:     task.trackPosition,
		track:           task.track,
		audioCollection: task.audioCollection,
		albumTags:       task.albumTags,
		category:        task.metadata.category,
	})

	// Generate filename.
	switch task.metadata.category {
	case DownloadCategoryAudiobook:
		task.trackFilename = s.templateManager.GetAudiobookChapterFilename(
			ctx,
			task.trackTags,
			task.audioCollection.tracksCount,
		)
	case DownloadCategoryPodcast:
		task.trackFilename = s.templateManager.GetPodcastEpisodeFilename(
			ctx,
			task.trackTags,
			task.audioCollection.tracksCount,
		)
	default:
		task.trackFilename = s.templateManager.GetTrackFilename(
			ctx,
			task.metadata.category == DownloadCategoryPlaylist,
			task.trackTags,
			task.audioCollection.tracksCount,
		)
	}

	task.trackFilename = utils.SetFileExtension(
		utils.SanitizeFilename(task.trackFilename),
		task.quality.Extension(),
		false,
	)

	// tracksPath is normally populated when a collection is registered.
	// Some unit tests construct metadata manually without setting it, so fall back
	// to the configured output path to keep the downloader robust.
	basePath := ""
	if task.audioCollection != nil {
		basePath = task.audioCollection.tracksPath
	}

	if basePath == "" {
		basePath = s.cfg.OutputPath
	}

	if basePath == "" {
		basePath = "." // last-resort fallback to avoid writing to an empty path
	}

	task.trackPath = filepath.Join(basePath, task.trackFilename)
}

// resolveTrackPosition determines the display position for a track within its collection.
func resolveTrackPosition(task *downloadTrackTask) int64 {
	if task == nil {
		return 0
	}

	if task.track == nil || task.metadata == nil {
		return task.trackIndex
	}

	switch task.metadata.category {
	case DownloadCategoryAlbum, DownloadCategoryTrack:
		if task.track.Position > 0 {
			return task.track.Position
		}
	}

	return task.trackIndex
}

// downloadAndFinalizeTrack downloads the track and writes metadata.
func (s *ServiceImpl) downloadAndFinalizeTrack(
	ctx context.Context,
	task *downloadTrackTask,
) {
	unlockPath := s.lockPath(task.trackPath)
	defer unlockPath()

	logger.Infof(
		ctx,
		"Downloading %s %d of %d: %s (ID: %s, Quality: %s)",
		task.metadata.category.ToSubcategory(),
		task.trackIndex,
		task.audioCollection.tracksCount,
		task.track.Title,
		task.trackIDString,
		task.quality.String(),
	)

	// Download track.
	result, err := s.downloadAndSaveTrack(ctx, task.streamURL, task.trackPath)
	if err != nil {
		s.handleError(ctx, task.downloadError("downloading file", err), true)

		return
	}

	if result.IsExist {
		s.incrementTrackSkipped(SkipReasonExists)
		return
	}

	s.incrementTrackDownloaded(result.BytesDownloaded)

	trackLyrics := s.downloadAndSaveLyrics(ctx, task.track, task.trackFilename, task.audioCollection)
	s.writeTrackMetadata(ctx, task, task.trackTags, trackLyrics, result.TempPath)
}

// writeTrackMetadata writes metadata tags and finalizes the file.
func (s *ServiceImpl) writeTrackMetadata(
	ctx context.Context,
	t *downloadTrackTask,
	trackTags map[string]string,
	trackLyrics *zvuk.Lyrics,
	tempPath string,
) {
	var coverPath string
	if t.audioCollection != nil {
		for _, candidate := range []string{
			strings.TrimSpace(t.audioCollection.embeddableCoverPath),
			strings.TrimSpace(t.audioCollection.coverPath),
		} {
			if candidate == "" {
				continue
			}

			if _, err := os.Stat(candidate); err == nil {
				coverPath = candidate
				break
			}
		}
	}

	writeTagsRequest := &WriteTagsRequest{
		TrackPath:                  tempPath,
		CoverPath:                  coverPath,
		Quality:                    t.quality,
		TrackTags:                  trackTags,
		TrackLyrics:                trackLyrics,
		IsCoverEmbeddedToTrackTags: t.metadata.category != DownloadCategoryPlaylist,
	}

	// Skip in dry-run mode.
	if s.cfg.DryRun {
		return
	}

	// Write tags.
	err := s.tagProcessor.WriteTags(ctx, writeTagsRequest)
	if err != nil {
		s.handleError(ctx, t.downloadError("writing metadata tags", err), false)

		_ = os.Remove(tempPath)

		return
	}

	// Rename to final path.
	if err = utils.RenameFile(tempPath, t.trackPath, s.cfg.ReplaceTracks); err != nil {
		if errors.Is(err, os.ErrExist) && !s.cfg.ReplaceTracks {
			logger.Infof(ctx, "Track '%s' already exists, skipping download", t.trackPath)
			s.incrementTrackSkipped(SkipReasonExists)

			_ = os.Remove(tempPath)

			return
		}

		s.handleError(ctx, t.downloadError("renaming temporary file", err), false)

		_ = os.Remove(tempPath)
	}
}

// getOrRegisterAudioCollection gets or registers an audio collection.
func (s *ServiceImpl) getOrRegisterAudioCollection(
	ctx context.Context,
	album *zvuk.Release,
	tracksMetadata map[string]*zvuk.Track,
) *audioCollection {
	albumID := strconv.FormatInt(album.ID, 10)
	downloadItem := ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   albumID,
	}

	// Check if already registered (read lock).
	s.audioCollectionsMutex.Lock()
	collection, exists := s.audioCollections[downloadItem]
	s.audioCollectionsMutex.Unlock()

	if exists && collection != nil {
		return collection
	}

	// Register new collection (registerCollection handles its own locking).
	albumItems := map[string]*zvuk.Release{albumID: album}

	return registerCollection(ctx, s, albumID, albumItems, tracksMetadata, false, s.albumHandler)
}

// skipExistingTrack returns true when an existing final track file should not be replaced.
func (s *ServiceImpl) skipExistingTrack(ctx context.Context, trackPath string) bool {
	if s.cfg.ReplaceTracks {
		return false
	}

	if _, err := os.Stat(trackPath); err != nil {
		return false
	}

	if s.cfg.DryRun {
		logger.Infof(ctx, "[DRY-RUN] Track '%s' already exists, would skip", trackPath)

		return true
	}

	logger.Infof(ctx, "Track '%s' already exists, skipping download", trackPath)

	return true
}

// downloadAndSaveTrack downloads and saves a track to a file.
//
//nolint:funlen,gocognit // Function orchestrates complex download workflow with multiple sequential steps.
func (s *ServiceImpl) downloadAndSaveTrack(
	ctx context.Context,
	trackURL string,
	trackPath string,
) (*DownloadTrackResult, error) {
	// Check if final file already exists.
	if s.skipExistingTrack(ctx, trackPath) {
		return &DownloadTrackResult{IsExist: true}, nil
	}

	// Dry-run mode: simulate download without fetching actual data.
	if s.cfg.DryRun {
		logger.Infof(ctx, "[DRY-RUN] Would download track to: %s", trackPath)

		// Fetch metadata to get file size estimate.
		fetchResult, fetchErr := s.zvukClient.FetchTrack(ctx, trackURL)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch track metadata: %w", fetchErr)
		}

		// Close immediately without reading.
		_ = fetchResult.Body.Close()

		return &DownloadTrackResult{BytesDownloaded: fetchResult.TotalBytes}, nil
	}

	// Fetch the track.
	fetchResult, fetchErr := s.zvukClient.FetchTrack(ctx, trackURL)
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch track: %w", fetchErr)
	}

	defer fetchResult.Body.Close()

	// Download to temporary .part file first for atomic operation.
	// Use unique temp files per worker to avoid concurrent write collisions.
	tmpFile, openErr := os.CreateTemp(filepath.Dir(trackPath), filepath.Base(trackPath)+".part-*")
	if openErr != nil {
		return nil, fmt.Errorf("failed to create temporary track file: %w", openErr)
	}

	tempFilePath := tmpFile.Name()
	if chmodErr := tmpFile.Chmod(constants.DefaultFilePermissions); chmodErr != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tempFilePath)

		return nil, fmt.Errorf("failed to set temporary file permissions: %w", chmodErr)
	}

	f := tmpFile

	// Track whether download succeeded.
	// If not, we'll clean up the .part file on function exit.
	var downloadSucceeded bool

	defer func() {
		// Ensure file is closed before cleanup.
		closeErr := f.Close()

		// Clean up .part file if download failed.
		if !downloadSucceeded {
			// Small delay to ensure file handle is released (Windows needs this).
			time.Sleep(10 * time.Millisecond)

			if removeErr := os.Remove(tempFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				// Log warning but don't fail - this is best-effort cleanup.
				logger.Warnf(ctx, "Failed to clean up temporary file '%s': %v (close error: %v)",
					tempFilePath, removeErr, closeErr)
			}
		}
	}()

	// Initialize progress tracker.
	// Progress bars are disabled when downloading concurrently to avoid terminal output conflicts.
	var writer io.Writer

	if logger.Level() <= zap.InfoLevel && s.cfg.MaxConcurrentDownloads == 1 {
		bar := progressbar.DefaultBytes(
			fetchResult.TotalBytes,
			"Downloading",
		)

		writer = io.MultiWriter(f, bar)
	} else {
		writer = f
	}

	// Download logic.
	var (
		bytesWritten int64
		err          error
	)

	if s.cfg.ParsedDownloadSpeedLimit == 0 {
		bytesWritten, err = io.Copy(writer, fetchResult.Body)
	} else {
		for {
			var n int64

			n, err = io.CopyN(writer, fetchResult.Body, s.cfg.ParsedDownloadSpeedLimit)
			bytesWritten += n

			if errors.Is(err, io.EOF) {
				err = nil

				break
			}

			if err != nil {
				break
			}

			// Throttle to respect speed limit.
			time.Sleep(time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Verify that we downloaded the expected number of bytes.
	if bytesWritten != fetchResult.TotalBytes {
		return nil, fmt.Errorf(
			"%w: wrote %d bytes, expected %d bytes",
			ErrIncompleteDownload,
			bytesWritten,
			fetchResult.TotalBytes,
		)
	}

	// Mark download as successful to prevent cleanup by defer.
	// The .part file will be renamed to final name by the caller after tags are written.
	downloadSucceeded = true

	// Return the temp file path for the caller to rename after writing tags.
	return &DownloadTrackResult{
		TempPath:        tempFilePath,
		BytesDownloaded: bytesWritten,
	}, nil
}

// downloadAndSaveLyrics downloads and saves lyrics for a track.
func (s *ServiceImpl) downloadAndSaveLyrics(
	ctx context.Context,
	track *zvuk.Track,
	trackFilename string,
	audioCollection *audioCollection,
) *zvuk.Lyrics {
	if !s.cfg.DownloadLyrics ||
		!track.Lyrics ||
		audioCollection.category.IsChapterCollection() {
		return nil
	}

	logger.Infof(ctx, "Downloading lyrics for track: %s\n", track.Title)

	trackID := strconv.FormatInt(track.ID, 10)

	lyrics, err := s.zvukClient.GetTrackLyrics(ctx, trackID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get lyrics: %v", err)

		return nil
	}

	lyricsContent := strings.TrimSpace(lyrics.Lyrics)
	if lyricsContent == "" {
		logger.Info(ctx, "Lyrics is empty")

		return nil
	}

	lyricsPath := filepath.Join(
		audioCollection.tracksPath,
		utils.SetFileExtension(trackFilename, defaultLyricsExtension, true))

	// Dry-run mode: simulate lyrics download.
	if s.cfg.DryRun {
		// Check if lyrics file exists.
		if _, statErr := os.Stat(lyricsPath); statErr == nil && !s.cfg.ReplaceLyrics {
			logger.Infof(ctx, "[DRY-RUN] Lyrics '%s' already exists, would skip", lyricsPath)
			s.incrementLyricsSkipped()
		} else {
			logger.Infof(ctx, "[DRY-RUN] Would save lyrics to: %s", lyricsPath)
			s.incrementLyricsDownloaded()
		}

		return lyrics
	}

	isLyricsExist, err := s.writeLyrics(ctx, lyrics.Lyrics, lyricsPath)
	if err != nil {
		logger.Errorf(ctx, "Failed to write lyrics: %v", err)

		return nil
	}

	if isLyricsExist {
		s.incrementLyricsSkipped()
	} else {
		s.incrementLyricsDownloaded()
		logger.Infof(ctx, "Lyrics saved to file: %s", lyricsPath)
	}

	return lyrics
}

// writeLyrics writes lyrics to a file.
func (s *ServiceImpl) writeLyrics(ctx context.Context, lyrics, destinationPath string) (bool, error) {
	destinationPath = filepath.Clean(destinationPath)

	if !s.cfg.ReplaceLyrics {
		if _, err := os.Stat(destinationPath); err == nil {
			logger.Infof(ctx, "File '%s' already exists, skipping download", destinationPath)

			return true, nil
		}
	}

	// Write into a temp file in the same folder, then atomically rename to destination.
	// This avoids partially-written .lrc files on crashes/interruption.
	tmpFile, err := os.CreateTemp(filepath.Dir(destinationPath), filepath.Base(destinationPath)+".tmp-*")
	if err != nil {
		return false, err
	}

	tmpPath := tmpFile.Name()

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if chmodErr := os.Chmod(tmpPath, constants.DefaultFilePermissions); chmodErr != nil {
		return false, chmodErr
	}

	_, err = tmpFile.WriteString(lyrics)
	if err != nil {
		return false, err
	}

	if err = tmpFile.Close(); err != nil {
		return false, err
	}

	if err = utils.RenameFile(tmpPath, destinationPath, s.cfg.ReplaceLyrics); err != nil {
		if errors.Is(err, os.ErrExist) && !s.cfg.ReplaceLyrics {
			logger.Infof(ctx, "File '%s' already exists, skipping download", destinationPath)

			return true, nil
		}

		return false, err
	}

	return false, nil
}
