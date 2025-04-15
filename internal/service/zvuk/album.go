package zvuk

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

type fetchAlbumDataResponse struct {
	tracks   map[string]*zvuk.Track
	album    *zvuk.Release
	releases map[string]*zvuk.Release
	labels   map[string]*zvuk.Label
}

const (
	defaultAlbumCoverExtension = ".jpg"
	defaultAlbumCoverBasename  = "cover"
)

func (s *ServiceImpl) downloadAlbum(ctx context.Context, albumID string) {
	// Fetch album data (tracks, metadata, labels)
	fetchAlbumDataResponse, err := s.fetchAlbumData(ctx, albumID)
	if err != nil {
		logger.Errorf(ctx, "Failed to fetch album data for ID '%s': %v", albumID, err)

		return
	}

	// Generate tags for templating (e.g., folder names, filenames)
	albumTags := s.fillAlbumTagsForTemplating(fetchAlbumDataResponse.album)

	// Register the album collection (create folders, download cover art, etc.)
	audioCollection := s.registerAlbumCollection(ctx, fetchAlbumDataResponse.album, albumTags, true)
	if audioCollection == nil {
		return
	}

	// Prepare metadata for downloading tracks
	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAlbum,
		trackIDs:        audioCollection.trackIDs,
		tracksMetadata:  fetchAlbumDataResponse.tracks,
		albumsMetadata:  fetchAlbumDataResponse.releases,
		albumsTags:      map[string]map[string]string{albumID: audioCollection.tags},
		labelsMetadata:  fetchAlbumDataResponse.labels,
		audioCollection: audioCollection,
	}

	// Download all tracks in the album
	s.downloadTracks(ctx, metadata)
}

func (s *ServiceImpl) fetchAlbumData(ctx context.Context, albumID string) (*fetchAlbumDataResponse, error) {
	// Fetch album metadata from the API
	getAlbumsMetadataResponse, err := s.zvukClient.GetAlbumsMetadata(ctx, []string{albumID}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get album metadata: %w", err)
	}

	// Retrieve the album from the response
	album, ok := getAlbumsMetadataResponse.Releases[albumID]
	if !ok || album == nil {
		return nil, fmt.Errorf("album with ID '%s' is not found", albumID)
	}

	// Fetch label metadata for the album
	labelIDString := strconv.FormatInt(album.LabelID, 10)

	labelsMetadata, err := s.zvukClient.GetLabelsMetadata(ctx, []string{labelIDString})
	if err != nil {
		return nil, fmt.Errorf("failed to get label metadata: %w", err)
	}

	// Return the fetched data
	return &fetchAlbumDataResponse{
		tracks:   getAlbumsMetadataResponse.Tracks,
		album:    album,
		releases: getAlbumsMetadataResponse.Releases,
		labels:   labelsMetadata,
	}, nil
}

func (s *ServiceImpl) registerAlbumCollection(
	ctx context.Context,
	album *zvuk.Release,
	albumTags map[string]string,
	isAlbumDownload bool,
) *audioCollection {
	// Log the album being downloaded
	if isAlbumDownload {
		logger.Infof(
			ctx,
			"Downloading '%s - %s (%s)'",
			albumTags["albumArtist"],
			albumTags["albumTitle"],
			albumTags["releaseYear"])
	}

	// Determine if the album is a single and should not have a dedicated folder
	isSingleWithoutFolder := !s.cfg.CreateFolderForSingles && len(album.TrackIDs) == 1
	albumFolderName := ""

	// Generate a folder name for the album if it's not a single or if singles require folders
	if !isSingleWithoutFolder {
		// Get raw template output before sanitization (might contain invalid characters)
		rawAlbumFolderName := s.templateManager.GetAlbumFolderName(ctx, albumTags)

		// Universal path handling: process both Unix and Windows separators
		albumFolderName = s.generateSanitizedFolderPath(ctx, rawAlbumFolderName)
	}

	// Create the album folder path by joining with the base output path
	albumPath := filepath.Join(s.cfg.OutputPath, albumFolderName)

	err := os.MkdirAll(albumPath, defaultFolderPermissions)
	if err != nil {
		logger.Errorf(ctx, "Failed to create album folder '%s': %v", albumPath, err)

		return nil
	}

	// Download the album cover art
	albumCoverPath := s.downloadAlbumCover(ctx, album, albumPath)
	albumID := strconv.FormatInt(album.ID, 10)

	// Lock to ensure thread-safe access to audioCollections
	s.audioCollectionsMutex.Lock()
	defer s.audioCollectionsMutex.Unlock()

	// Create and register the audio collection
	audioCollectionKey := ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   albumID,
	}
	audioCollection := &audioCollection{
		category:    DownloadCategoryAlbum,
		title:       albumTags["albumTitle"],
		tags:        albumTags,
		tracksPath:  albumPath,
		coverPath:   albumCoverPath,
		trackIDs:    album.TrackIDs,
		tracksCount: int64(len(album.TrackIDs)),
	}

	s.audioCollections[audioCollectionKey] = audioCollection

	return audioCollection
}

func (s *ServiceImpl) generateSanitizedFolderPath(ctx context.Context, rawPath string) string {
	// Split using both separators to handle mixed/foreign path formats
	components := strings.FieldsFunc(rawPath, func(r rune) bool {
		return r == '/' || r == '\\' // Handle both Unix and Windows paths
	})

	sanitizedComponents := make([]string, 0, len(components))
	for _, component := range components {
		// Sanitize each component individually to prevent path traversal attacks
		clean := utils.SanitizeFilename(component)

		// Keep empty components to maintain path structure (e.g., "a//b" becomes "a/b")
		sanitizedComponents = append(sanitizedComponents, clean)
	}

	// Join with OS-specific separators and normalize path
	joinedPath := filepath.Join(sanitizedComponents...)

	// Truncate to filesystem limits while preserving extension (if any)
	return s.truncateFolderName(ctx, "Album", joinedPath)
}

func (s *ServiceImpl) parseAlbumDate(rawDate int64) (time.Time, string) {
	dateString := strconv.FormatInt(rawDate, 10)

	// Attempt to parse the date in "YYYYMMDD" format
	parsedDate, err := time.Parse("20060102", dateString)
	if err != nil {
		// Fallback: return only the year if parsing fails
		return time.Time{}, dateString[:4]
	}

	return parsedDate, strconv.Itoa(parsedDate.Year())
}

func (s *ServiceImpl) fillAlbumTagsForTemplating(release *zvuk.Release) map[string]string {
	albumDate, albumYear := s.parseAlbumDate(release.Date)

	return map[string]string{
		"albumArtist":      strings.Join(release.ArtistNames, ", "),
		"albumID":          strconv.FormatInt(release.ID, 10),
		"albumTitle":       release.Title,
		"albumTrackCount":  strconv.FormatInt(int64(len(release.TrackIDs)), 10),
		"releaseDate":      albumDate.Format("2006-01-02"),
		"releaseTimestamp": strconv.FormatInt(albumDate.Unix(), 10),
		"releaseYear":      albumYear,
		"type":             "album",
	}
}

func (s *ServiceImpl) downloadAlbumCover(
	ctx context.Context,
	album *zvuk.Release,
	albumPath string,
) string {
	// Check if the album has an image
	if album.Image == nil {
		return ""
	}

	// Trim and validate the source URL
	trimmedSourceURL := strings.TrimSpace(album.Image.SourceURL)
	if trimmedSourceURL == "" {
		return ""
	}

	// Parse the cover URL and determine its extension
	albumCoverURL, albumCoverExtension := s.parseAlbumCoverURL(trimmedSourceURL)
	if albumCoverExtension == "" {
		albumCoverExtension = defaultAlbumCoverExtension
	}

	// Generate the cover filename and path
	albumCoverFilename := utils.SetFileExtension(defaultAlbumCoverBasename, albumCoverExtension, false)
	albumCoverPath := filepath.Join(albumPath, albumCoverFilename)

	// Download and save the cover art
	if err := s.downloadAndSaveFile(ctx, albumCoverURL, albumCoverPath, s.cfg.ReplaceCovers); err != nil {
		logger.Errorf(ctx, "Failed to download album cover: %v", err)

		return ""
	}

	return albumCoverPath
}

func (s *ServiceImpl) parseAlbumCoverURL(sourceURL string) (string, string) {
	// Parse the URL to extract query parameters
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		// Fallback: remove the size parameter and return the URL as-is
		return strings.Replace(sourceURL, "&size={size}", "", 1), ""
	}

	// Extract the file extension from the query parameters
	query := parsedURL.Query()
	ext := strings.TrimSpace(query.Get("ext"))
	query.Del("size")
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), ext
}
