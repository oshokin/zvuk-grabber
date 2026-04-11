package zvuk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
)

// PlaylistCollectionHandler handles playlist collection logic.
type PlaylistCollectionHandler struct {
	BaseCollectionHandler
}

func NewPlaylistCollectionHandler(templateManager TemplateManager) *PlaylistCollectionHandler {
	return &PlaylistCollectionHandler{
		BaseCollectionHandler: BaseCollectionHandler{
			Category:             DownloadCategoryPlaylist,
			TemplateManager:      templateManager,
			SingleFolderHandling: false,
			DescriptionSupport:   false,
		},
	}
}

// LogMessage returns the log message for a playlist.
func (h *PlaylistCollectionHandler) LogMessage(
	ctx context.Context,
	item *zvuk.Playlist,
	tags map[string]string,
) string {
	return fmt.Sprintf("Downloading %s: %s", h.Category.ToTitleCase(), item.Title)
}

// FillTags fills the tags for a playlist.
func (h *PlaylistCollectionHandler) FillTags(item *zvuk.Playlist) map[string]string {
	// Moved from fillPlaylistTags.
	return map[string]string{
		TagType:               h.Category.ToLowerCase(),
		TagPlaylistID:         strconv.FormatInt(item.ID, 10),
		TagPlaylistTitle:      item.Title,
		TagPlaylistTrackCount: strconv.FormatInt(int64(len(item.TrackIDs)), 10),
	}
}

// GetTitle returns the title for a playlist.
func (h *PlaylistCollectionHandler) GetTitle(item *zvuk.Playlist) string {
	return item.Title
}

// GetTrackIDs returns the track IDs for a playlist.
func (h *PlaylistCollectionHandler) GetTrackIDs(item *zvuk.Playlist) []int64 {
	return item.TrackIDs
}

// GetCoverURL returns the cover URL for a playlist.
func (h *PlaylistCollectionHandler) GetCoverURL(item *zvuk.Playlist) string {
	return item.BigImageURL
}

// GetDescription returns the description for a playlist.
func (h *PlaylistCollectionHandler) GetDescription(item *zvuk.Playlist) string {
	return "" // Playlists don't have descriptions.
}

// GetFolderNameTemplate returns the folder name template for a playlist.
func (h *PlaylistCollectionHandler) GetFolderNameTemplate(ctx context.Context, tags map[string]string) string {
	return "" // Playlists don't use template, handled differently.
}

// GetFirstTrackFilename returns the first track filename for a playlist.
func (h *PlaylistCollectionHandler) GetFirstTrackFilename(
	ctx context.Context,
	track *zvuk.Track,
	tags map[string]string,
	tracksCount int64,
) string {
	return "" // No single handling for playlists.
}
