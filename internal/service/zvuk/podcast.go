package zvuk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/media"
)

// PodcastCollectionHandler handles podcast collection logic.
type PodcastCollectionHandler struct {
	// BaseCollectionHandler provides shared collection handler behavior.
	BaseCollectionHandler
}

// NewPodcastCollectionHandler creates a handler for podcast downloads.
func NewPodcastCollectionHandler(templateManager media.TemplateManager) *PodcastCollectionHandler {
	return &PodcastCollectionHandler{
		BaseCollectionHandler: newBaseCollectionHandler(DownloadCategoryPodcast, templateManager, true, true),
	}
}

// parseEpisodePublicationDate parses episode publication date to YYYY-MM-DD format.
func parseEpisodePublicationDate(publicationDateISO string) string {
	date, _ := parsePublicationDateAndYear(publicationDateISO)
	return date
}

// LogMessage returns the log message for a podcast.
func (h *PodcastCollectionHandler) LogMessage(ctx context.Context, item *zvuk.Podcast, tags map[string]string) string {
	return fmt.Sprintf(
		"Downloading %s: %s by %s",
		h.Category.ToTitleCase(),
		tags[media.TagPodcastTitle],
		tags[media.TagPodcastAuthors],
	)
}

// FillTags fills the tags for a podcast.
func (h *PodcastCollectionHandler) FillTags(item *zvuk.Podcast) map[string]string {
	// Determine genre: use podcast category if available, otherwise default to "Podcast".
	genreTag := h.Category.ToTitleCase()
	if item.Category != "" {
		genreTag = item.Category
	}

	tags := map[string]string{
		media.TagType:               h.Category.String(),
		media.TagPodcastID:          strconv.FormatInt(item.ID, 10),
		media.TagPodcastTitle:       item.Title,
		media.TagPodcastAuthors:     strings.Join(item.ArtistNames, ", "),
		media.TagPodcastTrackCount:  strconv.FormatInt(int64(len(item.TrackIDs)), 10),
		media.TagPodcastDescription: item.Description,
		media.TagPodcastCategory:    item.Category,
		// Tag processor compatibility fields.
		media.TagAlbumID:     strconv.FormatInt(item.ID, 10),
		media.TagAlbumArtist: strings.Join(item.ArtistNames, ", "),
		media.TagTrackGenre:  genreTag,
	}

	// Add explicit flag if set.
	if item.Explicit {
		tags[media.TagPodcastExplicit] = "true"
	}

	return tags
}

// GetTitle returns the title for a podcast.
func (h *PodcastCollectionHandler) GetTitle(item *zvuk.Podcast) string { return item.Title }

// GetTrackIDs returns the track IDs for a podcast.
func (h *PodcastCollectionHandler) GetTrackIDs(item *zvuk.Podcast) []int64 { return item.TrackIDs }

// GetCoverURL returns the cover URL for a podcast.
func (h *PodcastCollectionHandler) GetCoverURL(item *zvuk.Podcast) string { return item.BigImageURL }

// GetDescription returns the description for a podcast.
func (h *PodcastCollectionHandler) GetDescription(item *zvuk.Podcast) string { return item.Description }

// GetFolderNameTemplate returns the folder name template for a podcast.
func (h *PodcastCollectionHandler) GetFolderNameTemplate(ctx context.Context, tags map[string]string) string {
	return h.TemplateManager.GetPodcastFolderName(ctx, tags)
}
