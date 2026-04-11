package zvuk

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// AudiobookCollectionHandler handles audiobook collection logic.
type AudiobookCollectionHandler struct {
	BaseCollectionHandler
}

func NewAudiobookCollectionHandler(templateManager TemplateManager) *AudiobookCollectionHandler {
	return &AudiobookCollectionHandler{
		BaseCollectionHandler: BaseCollectionHandler{
			Category:             DownloadCategoryAudiobook,
			TemplateManager:      templateManager,
			SingleFolderHandling: true,
			DescriptionSupport:   true,
		},
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
		tags[TagAudiobookTitle],
		tags[TagAudiobookAuthors],
	)
}

// FillTags fills the tags for an audiobook.
func (h *AudiobookCollectionHandler) FillTags(item *zvuk.Audiobook) map[string]string {
	releaseDate, publishYear := h.parsePublicationDateAndYear(item.PublicationDate)

	genreTag := h.Category.ToTitleCase()
	if len(item.Genres) > 0 {
		genreTag = strings.Join(item.Genres, ", ")
	}

	tags := map[string]string{
		TagType:                   h.Category.ToLowerCase(),
		TagAudiobookID:            strconv.FormatInt(item.ID, 10),
		TagAudiobookTitle:         item.Title,
		TagAudiobookAuthors:       strings.Join(item.ArtistNames, ", "),
		TagAudiobookTrackCount:    strconv.FormatInt(int64(len(item.TrackIDs)), 10),
		TagAudiobookPublisher:     item.PublisherBrand,
		TagAudiobookPublisherName: item.PublisherName,
		TagAudiobookCopyright:     item.Copyright,
		TagAudiobookDescription:   item.Description,
		TagAudiobookGenres:        strings.Join(item.Genres, ", "),
		TagPublishYear:            publishYear,
		TagReleaseDate:            releaseDate,
		TagReleaseYear:            publishYear,
		// Tag processor compatibility fields.
		TagAlbumID:     strconv.FormatInt(item.ID, 10),
		TagAlbumArtist: strings.Join(item.ArtistNames, ", "),
		TagTrackGenre:  genreTag,
		TagRecordLabel: item.PublisherBrand,
	}

	if item.PublicationDate != "" {
		tags[TagAudiobookPublicationDate] = item.PublicationDate
	}

	if len(item.PerformerNames) > 0 {
		tags[TagAudiobookPerformers] = strings.Join(item.PerformerNames, ", ")
	}

	if item.AgeLimit > 0 {
		tags[TagAudiobookAgeLimit] = strconv.FormatInt(item.AgeLimit, 10)
	}

	if item.FullDuration > 0 {
		tags[TagAudiobookDuration] = strconv.FormatInt(item.FullDuration, 10)
	}

	return tags
}

// GetTitle returns the title for an audiobook.
func (h *AudiobookCollectionHandler) GetTitle(item *zvuk.Audiobook) string {
	return item.Title
}

// GetTrackIDs returns the track IDs for an audiobook.
func (h *AudiobookCollectionHandler) GetTrackIDs(item *zvuk.Audiobook) []int64 {
	return item.TrackIDs
}

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
func (h *AudiobookCollectionHandler) parsePublicationDateAndYear(publicationDate string) (string, string) {
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
