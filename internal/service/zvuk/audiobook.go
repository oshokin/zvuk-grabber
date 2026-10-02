package zvuk

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// AudiobookCollectionHandler handles audiobook collection logic.
type AudiobookCollectionHandler struct {
	// BaseCollectionHandler provides shared collection handler behavior.
	BaseCollectionHandler
}

// NewAudiobookCollectionHandler creates a handler for audiobook downloads.
func NewAudiobookCollectionHandler(templateManager media.TemplateManager) *AudiobookCollectionHandler {
	return &AudiobookCollectionHandler{
		BaseCollectionHandler: newBaseCollectionHandler(DownloadCategoryAudiobook, templateManager, true, true),
	}
}

// LogMessage returns the log message for an audiobook.
func (h *AudiobookCollectionHandler) LogMessage(
	ctx context.Context,
	item *zvuk.Audiobook,
	tags map[string]string,
) string {
	return fmt.Sprintf(
		"Downloading %s: %s by %s",
		h.Category.ToTitleCase(),
		tags[media.TagAudiobookTitle],
		tags[media.TagAudiobookAuthors],
	)
}

// FillTags fills the tags for an audiobook.
func (h *AudiobookCollectionHandler) FillTags(item *zvuk.Audiobook) map[string]string {
	releaseDate, publishYear := parsePublicationDateAndYear(item.PublicationDate)

	genreTag := h.Category.ToTitleCase()
	if len(item.Genres) > 0 {
		genreTag = strings.Join(item.Genres, ", ")
	}

	tags := map[string]string{
		media.TagType:                   h.Category.String(),
		media.TagAudiobookID:            strconv.FormatInt(item.ID, 10),
		media.TagAudiobookTitle:         item.Title,
		media.TagAudiobookAuthors:       strings.Join(item.ArtistNames, ", "),
		media.TagAudiobookTrackCount:    strconv.FormatInt(int64(len(item.TrackIDs)), 10),
		media.TagAudiobookPublisher:     item.PublisherBrand,
		media.TagAudiobookPublisherName: item.PublisherName,
		media.TagAudiobookCopyright:     item.Copyright,
		media.TagAudiobookDescription:   item.Description,
		media.TagAudiobookGenres:        strings.Join(item.Genres, ", "),
		media.TagPublishYear:            publishYear,
		media.TagReleaseDate:            releaseDate,
		media.TagReleaseYear:            publishYear,
		// Tag processor compatibility fields.
		media.TagAlbumID:     strconv.FormatInt(item.ID, 10),
		media.TagAlbumArtist: strings.Join(item.ArtistNames, ", "),
		media.TagTrackGenre:  genreTag,
		media.TagRecordLabel: item.PublisherBrand,
	}

	if item.PublicationDate != "" {
		tags[media.TagAudiobookPublicationDate] = item.PublicationDate
	}

	if len(item.PerformerNames) > 0 {
		tags[media.TagAudiobookPerformers] = strings.Join(item.PerformerNames, ", ")
	}

	if item.AgeLimit > 0 {
		tags[media.TagAudiobookAgeLimit] = strconv.FormatInt(item.AgeLimit, 10)
	}

	if item.FullDuration > 0 {
		tags[media.TagAudiobookDuration] = strconv.FormatInt(item.FullDuration, 10)
	}

	return tags
}

// GetTitle returns the title for an audiobook.
func (h *AudiobookCollectionHandler) GetTitle(item *zvuk.Audiobook) string { return item.Title }

// GetTrackIDs returns the track IDs for an audiobook.
func (h *AudiobookCollectionHandler) GetTrackIDs(item *zvuk.Audiobook) []int64 { return item.TrackIDs }

// GetCoverURL returns the cover URL for an audiobook.
func (h *AudiobookCollectionHandler) GetCoverURL(item *zvuk.Audiobook) string {
	return item.BigImageURL
}

// GetDescription returns the description for an audiobook.
func (h *AudiobookCollectionHandler) GetDescription(item *zvuk.Audiobook) string {
	return item.Description
}

// GetFolderNameTemplate returns the folder name template for an audiobook.
func (h *AudiobookCollectionHandler) GetFolderNameTemplate(ctx context.Context, tags map[string]string) string {
	return h.TemplateManager.GetAudiobookFolderName(ctx, tags)
}

// parsePublicationDateAndYear parses the publication date and year from an ISO 8601 date string.
func parsePublicationDateAndYear(publicationDate string) (string, string) {
	if publicationDate == "" {
		return "", defaultUnknownYear
	}

	parsedDate, err := time.Parse(time.RFC3339, publicationDate)
	if err != nil {
		if len(publicationDate) >= 10 {
			return publicationDate[:10], publicationDate[:4]
		}

		return "", defaultUnknownYear
	}

	return parsedDate.Format("2006-01-02"), strconv.Itoa(parsedDate.Year())
}
