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
	// category indicates the type of download (album, playlist, etc.).
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
const defaultLyricsExtension = ".lrc"

var (
	// ErrTrackNotFound indicates that the requested track was not found.
	ErrTrackNotFound = errors.New("track not found")
	// ErrTrackAlbumNotFound indicates that the album for the track was not found.
	ErrTrackAlbumNotFound = errors.New("track album not found")
	// ErrIncompleteDownload indicates that the downloaded file size doesn't match expected size.
	ErrIncompleteDownload = errors.New("incomplete download")
	// ErrQualityBelowThreshold indicates that track quality is below the configured minimum.
	ErrQualityBelowThreshold = errors.New("quality below minimum threshold")
	// ErrDurationBelowThreshold indicates that track duration is below the configured minimum.
	ErrDurationBelowThreshold = errors.New("duration below minimum threshold")
	// ErrDurationAboveThreshold indicates that track duration exceeds the configured maximum.
	ErrDurationAboveThreshold = errors.New("duration above maximum threshold")
)

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

//nolint:funlen,gocognit,cyclop // Function orchestrates complex download workflow with multiple sequential steps.
func (s *ServiceImpl) downloadTrack(
	ctx context.Context,
	req *downloadTrackRequest,
) {
	metadata := req.metadata
	// Retrieve track metadata.
	trackIDString := strconv.FormatInt(req.trackID, 10)

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

		return
	}

	// Retrieve album metadata.
	albumIDString := strconv.FormatInt(track.ReleaseID, 10)

	album, ok := metadata.albumsMetadata[albumIDString]
	if !ok || album == nil {
		err := fmt.Errorf("album with ID '%s': %w", albumIDString, ErrTrackAlbumNotFound)
		logger.Errorf(ctx, "Album with ID '%s' is not found", albumIDString)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryTrack,
			ItemID:    trackIDString,
			ItemTitle: track.Title,
			Phase:     "fetching album metadata",
		}, err)

		return
	}

	// Retrieve album tags.
	albumTags, ok := metadata.albumsTags[albumIDString]
	if !ok || albumTags == nil {
		logger.Errorf(ctx, "Tags for album with ID '%s' are not found", albumIDString)

		return
	}

	// If separate tracks are being downloaded, we must create folders for albums.
	audioCollection := metadata.audioCollection
	if audioCollection == nil {
		audioCollection = s.getOrRegisterAudioCollection(ctx, album, albumTags)
	}

	// If audio collection is not found, return.
	if audioCollection == nil {
		logger.Errorf(ctx, "Audio collection wasn't found for track with ID '%s'", trackIDString)

		return
	}

	// Retrieve label metadata.
	labelIDString := strconv.FormatInt(album.LabelID, 10)

	label, ok := metadata.labelsMetadata[labelIDString]
	if !ok || label == nil {
		logger.Errorf(ctx, "Label with ID '%s' is not found", labelIDString)

		return
	}

	// Determine track quality.
	quality := TrackQuality(s.cfg.Quality)

	highestQuality := s.getTrackHighestQuality(ctx, track)
	if highestQuality < quality {
		quality = highestQuality

		logger.Infof(ctx, "Track is only available in quality: %s", highestQuality)
	}

	// Check minimum quality threshold if set.
	if s.cfg.MinQuality > 0 && quality < TrackQuality(s.cfg.MinQuality) {
		logger.Warnf(ctx, "Track quality %s is below minimum threshold %s, skipping",
			quality, TrackQuality(s.cfg.MinQuality))

		s.incrementTrackSkipped(SkipReasonQuality)

		s.recordError(&ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         trackIDString,
			ItemTitle:      track.Title,
			Phase:          "quality check",
			ParentCategory: metadata.category,
			ParentID:       albumIDString,
			ParentTitle:    album.Title,
		}, fmt.Errorf("%w: %s below %s",
			ErrQualityBelowThreshold, quality, TrackQuality(s.cfg.MinQuality)))

		return
	}

	// Check minimum duration threshold if set.
	if s.cfg.ParsedMinDuration > 0 && time.Duration(track.Duration)*time.Second < s.cfg.ParsedMinDuration {
		logger.Warnf(ctx, "Track duration %ds is below minimum threshold %s, skipping",
			track.Duration, s.cfg.ParsedMinDuration)

		s.incrementTrackSkipped(SkipReasonDuration)

		s.recordError(&ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         trackIDString,
			ItemTitle:      track.Title,
			Phase:          "duration check",
			ParentCategory: metadata.category,
			ParentID:       albumIDString,
			ParentTitle:    album.Title,
		}, fmt.Errorf("%w: %ds below %s",
			ErrDurationBelowThreshold, track.Duration, s.cfg.ParsedMinDuration))

		return
	}

	// Check maximum duration threshold if set.
	if s.cfg.ParsedMaxDuration > 0 && time.Duration(track.Duration)*time.Second > s.cfg.ParsedMaxDuration {
		logger.Warnf(ctx, "Track duration %ds exceeds maximum threshold %s, skipping",
			track.Duration, s.cfg.ParsedMaxDuration)

		s.incrementTrackSkipped(SkipReasonDuration)

		s.recordError(&ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         trackIDString,
			ItemTitle:      track.Title,
			Phase:          "duration check",
			ParentCategory: metadata.category,
			ParentID:       albumIDString,
			ParentTitle:    album.Title,
		}, fmt.Errorf("%w: %ds exceeds %s",
			ErrDurationAboveThreshold, track.Duration, s.cfg.ParsedMaxDuration))

		return
	}

	// Fetch track streaming metadata.
	streamMetadata, err := s.zvukClient.GetStreamMetadata(ctx, trackIDString, quality.AsStreamURLParameterValue())
	if err != nil {
		// Don't log context cancellation - it's expected when user presses CTRL+C.
		if !errors.Is(err, context.Canceled) {
			logger.Errorf(ctx, "Failed to get track streaming metadata: %v", err)
		}

		s.recordError(&ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         trackIDString,
			ItemTitle:      track.Title,
			Phase:          "fetching stream metadata",
			ParentCategory: metadata.category,
			ParentID:       albumIDString,
			ParentTitle:    album.Title,
		}, err)

		return
	}

	streamURL := streamMetadata.Stream
	quality = s.defineQualityByStreamURL(streamURL)

	// Determine the track's position in the album or playlist.
	isPlaylist := metadata.category == DownloadCategoryPlaylist

	trackPosition := req.trackIndex
	if !isPlaylist {
		// For album downloads, use the track's position in the album metadata.
		// For playlists, use the track's position in the playlist.
		trackPosition = track.Position
	}

	// Generate track filename with proper extension.
	trackTags := s.fillTrackTagsForTemplating(trackPosition, track, label.Title, audioCollection, albumTags)
	trackFilename := s.templateManager.GetTrackFilename(ctx, isPlaylist, trackTags, audioCollection.tracksCount)
	trackFilename = utils.SetFileExtension(utils.SanitizeFilename(trackFilename), quality.Extension(), true)
	trackPath := filepath.Join(audioCollection.tracksPath, trackFilename)

	// Download and save the track.
	logger.Infof(
		ctx,
		"Downloading track %d of %d: %s (%s)",
		req.trackIndex,
		audioCollection.tracksCount,
		track.Title,
		quality)

	result, err := s.downloadAndSaveTrack(ctx, streamURL, trackPath)
	if err != nil {
		// Don't log context cancellation - it's expected when user presses CTRL+C.
		if !errors.Is(err, context.Canceled) {
			logger.Errorf(ctx, "Failed to download track: %v", err)
		}

		s.incrementTrackFailed()
		s.recordError(&ErrorContext{
			Category:       DownloadCategoryTrack,
			ItemID:         trackIDString,
			ItemTitle:      track.Title,
			Phase:          "downloading file",
			ParentCategory: metadata.category,
			ParentID:       albumIDString,
			ParentTitle:    album.Title,
		}, err)

		return
	}

	if result.IsExist {
		s.incrementTrackSkipped(SkipReasonExists)

		return
	}

	s.incrementTrackDownloaded(result.BytesDownloaded)

	// Download and save track lyrics if available.
	trackLyrics := s.downloadAndSaveLyrics(ctx, track, trackFilename, audioCollection)

	// Write metadata tags to .part file BEFORE renaming for atomic operation.
	writeTagsRequest := &WriteTagsRequest{
		TrackPath:                  result.TempPath, // Write to .part file.
		CoverPath:                  audioCollection.coverPath,
		Quality:                    quality,
		TrackTags:                  trackTags,
		TrackLyrics:                trackLyrics,
		IsCoverEmbeddedToTrackTags: !isPlaylist,
	}

	// Skip tag writing and file operations in dry-run mode.
	if !s.cfg.DryRun {
		err = s.tagProcessor.WriteTags(ctx, writeTagsRequest)
		if err != nil {
			logger.Errorf(ctx, "Failed to write track tags: %v", err)
			s.recordError(&ErrorContext{
				Category:       DownloadCategoryTrack,
				ItemID:         trackIDString,
				ItemTitle:      track.Title,
				Phase:          "writing metadata tags",
				ParentCategory: metadata.category,
				ParentID:       albumIDString,
				ParentTitle:    album.Title,
			}, err)

			// Clean up .part file on tagging failure.
			_ = os.Remove(result.TempPath)

			return
		}

		// Atomically rename .part file to final name.
		// At this point, the file has complete audio data AND metadata tags.
		if err = os.Rename(result.TempPath, trackPath); err != nil {
			logger.Errorf(ctx, "Failed to finalize track file: %v", err)
			s.recordError(&ErrorContext{
				Category:       DownloadCategoryTrack,
				ItemID:         trackIDString,
				ItemTitle:      track.Title,
				Phase:          "renaming temporary file",
				ParentCategory: metadata.category,
				ParentID:       albumIDString,
				ParentTitle:    album.Title,
			}, err)

			// Clean up .part file on rename failure.
			_ = os.Remove(result.TempPath)

			return
		}
	}

	// Handle album cover art finalization.
	s.finalizeAlbumCoverArt(ctx, req.trackIndex, audioCollection, trackFilename)
}

func (s *ServiceImpl) getOrRegisterAudioCollection(
	ctx context.Context,
	album *zvuk.Release,
	albumTags map[string]string,
) *audioCollection {
	s.audioCollectionsMutex.Lock()

	downloadItem := ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   strconv.FormatInt(album.ID, 10),
	}
	collection, exists := s.audioCollections[downloadItem]

	s.audioCollectionsMutex.Unlock()

	if !exists || collection == nil {
		collection = s.registerAlbumCollection(ctx, album, albumTags, false)
	}

	return collection
}

func (s *ServiceImpl) fillTrackTagsForTemplating(
	trackNumber int64,
	track *zvuk.Track,
	label string,
	audioCollection *audioCollection,
	albumTags map[string]string,
) map[string]string {
	// Initialize result map with album tags first.
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
	result["trackNumberPad"] = fmt.Sprintf("%02d", trackNumber)
	result["trackTitle"] = track.Title
	result["recordLabel"] = label
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
	f, openErr := os.OpenFile(filepath.Clean(tempFilePath), overwriteFileOptions, constants.DefaultFolderPermissions)
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

	file, err := os.OpenFile(filepath.Clean(destinationPath), fileOptions, constants.DefaultFolderPermissions)
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

	coverExt := filepath.Ext(audioCollection.coverPath)
	if coverExt == "" {
		// Assign a default extension if none is found.
		coverExt = defaultAlbumCoverExtension
	}

	coverFilename := utils.SetFileExtension(defaultAlbumCoverBasename, coverExt, false)

	// For single-track albums without a dedicated folder, rename the cover to match the track filename.
	if !s.cfg.CreateFolderForSingles && audioCollection.tracksCount == 1 {
		coverFilename = utils.SetFileExtension(trackFilename, coverExt, true)
	}

	newCoverPath := filepath.Join(audioCollection.tracksPath, coverFilename)

	// Check if the existing cover file exists.
	originalCoverStat, err := os.Stat(audioCollection.coverPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Errorf(ctx, "Album cover file not found: '%s'", audioCollection.coverPath)
		} else {
			logger.Errorf(ctx, "Unable to retrieve album cover file info: %v", err)
		}

		return
	}

	// Check if the new cover file already exists and is the same as the original.
	existingCoverStat, err := os.Stat(newCoverPath)
	if err == nil && os.SameFile(originalCoverStat, existingCoverStat) {
		// No need to rename if the file is already correctly named.
		return
	}

	// Rename the cover file to the new location.
	if err = os.Rename(audioCollection.coverPath, newCoverPath); err != nil {
		logger.Errorf(
			ctx,
			"Failed to rename album cover from '%s' to '%s': %v",
			audioCollection.coverPath,
			newCoverPath, err)
	}
}
