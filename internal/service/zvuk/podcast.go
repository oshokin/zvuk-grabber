package zvuk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// PodcastCollectionHandler handles podcast collection logic.
type PodcastCollectionHandler struct {
	// BaseCollectionHandler provides shared collection handler behavior.
	BaseCollectionHandler
}

// parseEpisodePublicationDate parses episode publication date to YYYY-MM-DD format.
func parseEpisodePublicationDate(publicationDateISO string) string {
	date, _ := parsePublicationDateAndYear(publicationDateISO)
	return date
}

// NewPodcastCollectionHandler creates a handler for podcast downloads.
func NewPodcastCollectionHandler(templateManager TemplateManager) *PodcastCollectionHandler {
	return &PodcastCollectionHandler{
		BaseCollectionHandler: newBaseCollectionHandler(DownloadCategoryPodcast, templateManager, true, true),
	}
}

// LogMessage returns the log message for a podcast.
func (h *PodcastCollectionHandler) LogMessage(ctx context.Context, item *zvuk.Podcast, tags map[string]string) string {
	return fmt.Sprintf(
		"Downloading %s: %s by %s",
		h.Category.ToTitleCase(),
		tags[TagPodcastTitle],
		tags[TagPodcastAuthors],
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
		TagType:               h.Category.String(),
		TagPodcastID:          strconv.FormatInt(item.ID, 10),
		TagPodcastTitle:       item.Title,
		TagPodcastAuthors:     strings.Join(item.ArtistNames, ", "),
		TagPodcastTrackCount:  strconv.FormatInt(int64(len(item.TrackIDs)), 10),
		TagPodcastDescription: item.Description,
		TagPodcastCategory:    item.Category,
		// Tag processor compatibility fields.
		TagAlbumID:     strconv.FormatInt(item.ID, 10),
		TagAlbumArtist: strings.Join(item.ArtistNames, ", "),
		TagTrackGenre:  genreTag,
	}

	// Add explicit flag if set.
	if item.Explicit {
		tags[TagPodcastExplicit] = "true"
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
