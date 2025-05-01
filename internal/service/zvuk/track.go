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
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"
)

const defaultLyricsExtension = ".lrc"

type fetchAlbumsDataFromTracksResponse struct {
	releases     map[string]*zvuk.Release
	releasesTags map[string]map[string]string
	labels       map[string]*zvuk.Label
}

type downloadTracksMetadata struct {
	category        DownloadCategory
	trackIDs        []int64
	tracksMetadata  map[string]*zvuk.Track
	albumsMetadata  map[string]*zvuk.Release
	albumsTags      map[string]map[string]string
	labelsMetadata  map[string]*zvuk.Label
	audioCollection *audioCollection
}

type downloadTrackRequest struct {
	trackIndex int64
	trackID    int64
	metadata   *downloadTracksMetadata
}

func (s *ServiceImpl) fetchAlbumsDataFromTracks(
	ctx context.Context,
	tracks map[string]*zvuk.Track,
) (*fetchAlbumsDataFromTracksResponse, error) {
	// Collect unique album IDs from the tracks
	uniqueAlbumIDs := make(map[int64]struct{}, len(tracks))
	for _, track := range tracks {
		uniqueAlbumIDs[track.ReleaseID] = struct{}{}
	}

	// Convert album IDs to strings for API request
	albumIDs := utils.MapIterator(maps.Keys(uniqueAlbumIDs),
		func(v int64) string {
			return strconv.FormatInt(v, 10)
		})

	// Fetch album metadata
	albumsMetadataResponse, err := s.zvukClient.GetAlbumsMetadata(ctx, albumIDs, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get albums metadata: %w", err)
	}

	// Generate tags for each album
	releasesTags := make(map[string]map[string]string, len(albumsMetadataResponse.Releases))
	for _, album := range albumsMetadataResponse.Releases {
		releasesTags[strconv.FormatInt(album.ID, 10)] = s.fillAlbumTagsForTemplating(album)
	}

	// Collect label IDs from the albums
	labelIDs := utils.MapIterator(maps.Values(albumsMetadataResponse.Releases),
		func(v *zvuk.Release) string {
			if v == nil {
				return ""
			}

			return strconv.FormatInt(v.LabelID, 10)
		})

	// Fetch label metadata
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
	for i, trackID := range metadata.trackIDs {
		request := &downloadTrackRequest{
			// Track numbers start at 1 for user-facing numbering
			trackIndex: int64(i) + 1,
			trackID:    trackID,
			metadata:   metadata,
		}

		s.downloadTrack(ctx, request)

		// Add a random pause between downloads to avoid rate limiting
		utils.RandomPause(0, s.cfg.ParsedMaxDownloadPause)
	}
}

//nolint:funlen // Function orchestrates complex download workflow with multiple sequential steps
func (s *ServiceImpl) downloadTrack(
	ctx context.Context,
	req *downloadTrackRequest,
) {
	metadata := req.metadata
	// Retrieve track metadata
	trackIDString := strconv.FormatInt(req.trackID, 10)

	track, ok := metadata.tracksMetadata[trackIDString]
	if !ok || track == nil {
		logger.Errorf(ctx, "Track with ID '%s' is not found", trackIDString)

		return
	}

	// Retrieve album metadata
	albumIDString := strconv.FormatInt(track.ReleaseID, 10)

	album, ok := metadata.albumsMetadata[albumIDString]
	if !ok || album == nil {
		logger.Errorf(ctx, "Album with ID '%s' is not found", albumIDString)

		return
	}

	// Retrieve album tags
	albumTags, ok := metadata.albumsTags[albumIDString]
	if !ok || albumTags == nil {
		logger.Errorf(ctx, "Tags for album with ID '%s' are not found", albumIDString)

		return
	}

	// If separate tracks are being downloaded, we must create folders for albums
	audioCollection := metadata.audioCollection
	if audioCollection == nil {
		audioCollection = s.getOrRegisterAudioCollection(ctx, album, albumTags)
	}

	// If audio collection is not found, return
	if audioCollection == nil {
		logger.Errorf(ctx, "Audio collection wasn't found for track with ID '%s'", trackIDString)

		return
	}

	// Retrieve label metadata
	labelIDString := strconv.FormatInt(album.LabelID, 10)

	label, ok := metadata.labelsMetadata[labelIDString]
	if !ok || label == nil {
		logger.Errorf(ctx, "Label with ID '%s' is not found", labelIDString)

		return
	}

	// Determine track quality
	quality := TrackQuality(s.cfg.DownloadFormat)

	highestQuality := s.getTrackHighestQuality(ctx, track)
	if highestQuality < quality {
		quality = highestQuality

		logger.Infof(ctx, "Track is only available in quality: %s", highestQuality)
	}

	// Fetch track streaming metadata
	streamMetadata, err := s.zvukClient.GetStreamMetadata(ctx, trackIDString, quality.AsStreamURLParameterValue())
	if err != nil {
		logger.Errorf(ctx, "Failed to get track streaming metadata: %v", err)

		return
	}

	streamURL := streamMetadata.Stream
	quality = s.defineQualityByStreamURL(streamURL)

	// Determine the track's position in the album or playlist
	isPlaylist := metadata.category == DownloadCategoryPlaylist

	trackPosition := req.trackIndex
	if !isPlaylist {
		// For album downloads, use the track's position in the album metadata
		// For playlists, use the track's position in the playlist
		trackPosition = track.Position
	}

	// Generate track filename with proper extension
	trackTags := s.fillTrackTagsForTemplating(trackPosition, track, label.Title, audioCollection, albumTags)
	trackFilename := s.templateManager.GetTrackFilename(ctx, isPlaylist, trackTags, audioCollection.tracksCount)
	trackFilename = utils.SetFileExtension(utils.SanitizeFilename(trackFilename), quality.Extension(), false)
	trackPath := filepath.Join(audioCollection.tracksPath, trackFilename)

	// Download and save the track
	logger.Infof(
		ctx,
		"Downloading track %d of %d: %s (%s)",
		req.trackIndex,
		audioCollection.tracksCount,
		track.Title,
		quality)

	isExist, err := s.downloadAndSaveTrack(ctx, streamURL, trackPath)
	if err != nil {
		logger.Errorf(ctx, "Failed to download track: %v", err)

		return
	}

	if isExist {
		return
	}

	// Download and save track lyrics if available
	trackLyrics := s.downloadAndSaveLyrics(ctx, track, trackFilename, audioCollection)

	writeTagsRequest := &WriteTagsRequest{
		TrackPath:                  trackPath,
		CoverPath:                  audioCollection.coverPath,
		Quality:                    quality,
		TrackTags:                  trackTags,
		TrackLyrics:                trackLyrics,
		IsCoverEmbeddedToTrackTags: !isPlaylist,
	}

	// Write metadata tags to track
	err = s.tagProcessor.WriteTags(ctx, writeTagsRequest)
	if err != nil {
		logger.Errorf(ctx, "Failed to write track tags: %v", err)

		return
	}

	// Handle album cover art finalization
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
	// Initialize result map with album tags first
	result := make(map[string]string, len(albumTags)+len(audioCollection.tags))
	maps.Copy(result, albumTags)

	// Apply collection tags (if it's a playlist, these will override album-specific tags)
	maps.Copy(result, audioCollection.tags)

	// Apply track-specific tags
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

func (s *ServiceImpl) downloadAndSaveTrack(ctx context.Context, trackURL, trackPath string) (bool, error) {
	// Determine file creation flags
	fileOptions := overwriteFileOptions
	if !s.cfg.ReplaceTracks {
		fileOptions = createNewFileOptions
	}

	// Fetch the track
	body, totalBytes, err := s.zvukClient.FetchTrack(ctx, trackURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch track: %w", err)
	}

	defer body.Close()

	// Attempt to open file
	f, err := os.OpenFile(filepath.Clean(trackPath), fileOptions, defaultFilePermissions)
	if err != nil {
		if os.IsExist(err) && !s.cfg.ReplaceTracks {
			logger.Infof(ctx, "Track '%s' already exists, skipping download", trackPath)

			return true, nil
		}

		return false, fmt.Errorf("failed to create file: %w", err)
	}

	defer f.Close()

	// Initialize progress tracker
	var writer io.Writer

	if logger.Level() <= zap.InfoLevel {
		bar := progressbar.DefaultBytes(
			totalBytes,
			"Downloading",
		)

		writer = io.MultiWriter(f, bar)
	} else {
		writer = f
	}

	// Download logic
	if s.cfg.ParsedDownloadSpeedLimit == 0 {
		_, err = io.Copy(writer, body)
	} else {
		for {
			_, err = io.CopyN(writer, body, s.cfg.ParsedDownloadSpeedLimit)
			if errors.Is(err, io.EOF) {
				err = nil

				break
			}

			if err != nil {
				break
			}

			// Throttle to respect speed limit
			time.Sleep(time.Second)
		}
	}

	if err != nil {
		return false, fmt.Errorf("failed to write file: %w", err)
	}

	return false, nil
}

func (s *ServiceImpl) downloadAndSaveLyrics(
	ctx context.Context,
	track *zvuk.Track,
	trackFilename string,
	audioCollection *audioCollection) *zvuk.Lyrics {
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

	isLyricsExist, err := s.writeLyrics(ctx, lyrics.Lyrics, lyricsPath)
	if err != nil {
		logger.Errorf(ctx, "Failed to write lyrics: %v", err)
	}

	if !isLyricsExist && err == nil {
		logger.Infof(ctx, "Lyrics saved to file: %s", lyricsPath)
	}

	return lyrics
}

func (s *ServiceImpl) writeLyrics(ctx context.Context, lyrics, destinationPath string) (bool, error) {
	fileOptions := overwriteFileOptions
	if !s.cfg.ReplaceLyrics {
		fileOptions = createNewFileOptions
	}

	file, err := os.OpenFile(filepath.Clean(destinationPath), fileOptions, defaultFilePermissions)
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

func (s *ServiceImpl) finalizeAlbumCoverArt(
	ctx context.Context,
	trackIndex int64,
	audioCollection *audioCollection,
	trackFilename string,
) {
	// Ensure this is the last track and a valid cover exists
	if trackIndex != audioCollection.tracksCount || audioCollection.coverPath == "" || trackFilename == "" {
		return
	}

	coverExt := filepath.Ext(audioCollection.coverPath)
	if coverExt == "" {
		// Assign a default extension if none is found
		coverExt = defaultAlbumCoverExtension
	}

	coverFilename := utils.SetFileExtension(defaultAlbumCoverBasename, coverExt, false)

	// For single-track albums without a dedicated folder, rename the cover to match the track filename
	if !s.cfg.CreateFolderForSingles && audioCollection.tracksCount == 1 {
		coverFilename = utils.SetFileExtension(trackFilename, coverExt, true)
	}

	newCoverPath := filepath.Join(audioCollection.tracksPath, coverFilename)

	// Check if the existing cover file exists
	originalCoverStat, err := os.Stat(audioCollection.coverPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Errorf(ctx, "Album cover file not found: '%s'", audioCollection.coverPath)
		} else {
			logger.Errorf(ctx, "Unable to retrieve album cover file info: %v", err)
		}

		return
	}

	// Check if the new cover file already exists and is the same as the original
	existingCoverStat, err := os.Stat(newCoverPath)
	if err == nil && os.SameFile(originalCoverStat, existingCoverStat) {
		// No need to rename if the file is already correctly named
		return
	}

	// Rename the cover file to the new location
	if err = os.Rename(audioCollection.coverPath, newCoverPath); err != nil {
		logger.Errorf(
			ctx,
			"Failed to rename album cover from '%s' to '%s': %v",
			audioCollection.coverPath,
			newCoverPath, err)
	}
}
