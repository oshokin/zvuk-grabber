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
	// category indicates the type of download (album, playlist, audiobook, etc.).
	category DownloadCategory
	// trackIDs is the list of track IDs to download.
	trackIDs []int64
	// tracksMetadata contains track metadata mapped by track ID.
	tracksMetadata map[string]*zvuk.Track
	// albumsMetadata contains album metadata mapped by album ID.
	albumsMetadata map[string]*zvuk.Release
	// albumsTags contains tag metadata for albums.
	albumsTags map[string]map[string]string
	// labelsMetadata contains music label metadata mapped by label ID.
	labelsMetadata map[string]*zvuk.Label
	// audioCollection contains the collection structure for the download.
	audioCollection *audioCollection
	// chapterStreams contains stream metadata for audiobook chapters (only used for audiobooks).
	chapterStreams map[string]*zvuk.ChapterStreamMetadata
}

// downloadTrackRequest contains parameters for downloading a single track.
type downloadTrackRequest struct {
	// trackIndex is the position of the track in the download queue.
	trackIndex int64
	// trackID is the unique identifier of the track.
	trackID int64
	// metadata contains all metadata needed for downloading.
	metadata *downloadTracksMetadata
}

// defaultLyricsExtension is the default file extension for lyrics files.
const defaultLyricsExtension = extensionLRC

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
		releasesTags[strconv.FormatInt(album.ID, 10)] = s.fillAlbumTagsForTemplating(album)
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

// executeTrackDownload creates a download request and executes the track download.
// This is the common logic shared between sequential and concurrent downloads.
func (s *ServiceImpl) executeTrackDownload(
	ctx context.Context,
	trackIndex int,
	trackID int64,
	metadata *downloadTracksMetadata,
) {
	request := &downloadTrackRequest{
		// Track numbers start at 1 for user-facing numbering.
		trackIndex: int64(trackIndex) + 1,
		trackID:    trackID,
		metadata:   metadata,
	}

	s.downloadTrack(ctx, request)

	// Add a random pause between downloads to avoid rate limiting.
	utils.RandomPause(0, s.cfg.ParsedMaxDownloadPause)
}

// downloadTracksSequentially downloads tracks one by one (original behavior).
func (s *ServiceImpl) downloadTracksSequentially(ctx context.Context, metadata *downloadTracksMetadata) {
	for i, trackID := range metadata.trackIDs {
		// Check if context was canceled (CTRL+C pressed) - stop immediately.
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.executeTrackDownload(ctx, i, trackID, metadata)
	}
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
	for index, trackID := range metadata.trackIDs {
		// Check if context was canceled (CTRL+C pressed) - stop queueing new downloads.
		select {
		case <-ctx.Done():
			goto waitForCompletion
		default:
		}

		waitGroup.Add(1)

		go func(trackIndex int, currentTrackID int64) {
			defer waitGroup.Done()

			// Acquire semaphore slot (blocks if all workers are busy).
			semaphore <- struct{}{}

			defer func() {
				// Release semaphore slot when done.
				<-semaphore
			}()

			// Execute the track download with common logic.
			s.executeTrackDownload(ctx, trackIndex, currentTrackID, metadata)
		}(index, trackID)
	}

waitForCompletion:
	// Wait for all in-flight downloads to complete.
	waitGroup.Wait()
}

func (s *ServiceImpl) downloadTrack(
	ctx context.Context,
	req *downloadTrackRequest,
) {
	// Prepare download context.
	dc, err := s.prepareDownloadContext(ctx, req)
	if err != nil {
		return // Errors already handled in prepareDownloadContext.
	}

	// Validate track constraints BEFORE fetching stream (duration, etc.).
	// This avoids unnecessary API calls for tracks that will be skipped anyway.
	if !s.validateTrackConstraints(ctx, dc) {
		return // Validation failed, track skipped.
	}

	// Resolve quality and stream URL.
	if !s.resolveQualityAndStream(ctx, dc, req.metadata) {
		return // Errors already handled.
	}

	// Generate file paths and tags.
	s.prepareTrackFiles(ctx, dc)

	// Download and finalize.
	s.downloadAndFinalizeTrack(ctx, dc)
}

// prepareDownloadContext initializes the download context with track and collection metadata.
func (s *ServiceImpl) prepareDownloadContext(
	ctx context.Context,
	req *downloadTrackRequest,
) (*TrackDownloadContext, error) {
	metadata := req.metadata
	trackIDString := strconv.FormatInt(req.trackID, 10)

	// Retrieve track metadata.
	track, ok := metadata.tracksMetadata[trackIDString]
	if !ok || track == nil {
		err := fmt.Errorf("track with ID '%s': %w", trackIDString, ErrTrackNotFound)
		logger.Errorf(ctx, "Track with ID '%s' is not found", trackIDString)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryTrack,
			ItemID:    trackIDString,
			ItemTitle: "Unknown Track",
			Phase:     "fetching metadata",
		}, err)

		return nil, err
	}

	// Create base context.
	dc := NewTrackDownloadContext(trackIDString, track, req.trackIndex, metadata)

	// Fetch collection metadata based on category.
	switch {
	case dc.IsAudiobook:
		if !s.prepareAudiobookContext(ctx, dc, metadata) {
			return nil, ErrAudiobookContextFailed
		}
	case dc.IsPodcast:
		if !s.preparePodcastContext(ctx, dc, metadata) {
			return nil, ErrPodcastContextFailed
		}
	case !s.prepareAlbumContext(ctx, dc, metadata):
		return nil, ErrAlbumContextFailed
	}

	return dc, nil
}

// prepareAudiobookContext prepares context for audiobook chapters.
func (s *ServiceImpl) prepareAudiobookContext(
	ctx context.Context,
	dc *TrackDownloadContext,
	metadata *downloadTracksMetadata,
) bool {
	dc.AudioCollection = metadata.audioCollection
	if dc.AudioCollection == nil {
		logger.Errorf(ctx, "Audio collection wasn't found for audiobook chapter with ID '%s'", dc.TrackID)
		return false
	}

	dc.ParentTitle = dc.AudioCollection.title
	dc.ParentID = strconv.FormatInt(dc.AudioCollection.trackIDs[0], 10)

	return true
}

// preparePodcastContext prepares context for podcast episodes.
func (s *ServiceImpl) preparePodcastContext(
	ctx context.Context,
	dc *TrackDownloadContext,
	metadata *downloadTracksMetadata,
) bool {
	dc.AudioCollection = metadata.audioCollection
	if dc.AudioCollection == nil {
		logger.Errorf(ctx, "Audio collection wasn't found for podcast episode with ID '%s'", dc.TrackID)
		return false
	}

	dc.ParentTitle = dc.AudioCollection.title
	dc.ParentID = strconv.FormatInt(dc.AudioCollection.trackIDs[0], 10)

	return true
}

// prepareAlbumContext prepares context for regular tracks/albums.
func (s *ServiceImpl) prepareAlbumContext(
	ctx context.Context,
	dc *TrackDownloadContext,
	metadata *downloadTracksMetadata,
) bool {
	albumIDString := strconv.FormatInt(dc.Track.ReleaseID, 10)

	// Retrieve album metadata.
	album, ok := metadata.albumsMetadata[albumIDString]
	if !ok || album == nil {
		err := fmt.Errorf("album with ID '%s': %w", albumIDString, ErrTrackAlbumNotFound)
		logger.Errorf(ctx, "Album with ID '%s' is not found", albumIDString)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryTrack,
			ItemID:    dc.TrackID,
			ItemTitle: dc.Track.Title,
			Phase:     "fetching album metadata",
		}, err)

		return false
	}

	// Retrieve album tags.
	albumTags, ok := metadata.albumsTags[albumIDString]
	if !ok || albumTags == nil {
		logger.Errorf(ctx, "Tags for album with ID '%s' are not found", albumIDString)
		return false
	}

	// Get or create audio collection.
	audioCollection := metadata.audioCollection
	if audioCollection == nil {
		audioCollection = s.getOrRegisterAudioCollection(ctx, album, albumTags)
	}

	if audioCollection == nil {
		logger.Errorf(ctx, "Audio collection wasn't found for track with ID '%s'", dc.TrackID)
		return false
	}

	// Verify label exists.
	labelIDString := strconv.FormatInt(album.LabelID, 10)
	if _, labelExists := metadata.labelsMetadata[labelIDString]; !labelExists {
		logger.Errorf(ctx, "Label with ID '%s' is not found", labelIDString)
		return false
	}

	// Populate context.
	dc.Album = album
	dc.AlbumTags = albumTags
	dc.AudioCollection = audioCollection
	dc.ParentTitle = album.Title
	dc.ParentID = albumIDString

	return true
}

// resolveQualityAndStream resolves the quality and stream URL for the track.
func (s *ServiceImpl) resolveQualityAndStream(
	ctx context.Context,
	dc *TrackDownloadContext,
	metadata *downloadTracksMetadata,
) bool {
	errorHandler := NewErrorHandler(s)

	qualityResult, err := s.resolveTrackQuality(ctx, dc.TrackID, dc.Track, metadata)
	if err != nil {
		errorHandler.HandleError(ctx, err, &ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         dc.TrackID,
			ItemTitle:      dc.Track.Title,
			Phase:          "resolving quality",
			ParentCategory: dc.ParentCategory,
			ParentID:       dc.ParentID,
			ParentTitle:    dc.ParentTitle,
		}, true)

		return false
	}

	// Check if track should be skipped due to quality constraints.
	if qualityResult.ShouldSkip {
		errorHandler.HandleSkip(ctx, SkipReasonQuality, qualityResult.SkipReason, &ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         dc.TrackID,
			ItemTitle:      dc.Track.Title,
			Phase:          "quality check",
			ParentCategory: dc.ParentCategory,
			ParentID:       dc.ParentID,
			ParentTitle:    dc.ParentTitle,
		})

		return false
	}

	dc.Quality = qualityResult.Quality
	dc.StreamURL = qualityResult.StreamURL

	return true
}

// validateTrackConstraints validates duration and other constraints.
func (s *ServiceImpl) validateTrackConstraints(
	ctx context.Context,
	dc *TrackDownloadContext,
) bool {
	validator := NewTrackValidator(s.cfg)
	result := validator.Validate(ctx, dc.Track)

	if !result.IsValid {
		errorHandler := NewErrorHandler(s)
		errorHandler.HandleSkip(ctx, result.SkipReason, result.Error, &ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         dc.TrackID,
			ItemTitle:      dc.Track.Title,
			Phase:          "duration check",
			ParentCategory: dc.ParentCategory,
			ParentID:       dc.ParentID,
			ParentTitle:    dc.ParentTitle,
		})

		return false
	}

	return true
}

// prepareTrackFiles generates file paths and tags for the track.
func (s *ServiceImpl) prepareTrackFiles(
	ctx context.Context,
	dc *TrackDownloadContext,
) {
	// Calculate track position.
	dc.TrackPosition = dc.TrackIndex
	if !dc.IsAudiobook && !dc.IsPodcast && !dc.IsPlaylist {
		dc.TrackPosition = dc.Track.Position
	}

	// Generate track tags.
	var trackTags map[string]string
	if dc.IsPodcast {
		// For podcasts, use episode-specific tags with publication date.
		trackTags = s.fillEpisodeTags(dc.Track, dc.AudioCollection.tags, dc.TrackPosition)
	} else {
		trackTags = s.fillTrackTagsForTemplating(
			dc.TrackPosition,
			dc.Track,
			dc.AudioCollection,
			dc.AlbumTags,
			dc.IsAudiobook,
		)
	}

	// Generate filename.
	switch {
	case dc.IsAudiobook:
		dc.TrackFilename = s.templateManager.GetAudiobookChapterFilename(ctx, trackTags, dc.AudioCollection.tracksCount)
	case dc.IsPodcast:
		dc.TrackFilename = s.templateManager.GetPodcastEpisodeFilename(ctx, trackTags, dc.AudioCollection.tracksCount)
	default:
		dc.TrackFilename = s.templateManager.GetTrackFilename(
			ctx,
			dc.IsPlaylist,
			trackTags,
			dc.AudioCollection.tracksCount,
		)
	}

	dc.TrackFilename = utils.SetFileExtension(utils.SanitizeFilename(dc.TrackFilename), dc.Quality.Extension(), false)
	dc.TrackPath = filepath.Join(dc.AudioCollection.tracksPath, dc.TrackFilename)
}

// downloadAndFinalizeTrack downloads the track and writes metadata.
func (s *ServiceImpl) downloadAndFinalizeTrack(
	ctx context.Context,
	dc *TrackDownloadContext,
) {
	s.logTrackDownloadStart(ctx, dc)

	errorHandler := NewErrorHandler(s)

	// Download track.
	result, err := s.downloadAndSaveTrack(ctx, dc.StreamURL, dc.TrackPath)
	if err != nil {
		errorHandler.HandleError(ctx, err, &ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         dc.TrackID,
			ItemTitle:      dc.Track.Title,
			Phase:          "downloading file",
			ParentCategory: dc.ParentCategory,
			ParentID:       dc.ParentID,
			ParentTitle:    dc.ParentTitle,
		}, true)

		return
	}

	if result.IsExist {
		s.incrementTrackSkipped(SkipReasonExists)
		return
	}

	s.incrementTrackDownloaded(result.BytesDownloaded)

	// Write metadata and finalize assets.
	s.writeAndFinalizeTrackAssets(ctx, dc, result.TempPath)
}

// logTrackDownloadStart logs the appropriate message based on content type.
func (s *ServiceImpl) logTrackDownloadStart(ctx context.Context, dc *TrackDownloadContext) {
	switch {
	case dc.IsAudiobook:
		logger.Infof(
			ctx,
			"Downloading chapter %d of %d: %s (ID: %s, Quality: %s)",
			dc.TrackIndex,
			dc.AudioCollection.tracksCount,
			dc.Track.Title,
			dc.TrackID,
			dc.Quality)
	case dc.IsPodcast:
		logger.Infof(
			ctx,
			"Downloading episode %d of %d: %s (ID: %s, Quality: %s)",
			dc.TrackIndex,
			dc.AudioCollection.tracksCount,
			dc.Track.Title,
			dc.TrackID,
			dc.Quality)
	default:
		logger.Infof(
			ctx,
			"Downloading track %d of %d: %s (%s)",
			dc.TrackIndex,
			dc.AudioCollection.tracksCount,
			dc.Track.Title,
			dc.Quality)
	}
}

// writeAndFinalizeTrackAssets writes track metadata and finalizes covers/descriptions.
func (s *ServiceImpl) writeAndFinalizeTrackAssets(
	ctx context.Context,
	dc *TrackDownloadContext,
	tempPath string,
) {
	// Download lyrics and write metadata.
	var trackLyrics *zvuk.Lyrics

	if !dc.IsAudiobook && !dc.IsPodcast {
		trackTags := s.fillTrackTagsForTemplating(
			dc.TrackPosition,
			dc.Track,
			dc.AudioCollection,
			dc.AlbumTags,
			dc.IsAudiobook,
		)
		trackLyrics = s.downloadAndSaveLyrics(ctx, dc.Track, dc.TrackFilename, dc.AudioCollection)
		s.writeTrackMetadata(ctx, dc, trackTags, trackLyrics, tempPath)
	} else {
		// For audiobooks and podcasts, write metadata without lyrics.
		trackTags := s.fillTrackTagsForTemplating(
			dc.TrackPosition,
			dc.Track,
			dc.AudioCollection,
			dc.AlbumTags,
			dc.IsAudiobook,
		)
		s.writeTrackMetadata(ctx, dc, trackTags, nil, tempPath)
	}

	// Finalize cover art.
	s.finalizeAlbumCoverArt(ctx, dc.TrackIndex, dc.AudioCollection, dc.TrackFilename)

	// Finalize descriptions.
	if dc.IsAudiobook {
		s.finalizeAudiobookDescription(ctx, dc.TrackIndex, dc.AudioCollection, dc.TrackFilename)
	} else if dc.IsPodcast {
		s.finalizePodcastDescription(ctx, dc.TrackIndex, dc.AudioCollection, dc.TrackFilename)
	}
}

// writeTrackMetadata writes metadata tags and finalizes the file.
func (s *ServiceImpl) writeTrackMetadata(
	ctx context.Context,
	dc *TrackDownloadContext,
	trackTags map[string]string,
	trackLyrics *zvuk.Lyrics,
	tempPath string,
) {
	errorHandler := NewErrorHandler(s)

	writeTagsRequest := &WriteTagsRequest{
		TrackPath:                  tempPath,
		CoverPath:                  dc.AudioCollection.coverPath,
		Quality:                    dc.Quality,
		TrackTags:                  trackTags,
		TrackLyrics:                trackLyrics,
		IsCoverEmbeddedToTrackTags: dc.IsAudiobook || dc.IsPodcast || !dc.IsPlaylist,
	}

	// Skip in dry-run mode.
	if s.cfg.DryRun {
		return
	}

	// Write tags.
	err := s.tagProcessor.WriteTags(ctx, writeTagsRequest)
	if err != nil {
		errorHandler.HandleError(ctx, err, &ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         dc.TrackID,
			ItemTitle:      dc.Track.Title,
			Phase:          "writing metadata tags",
			ParentCategory: dc.ParentCategory,
			ParentID:       dc.ParentID,
			ParentTitle:    dc.ParentTitle,
		}, false)

		_ = os.Remove(tempPath)

		return
	}

	// Rename to final path.
	if err = os.Rename(tempPath, dc.TrackPath); err != nil {
		errorHandler.HandleError(ctx, err, &ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         dc.TrackID,
			ItemTitle:      dc.Track.Title,
			Phase:          "renaming temporary file",
			ParentCategory: dc.ParentCategory,
			ParentID:       dc.ParentID,
			ParentTitle:    dc.ParentTitle,
		}, false)

		_ = os.Remove(tempPath)
	}
}

func (s *ServiceImpl) getOrRegisterAudioCollection(
	ctx context.Context,
	album *zvuk.Release,
	albumTags map[string]string,
) *audioCollection {
	downloadItem := ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   strconv.FormatInt(album.ID, 10),
	}

	// Check if already registered (read lock).
	s.audioCollectionsMutex.Lock()
	collection, exists := s.audioCollections[downloadItem]
	s.audioCollectionsMutex.Unlock()

	if exists && collection != nil {
		return collection
	}

	// Register new collection (registerAlbumCollection handles its own locking).
	collection = s.registerAlbumCollection(ctx, album, albumTags, false)

	return collection
}

func (s *ServiceImpl) fillTrackTagsForTemplating(
	trackNumber int64,
	track *zvuk.Track,
	audioCollection *audioCollection,
	albumTags map[string]string,
	isAudiobook bool,
) map[string]string {
	// For audiobooks, use audioCollection.tags which already has all audiobook-specific fields.
	if isAudiobook {
		result := maps.Clone(audioCollection.tags)

		// Add track-specific fields.
		result["collectionTitle"] = audioCollection.title
		result["trackArtist"] = strings.Join(track.ArtistNames, ", ")
		result["trackID"] = strconv.FormatInt(track.ID, 10)
		result["trackNumber"] = strconv.FormatInt(trackNumber, 10)
		result["trackNumberPad"] = fmt.Sprintf("%0*d", trackNumberPaddingWidth, trackNumber)
		result["trackTitle"] = track.Title
		result["trackCount"] = strconv.FormatInt(audioCollection.tracksCount, 10)

		return result
	}

	// For regular tracks/albums/playlists: use albumTags.
	result := make(map[string]string, len(albumTags)+len(audioCollection.tags))
	maps.Copy(result, albumTags)

	// Apply collection tags (if it's a playlist, these will override album-specific tags).
	maps.Copy(result, audioCollection.tags)

	// Apply track-specific tags.
	result["collectionTitle"] = audioCollection.title
	result["trackArtist"] = strings.Join(track.ArtistNames, ", ")
	result["trackGenre"] = strings.Join(track.Genres, ", ")
	result["trackID"] = strconv.FormatInt(track.ID, 10)
	result["trackNumber"] = strconv.FormatInt(trackNumber, 10)
	result["trackNumberPad"] = fmt.Sprintf("%0*d", trackNumberPaddingWidth, trackNumber)
	result["trackTitle"] = track.Title
	result["trackCount"] = strconv.FormatInt(audioCollection.tracksCount, 10)

	return result
}

//nolint:cyclop,funlen,gocognit,nolintlint // Function orchestrates complex download workflow with multiple sequential steps.
func (s *ServiceImpl) downloadAndSaveTrack(
	ctx context.Context,
	trackURL string,
	trackPath string,
) (*DownloadTrackResult, error) {
	// Check if final file already exists.
	if !s.cfg.ReplaceTracks {
		if _, err := os.Stat(trackPath); err == nil {
			// In regular mode, return immediately.
			if !s.cfg.DryRun {
				logger.Infof(ctx, "Track '%s' already exists, skipping download", trackPath)

				return &DownloadTrackResult{
					IsExist:         true,
					TempPath:        "",
					BytesDownloaded: 0,
				}, nil
			}

			// In dry-run mode, just log and return (no need to fetch size).
			logger.Infof(ctx, "[DRY-RUN] Track '%s' already exists, would skip", trackPath)

			return &DownloadTrackResult{
				IsExist:         true,
				TempPath:        "",
				BytesDownloaded: 0,
			}, nil
		}
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

		return &DownloadTrackResult{
			IsExist:         false,
			TempPath:        "",
			BytesDownloaded: fetchResult.TotalBytes,
		}, nil
	}

	// Fetch the track.
	fetchResult, fetchErr := s.zvukClient.FetchTrack(ctx, trackURL)
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch track: %w", fetchErr)
	}

	defer fetchResult.Body.Close() //nolint:errcheck // Error on close is not critical here.

	// Download to temporary .part file first for atomic operation.
	// Use a local variable to avoid issues with named return values.
	tempFilePath := trackPath + ".part"

	// Always overwrite .part files (they indicate incomplete downloads).
	f, openErr := os.OpenFile(filepath.Clean(tempFilePath), overwriteFileOptions, defaultFolderPermissions)
	if openErr != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", openErr)
	}

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
		IsExist:         false,
		TempPath:        tempFilePath,
		BytesDownloaded: bytesWritten,
	}, nil
}

func (s *ServiceImpl) downloadAndSaveLyrics(
	ctx context.Context,
	track *zvuk.Track,
	trackFilename string,
	audioCollection *audioCollection,
) *zvuk.Lyrics {
	if !s.cfg.DownloadLyrics || !track.Lyrics {
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

func (s *ServiceImpl) writeLyrics(ctx context.Context, lyrics, destinationPath string) (bool, error) {
	fileOptions := overwriteFileOptions
	if !s.cfg.ReplaceLyrics {
		fileOptions = createNewFileOptions
	}

	file, err := os.OpenFile(filepath.Clean(destinationPath), fileOptions, defaultFolderPermissions)
	if err != nil {
		if os.IsExist(err) && !s.cfg.ReplaceLyrics {
			logger.Infof(ctx, "File '%s' already exists, skipping download", destinationPath)

			return true, nil
		}

		return false, err
	}

	defer file.Close()

	_, err = file.WriteString(lyrics)

	return false, err
}

//nolint:cyclop,funlen,nolintlint // Function doesn't seem to be complex.
func (s *ServiceImpl) finalizeAlbumCoverArt(
	ctx context.Context,
	trackIndex int64,
	audioCollection *audioCollection,
	trackFilename string,
) {
	// Ensure this is the last track and a valid cover exists.
	if trackIndex != audioCollection.tracksCount || audioCollection.coverPath == "" || trackFilename == "" {
		return
	}

	// Skip in dry-run mode (cover was never actually created).
	if s.cfg.DryRun {
		return
	}

	// Use temp path if available (for concurrent downloads), otherwise use regular path.
	sourceCoverPath := audioCollection.coverPath
	if audioCollection.coverTempPath != "" {
		sourceCoverPath = audioCollection.coverTempPath
	}

	coverExt := filepath.Ext(sourceCoverPath)
	if coverExt == "" {
		// Assign a default extension if none is found.
		coverExt = extensionJPG
	}

	var coverFilename string

	// For single-track albums/audiobooks without a dedicated folder, rename to match the track filename.
	if !s.cfg.CreateFolderForSingles && audioCollection.tracksCount == 1 {
		coverFilename = utils.SetFileExtension(trackFilename, coverExt, true)
	} else {
		// For multi-track or with folders, use standard name.
		coverFilename = utils.SetFileExtension(defaultCoverBasename, coverExt, false)
	}

	newCoverPath := filepath.Join(audioCollection.tracksPath, coverFilename)

	// Check if the existing cover file exists.
	originalCoverStat, err := os.Stat(sourceCoverPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Errorf(ctx, "Cover file not found: '%s'", sourceCoverPath)
		} else {
			logger.Errorf(ctx, "Unable to retrieve cover file info: %v", err)
		}

		return
	}

	// Check if the new cover file already exists and is the same as the original.
	existingCoverStat, err := os.Stat(newCoverPath)
	if err == nil && os.SameFile(originalCoverStat, existingCoverStat) {
		// No need to rename if the file is already correctly named.
		return
	}

	// Rename the cover file from temp UUID name (or original name) to final name.
	if err = os.Rename(sourceCoverPath, newCoverPath); err != nil {
		logger.Errorf(
			ctx,
			"Failed to rename cover from '%s' to '%s': %v",
			sourceCoverPath,
			newCoverPath, err)
	}
}
