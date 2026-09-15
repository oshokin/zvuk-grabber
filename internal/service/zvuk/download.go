package zvuk

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// downloadCollection downloads any type of collection based on item.Category.
//
//nolint:funlen,gocognit // It handles a lot of logic, so it's complex.
func (s *ServiceImpl) downloadCollection(ctx context.Context, item *DownloadItem) {
	// Fetch metadata based on category.
	var (
		itemID          = item.ItemID
		category        = item.Category
		tracksMetadata  map[string]*zvuk.Track
		audioCollection *audioCollection
		streamsMetadata map[string]*zvuk.StreamQualities
		albumsMetadata  map[string]*zvuk.Release
		albumsTags      map[string]map[string]string
		labelsMetadata  map[string]*zvuk.Label
		err             error
	)

	switch category {
	case DownloadCategoryAlbum:
		itemData, fetchErr := s.fetchAlbumData(ctx, itemID)
		if fetchErr != nil {
			s.recordCollectionError(item, "fetching "+category.String()+" data", fetchErr)
			return
		}

		audioCollection = registerCollection(ctx, s, itemID, itemData.releases, itemData.tracks, true, s.albumHandler)
		if audioCollection == nil {
			return
		}

		tracksMetadata = itemData.tracks
		albumsMetadata = itemData.releases
		albumsTags = map[string]map[string]string{itemID: audioCollection.tags}
		labelsMetadata = itemData.labels

	case DownloadCategoryPlaylist:
		getPlaylistsMetadataResponse, fetchErr := s.zvukClient.GetPlaylistsMetadata(ctx, []string{itemID})
		if fetchErr != nil {
			s.recordCollectionError(item, "fetching "+category.String()+" metadata", fetchErr)
			return
		}

		if playlist, ok := getPlaylistsMetadataResponse.Playlists[itemID]; ok && playlist != nil {
			tracks, tracksErr := s.zvukClient.GetTracksMetadata(
				ctx,
				utils.Map(playlist.TrackIDs, func(trackID int64) string {
					return strconv.FormatInt(trackID, 10)
				}),
			)
			if tracksErr != nil {
				s.recordCollectionError(item, "fetching "+category.String()+" track metadata", tracksErr)
				return
			}

			getPlaylistsMetadataResponse.Tracks = tracks
		}

		itemData, fetchErr := s.fetchAlbumsDataFromTracks(ctx, getPlaylistsMetadataResponse.Tracks)
		if fetchErr != nil {
			playlistTitle := "Playlist ID: " + itemID
			if playlist, ok := getPlaylistsMetadataResponse.Playlists[itemID]; ok && playlist != nil {
				playlistTitle = playlist.Title
			}

			s.recordError(&DownloadError{
				Category:  category,
				ItemID:    itemID,
				ItemTitle: playlistTitle,
				ItemURL:   item.URL,
				Phase:     "fetching track metadata",
				Error:     fetchErr,
			})

			return
		}

		audioCollection = registerCollection(
			ctx,
			s,
			itemID,
			getPlaylistsMetadataResponse.Playlists,
			getPlaylistsMetadataResponse.Tracks,
			true,
			s.playlistHandler,
		)
		if audioCollection == nil {
			return
		}

		tracksMetadata = getPlaylistsMetadataResponse.Tracks
		albumsMetadata = itemData.releases
		albumsTags = itemData.releasesTags
		labelsMetadata = itemData.labels

	case DownloadCategoryAudiobook:
		itemData, fetchErr := s.zvukClient.GetAudiobooksMetadata(ctx, []string{itemID})
		if fetchErr != nil {
			s.recordCollectionError(item, "fetching "+category.String()+" metadata", fetchErr)
			return
		}

		audioCollection = registerCollection(
			ctx,
			s,
			itemID,
			itemData.Audiobooks,
			itemData.Tracks,
			true,
			s.audiobookHandler,
		)
		if audioCollection == nil {
			return
		}

		tracksMetadata = itemData.Tracks
	case DownloadCategoryPodcast:
		itemData, fetchErr := s.zvukClient.GetPodcastsMetadata(ctx, []string{itemID})
		if fetchErr != nil {
			s.recordCollectionError(item, "fetching "+category.String()+" metadata", fetchErr)
			return
		}

		audioCollection = registerCollection(ctx, s, itemID, itemData.Podcasts, itemData.Tracks, true, s.podcastHandler)
		if audioCollection == nil {
			return
		}

		tracksMetadata = itemData.Tracks
	default:
		logger.Errorf(ctx, "Unknown collection category: %v", category)
		return
	}

	if category.IsChapterCollection() {
		// Sort tracks by position when metadata is available.
		// Keep original track order if metadata is unavailable (e.g., cache-only hits).
		if len(tracksMetadata) > 0 {
			audioCollection.trackIDs = s.getTracksSortedByPosition(tracksMetadata, audioCollection.trackIDs)
		}

		trackIDs := utils.Map(audioCollection.trackIDs, func(trackID int64) string {
			return strconv.FormatInt(trackID, 10)
		})

		streamsMetadata, err = s.zvukClient.GetStreamQualities(ctx, trackIDs)
		if err != nil {
			s.recordCollectionStreamError(item, audioCollection.title, err)
			return
		}
	}

	// Prepare unified metadata for downloading tracks.
	metadata := &downloadTracksMetadata{
		audioCollection:        audioCollection,
		category:               category,
		trackIDs:               audioCollection.trackIDs,
		tracksMetadata:         tracksMetadata,
		albumsMetadata:         albumsMetadata,
		albumsTags:             albumsTags,
		chapterStreamsMetadata: streamsMetadata,
		labelsMetadata:         labelsMetadata,
	}

	// Download all tracks using unified pipeline.
	s.downloadTracks(ctx, metadata)
}

// recordCollectionError records a download error for a collection item.
func (s *ServiceImpl) recordCollectionError(item *DownloadItem, phase string, err error) {
	s.recordError(&DownloadError{
		Category:  item.Category,
		ItemID:    item.ItemID,
		ItemTitle: item.Category.ToTitleCase() + " ID: " + item.ItemID,
		ItemURL:   item.URL,
		Phase:     phase,
		Error:     err,
	})
}

// recordCollectionStreamError records a stream fetch error for a collection item.
func (s *ServiceImpl) recordCollectionStreamError(item *DownloadItem, title string, err error) {
	s.recordError(&DownloadError{
		Category:       item.Category,
		ItemID:         item.ItemID,
		ItemTitle:      title,
		ItemURL:        item.URL,
		ParentCategory: item.Category,
		ParentID:       item.ItemID,
		ParentTitle:    title,
		Phase:          fmt.Sprintf("fetching %s streams", item.Category.ToSubcategory()),
		Error:          err,
	})
}

// getTracksSortedByPosition returns a sorted slice of track IDs by their position field from metadata.
func (s *ServiceImpl) getTracksSortedByPosition(
	tracksMetadata map[string]*zvuk.Track,
	trackIDs []int64,
) []int64 {
	if len(trackIDs) <= 1 {
		return append([]int64(nil), trackIDs...)
	}

	// trackWithPosition pairs a track ID with its playlist position for sorting.
	type trackWithPosition struct {
		// id is the track identifier.
		id int64
		// position is the track order within the collection.
		position int64
	}

	tracks := make([]*trackWithPosition, 0, len(trackIDs))
	missing := make([]int64, 0, len(trackIDs))

	for _, id := range trackIDs {
		idStr := strconv.FormatInt(id, 10)
		if track, ok := tracksMetadata[idStr]; ok && track != nil {
			tracks = append(tracks, &trackWithPosition{
				id:       id,
				position: track.Position,
			})

			continue
		}

		missing = append(missing, id)
	}

	// Fallback to original order when metadata is missing for all tracks.
	if len(tracks) == 0 {
		return append([]int64(nil), trackIDs...)
	}

	// Sort by position.
	slices.SortFunc(tracks, func(a, b *trackWithPosition) int {
		return cmp.Compare(a.position, b.position)
	})

	// Extract sorted IDs.
	result := make([]int64, 0, len(trackIDs))
	for _, track := range tracks {
		result = append(result, track.id)
	}

	// Keep tracks without metadata in their original relative order.
	result = append(result, missing...)

	return result
}

// getFolderNameAfterTemplateExecution gets a sanitized folder path from a raw path string after template execution and truncates it to the maximum allowed length.
func (s *ServiceImpl) getFolderNameAfterTemplateExecution(
	ctx context.Context,
	category DownloadCategory,
	rawPath string,
) string {
	// Split using both separators to handle mixed/foreign path formats.
	components := strings.FieldsFunc(rawPath, func(r rune) bool {
		// Handle both Unix and Windows paths.
		return r == '/' || r == '\\'
	})

	// Sanitize each component individually to prevent path traversal attacks.
	// Keep empty components to maintain path structure (e.g., "a//b" becomes "a/b").
	sanitizedComponents := utils.Map(components, utils.SanitizeTemplateName)

	// Join with OS-specific separators and normalize path.
	joinedPath := filepath.Join(sanitizedComponents...)

	// Truncate to filesystem limits while preserving extension (if any).
	return s.truncateFolderName(ctx, category, joinedPath)
}

// parseItemCoverURL parses a cover URL to extract the URL and extension.
func (s *ServiceImpl) parseItemCoverURL(sourceURL string) *parsedCoverURL {
	// Parse the URL to extract query parameters.
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		// Fallback: remove the size parameter and return the URL as-is.
		return &parsedCoverURL{
			url:       strings.Replace(sourceURL, "&size={size}", "", 1),
			extension: "",
		}
	}

	// Extract the file extension from the query parameters.
	query := parsedURL.Query()
	extension := strings.TrimSpace(query.Get("ext"))
	query.Del("size")
	parsedURL.RawQuery = query.Encode()

	return &parsedCoverURL{
		url:       parsedURL.String(),
		extension: extension,
	}
}

// downloadCover downloads the cover art for albums, playlists, audiobooks and podcasts.
func (s *ServiceImpl) downloadCover(
	ctx context.Context,
	category DownloadCategory,
	sourceURL string,
	itemPath string,
	firstTrackFilename string,
) (string, string) {
	// Trim and validate the cover art URL.
	trimmedSourceURL := strings.TrimSpace(sourceURL)
	if trimmedSourceURL == "" {
		return "", ""
	}

	var (
		coverURL       string
		coverExtension string
	)

	// Handle different URL types like full URLs and relative URLs.
	if strings.HasPrefix(trimmedSourceURL, "http") {
		parsedCover := s.parseItemCoverURL(trimmedSourceURL)

		coverURL = parsedCover.url
		coverExtension = parsedCover.extension
	} else {
		var err error

		coverURL, err = url.JoinPath(s.zvukClient.GetBaseURL(), trimmedSourceURL)
		if err != nil {
			logger.Errorf(ctx, "Failed to generate full cover URL '%s': %v", trimmedSourceURL, err)
			return "", ""
		}

		coverExtension = path.Ext(trimmedSourceURL)
	}

	// Set default extension if none found.
	if coverExtension == "" {
		switch category {
		case DownloadCategoryPlaylist:
			coverExtension = extensionPNG
		default:
			coverExtension = extensionJPG
		}
	}

	// Calculate the final cover filename.
	var finalCoverFilename string
	if firstTrackFilename != "" {
		finalCoverFilename = utils.SetFileExtension(firstTrackFilename, coverExtension, true)
	} else {
		finalCoverFilename = utils.SetFileExtension(defaultCoverFilename, coverExtension, false)
	}

	finalPath := filepath.Join(itemPath, finalCoverFilename)

	if !s.cfg.ReplaceCovers {
		if _, err := os.Stat(finalPath); err == nil {
			logger.Infof(ctx, "%s cover already exists, skipping download", category.ToTitleCase())
			s.incrementCoverSkipped()

			return finalPath, ""
		}
	}

	// Generate filename for the cover art.
	// Use UUID for the filename to avoid concurrent overwrites.
	downloadFilename := utils.SetFileExtension(defaultCoverFilename+"_"+uuid.New().String(), coverExtension, false)
	downloadPath := filepath.Join(itemPath, downloadFilename)

	// Download the cover art.
	isExist, err := s.downloadAndSaveFile(ctx, coverURL, downloadPath, s.cfg.ReplaceCovers)
	if err != nil {
		logger.Errorf(ctx, "Failed to download %s cover: %v", category.String(), err)
		return "", ""
	}

	// Increment the cover skipped or downloaded counter.
	if isExist {
		logger.Infof(ctx, "%s cover already exists, skipping download", category.ToTitleCase())
		s.incrementCoverSkipped()
	} else {
		logger.Infof(ctx, "Successfully downloaded %s cover", category.String())
		s.incrementCoverDownloaded()
	}

	// Return the temp or final path based on the useTemporaryFiles flag.
	return downloadPath, finalPath
}

// finalizeCover finalizes the album cover art.
func (s *ServiceImpl) finalizeCover(
	ctx context.Context,
	itemIndex int64,
	audioCollection *audioCollection,
) {
	// Skip in dry-run mode or
	// if the item index is not the last item
	// or if the embeddable cover path is not set
	// or if the cover path is not set.
	if s.cfg.DryRun ||
		(itemIndex != audioCollection.tracksCount) ||
		(audioCollection.embeddableCoverPath == "") ||
		(audioCollection.coverPath == "") {
		return
	}

	finalizeAsset(
		ctx,
		audioCollection.embeddableCoverPath,
		audioCollection.coverPath,
		s.cfg.ReplaceCovers,
		"cover",
	)
}

// finalizeAsset renames a temporary asset file to its final destination.
func finalizeAsset(ctx context.Context, sourcePath, destinationPath string, replace bool, assetName string) {
	sourceStats, err := os.Stat(sourcePath)
	if err != nil {
		return
	}

	destinationStats, err := os.Stat(destinationPath)
	if err == nil && os.SameFile(sourceStats, destinationStats) {
		return
	}

	if err = utils.RenameFile(sourcePath, destinationPath, replace); err != nil {
		logger.Errorf(ctx, "Failed to rename %s from '%s' to '%s': %v", assetName, sourcePath, destinationPath, err)
	}
}

// saveDescription saves the description file for audiobooks and podcasts.
func (s *ServiceImpl) saveDescription(
	ctx context.Context,
	category DownloadCategory,
	itemPath string,
	description string,
	descriptionFilename string,
) (string, string) {
	if description == "" {
		return "", ""
	}

	// Check if final destination already exists (to avoid inconsistent state).
	var finalFilename string
	if descriptionFilename != "" {
		finalFilename = utils.SetFileExtension(descriptionFilename, extensionTXT, true)
	} else {
		finalFilename = utils.SetFileExtension(defaultDescriptionFilename, extensionTXT, false)
	}

	finalPath := filepath.Join(itemPath, finalFilename)

	// Check if file exists and should not be replaced.
	_, err := os.Stat(finalPath)
	if err == nil && !s.cfg.ReplaceDescriptions {
		logMessage := "%s description already exists, skipping save"
		if s.cfg.DryRun {
			logMessage = "[DRY-RUN] %s description file already exists, would skip"
		}

		logger.Infof(ctx, logMessage, category.ToTitleCase())
		s.incrementDescriptionSkipped()

		return finalPath, ""
	}

	// Generate UUID-based temp filename to avoid concurrent download conflicts.
	downloadFilename := utils.SetFileExtension(defaultDescriptionFilename+"_"+uuid.New().String(), extensionTXT, false)
	downloadPath := filepath.Join(itemPath, downloadFilename)

	// Dry-run mode: simulate description save.
	if s.cfg.DryRun {
		logger.Infof(ctx, "[DRY-RUN] Would save %s description to: %s", category.String(), downloadFilename)
		return downloadPath, finalPath
	}

	// Write description in UTF-8 encoding.
	err = os.WriteFile(downloadPath, []byte(description), files.DefaultFilePermissions)
	if err != nil {
		logger.Errorf(ctx, "Failed to save %s description: %v", category.String(), err)
		return "", ""
	}

	logger.Infof(ctx, "Saved %s description to %s", category.String(), downloadFilename)
	s.incrementDescriptionSaved()

	return downloadPath, finalPath
}

// finalizeDescription renames the description file for audiobooks and podcasts.
func (s *ServiceImpl) finalizeDescription(
	ctx context.Context,
	audioCollection *audioCollection,
	itemIndex int64,
) {
	// Skip in dry-run mode or
	// if the collection is not an audiobook or podcast
	// or if the item index is not the last item
	// or if the embeddable description path is not set
	// or if the final description path is not set.
	if s.cfg.DryRun ||
		!audioCollection.category.IsChapterCollection() ||
		(itemIndex != audioCollection.tracksCount) ||
		(audioCollection.embeddableDescriptionPath == "") ||
		(audioCollection.descriptionPath == "") {
		return
	}

	finalizeAsset(
		ctx,
		audioCollection.embeddableDescriptionPath,
		audioCollection.descriptionPath,
		s.cfg.ReplaceDescriptions,
		"description",
	)
}
