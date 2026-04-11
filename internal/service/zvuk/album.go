package zvuk

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// fetchAlbumDataResponse contains the complete metadata for an album.
type fetchAlbumDataResponse struct {
	// tracks contains track metadata mapped by track ID.
	tracks map[string]*zvuk.Track
	// album contains the main album/release metadata.
	album *zvuk.Release
	// releases contains additional release metadata mapped by release ID.
	releases map[string]*zvuk.Release
	// labels contains music label metadata mapped by label ID.
	labels map[string]*zvuk.Label
}

// parsedCoverURL contains the parsed cover URL and extension.
type parsedCoverURL struct {
	url       string
	extension string
}

// Static error definitions for better error handling.
var (
	// ErrAlbumNotFound indicates that the requested album was not found.
	ErrAlbumNotFound = errors.New("album not found")
)

// fetchAlbumData fetches album data including tracks, metadata, and labels.
func (s *ServiceImpl) fetchAlbumData(ctx context.Context, albumID string) (*fetchAlbumDataResponse, error) {
	// Fetch album metadata from the API.
	getAlbumsMetadataResponse, err := s.zvukClient.GetAlbumsMetadata(ctx, []string{albumID}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get album metadata: %w", err)
	}

	// Retrieve the album from the response.
	album, ok := getAlbumsMetadataResponse.Releases[albumID]
	if !ok || album == nil {
		return nil, fmt.Errorf("%w: ID '%s'", ErrAlbumNotFound, albumID)
	}

	// Fetch label metadata for the album.
	labelIDString := strconv.FormatInt(album.LabelID, 10)

	labelsMetadata, err := s.zvukClient.GetLabelsMetadata(ctx, []string{labelIDString})
	if err != nil {
		return nil, fmt.Errorf("failed to get label metadata: %w", err)
	}

	// Return the fetched data.
	return &fetchAlbumDataResponse{
		tracks:   getAlbumsMetadataResponse.Tracks,
		album:    album,
		releases: getAlbumsMetadataResponse.Releases,
		labels:   labelsMetadata,
	}, nil
}

// AlbumCollectionHandler handles album collection logic.
type AlbumCollectionHandler struct {
	BaseCollectionHandler
}

func NewAlbumCollectionHandler(templateManager TemplateManager) *AlbumCollectionHandler {
	return &AlbumCollectionHandler{
		BaseCollectionHandler: BaseCollectionHandler{
			Category:             DownloadCategoryAlbum,
			TemplateManager:      templateManager,
			SingleFolderHandling: true,
			DescriptionSupport:   false,
		},
	}
}

// LogMessage returns the log message for an album.
func (h *AlbumCollectionHandler) LogMessage(ctx context.Context, item *zvuk.Release, tags map[string]string) string {
	return fmt.Sprintf(
		"Downloading %s: %s - %s (%s)",
		h.Category.ToLowerCase(),
		tags[TagAlbumArtist],
		tags[TagAlbumTitle],
		tags[TagReleaseYear],
	)
}

// FillTags fills the tags for an album.
func (h *AlbumCollectionHandler) FillTags(item *zvuk.Release) map[string]string {
	dateString := strconv.FormatInt(item.Date, 10)

	albumYear := defaultUnknownYear
	if len(dateString) >= 4 {
		albumYear = dateString[:4]
	}

	var (
		albumDate        string
		releaseTimestamp string
	)

	parsedDate, err := time.Parse("20060102", dateString)
	if err == nil {
		albumDate = parsedDate.Format("2006-01-02")
		albumYear = strconv.Itoa(parsedDate.Year())
		releaseTimestamp = strconv.FormatInt(parsedDate.Unix(), 10)
	}

	return map[string]string{
		TagAlbumArtist:      strings.Join(item.ArtistNames, ", "),
		TagAlbumID:          strconv.FormatInt(item.ID, 10),
		TagAlbumTitle:       item.Title,
		TagAlbumTrackCount:  strconv.FormatInt(int64(len(item.TrackIDs)), 10),
		TagReleaseDate:      albumDate,
		TagReleaseTimestamp: releaseTimestamp,
		TagReleaseYear:      albumYear,
		TagType:             h.Category.ToLowerCase(),
	}
}

// GetTitle returns the title for an album.
func (h *AlbumCollectionHandler) GetTitle(item *zvuk.Release) string {
	return item.Title
}

// GetTrackIDs returns the track IDs for an album.
func (h *AlbumCollectionHandler) GetTrackIDs(item *zvuk.Release) []int64 {
	return item.TrackIDs
}

// GetCoverURL returns the cover URL for an album.
func (h *AlbumCollectionHandler) GetCoverURL(item *zvuk.Release) string {
	if item.Image != nil {
		return item.Image.SourceURL
	}

	return ""
}

// GetDescription returns the description for an album.
func (h *AlbumCollectionHandler) GetDescription(item *zvuk.Release) string {
	return "" // Albums don't have descriptions.
}

// GetFolderNameTemplate returns the folder name template for an album.
func (h *AlbumCollectionHandler) GetFolderNameTemplate(ctx context.Context, tags map[string]string) string {
	return h.TemplateManager.GetAlbumFolderName(ctx, tags)
}
