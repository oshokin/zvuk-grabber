package zvuk

import (
	"context"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

const (
	// defaultPlaylistCoverExtension is the default file extension for playlist cover images.
	defaultPlaylistCoverExtension = ".png"
	// defaultPlaylistCoverFilename is the default filename for playlist cover images.
	defaultPlaylistCoverFilename = "cover" + defaultPlaylistCoverExtension
)

func (s *ServiceImpl) downloadPlaylist(ctx context.Context, playlistID string) {
	// Fetch metadata for the playlist.
	getPlaylistsMetadataResponse, err := s.zvukClient.GetPlaylistsMetadata(ctx, []string{playlistID})
	if err != nil {
		logger.Errorf(ctx, "Failed to get metadata for playlist with ID '%s': %v", playlistID, err)

		return
	}

	// Fetch album and label metadata for the tracks in the playlist.
	fetchAlbumsDataFromTracksResponse, err := s.fetchAlbumsDataFromTracks(ctx, getPlaylistsMetadataResponse.Tracks)
	if err != nil {
		logger.Errorf(ctx, "Failed to fetch album and label metadata: %v", err)

		return
	}

	// Register the playlist in the audio container (create folders, download cover, etc.).
	audioCollection := s.addPlaylistToAudioContainer(ctx, playlistID, getPlaylistsMetadataResponse.Playlists)
	if audioCollection == nil {
		return
	}

	// Prepare metadata for downloading the playlist tracks.
	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryPlaylist,
		trackIDs:        audioCollection.trackIDs,
		tracksMetadata:  getPlaylistsMetadataResponse.Tracks,
		albumsMetadata:  fetchAlbumsDataFromTracksResponse.releases,
		albumsTags:      fetchAlbumsDataFromTracksResponse.releasesTags,
		labelsMetadata:  fetchAlbumsDataFromTracksResponse.labels,
		audioCollection: audioCollection,
	}

	// Download all tracks in the playlist.
	s.downloadTracks(ctx, metadata)
}

func (s *ServiceImpl) addPlaylistToAudioContainer(
	ctx context.Context,
	playlistID string,
	playlists map[string]*zvuk.Playlist,
) *audioCollection {
	// Retrieve the playlist from the metadata.
	playlist, ok := playlists[playlistID]
	if !ok || playlist == nil {
		logger.Errorf(ctx, "Playlist with ID '%s' is not found", playlistID)

		return nil
	}

	logger.Infof(ctx, "Downloading playlist: %s", playlist.Title)

	// Generate a sanitized folder name for the playlist and truncate if necessary.
	playlistFolderName := s.truncateFolderName(ctx, "Playlist", utils.SanitizeFilename(playlist.Title))
	playlistPath := filepath.Join(s.cfg.OutputPath, playlistFolderName)

	// Create the playlist folder.
	err := os.MkdirAll(playlistPath, defaultFolderPermissions)
	if err != nil {
		logger.Errorf(ctx, "Failed to create playlist folder '%s': %v", playlistPath, err)

		return nil
	}

	// Download the playlist cover art.
	playlistCoverPath := s.downloadPlaylistCover(ctx, playlist.BigImageURL, playlistPath)

	// Generate tags for the playlist.
	playlistTags := s.fillPlaylistTags(playlist)

	// Lock to ensure thread-safe access to the audio collections.
	s.audioCollectionsMutex.Lock()
	defer s.audioCollectionsMutex.Unlock()

	// Create and register the audio collection for the playlist.
	audioCollectionKey := ShortDownloadItem{
		Category: DownloadCategoryPlaylist,
		ItemID:   playlistID,
	}
	audioCollection := &audioCollection{
		category:    DownloadCategoryPlaylist,
		title:       playlist.Title,
		tags:        playlistTags,
		tracksPath:  playlistPath,
		coverPath:   playlistCoverPath,
		trackIDs:    playlist.TrackIDs,
		tracksCount: int64(len(playlist.TrackIDs)),
	}

	s.audioCollections[audioCollectionKey] = audioCollection

	return audioCollection
}

func (s *ServiceImpl) downloadPlaylistCover(ctx context.Context, bigImageURL, playlistPath string) string {
	// Trim and validate the cover art URL.
	bigImageURL = strings.TrimSpace(bigImageURL)
	if bigImageURL == "" {
		return ""
	}

	// Generate the full URL for the cover art.
	playlistCoverURL, err := url.JoinPath(s.zvukClient.GetBaseURL(), bigImageURL)
	if err != nil {
		logger.Errorf(ctx, "Failed to generate full playlist cover URL '%s': %v", bigImageURL, err)

		return ""
	}

	// Determine the file extension for the cover art.
	playlistCoverExtension := path.Ext(bigImageURL)
	if playlistCoverExtension == "" {
		playlistCoverExtension = defaultPlaylistCoverExtension
	}

	// Generate the filename and path for the cover art.
	playlistCoverFilename := utils.SetFileExtension(defaultPlaylistCoverFilename, playlistCoverExtension, false)
	playlistCoverPath := filepath.Join(playlistPath, playlistCoverFilename)

	// Download and save the cover art.
	err = s.downloadAndSaveFile(ctx, playlistCoverURL, playlistCoverPath, s.cfg.ReplaceCovers)
	if err != nil {
		logger.Errorf(ctx, "Failed to download playlist cover: %v", err)

		return ""
	}

	return playlistCoverPath
}

func (s *ServiceImpl) fillPlaylistTags(playlist *zvuk.Playlist) map[string]string {
	return map[string]string{
		"type":               "playlist",
		"playlistID":         strconv.FormatInt(playlist.ID, 10),
		"playlistTitle":      playlist.Title,
		"playlistTrackCount": strconv.FormatInt(int64(len(playlist.TrackIDs)), 10),
	}
}
