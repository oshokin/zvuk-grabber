package zvuk

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// downloadPodcast downloads a podcast and its episodes.
func (s *ServiceImpl) downloadPodcast(ctx context.Context, item *DownloadItem) {
	podcastID := item.ItemID

	// Fetch podcast metadata (episodes) via GraphQL.
	getPodcastsMetadataResponse, err := s.zvukClient.GetPodcastsMetadata(ctx, []string{podcastID})
	if err != nil {
		logger.Errorf(ctx, "Failed to get metadata for podcast with ID '%s': %v", podcastID, err)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryPodcast,
			ItemID:    podcastID,
			ItemTitle: "Podcast ID: " + podcastID,
			ItemURL:   item.URL,
			Phase:     "fetching podcast metadata",
		}, err)

		return
	}

	// Register the podcast (create folder, download cover).
	audioCollection := s.addPodcastToAudioContainer(ctx, podcastID, getPodcastsMetadataResponse.Podcasts)
	if audioCollection == nil {
		return
	}

	// Sort episodes by position to ensure correct order.
	sortedEpisodeIDs := s.sortEpisodesByPosition(getPodcastsMetadataResponse.Tracks, audioCollection.trackIDs)

	// Fetch episode stream URLs via mediaContents GraphQL query.
	episodeIDs := make([]string, len(sortedEpisodeIDs))
	for i, trackID := range sortedEpisodeIDs {
		episodeIDs[i] = strconv.FormatInt(trackID, 10)
	}

	episodeStreams, err := s.zvukClient.GetChapterStreamMetadata(ctx, episodeIDs)
	if err != nil {
		logger.Errorf(ctx, "Failed to get episode stream URLs: %v", err)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryPodcast,
			ItemID:    podcastID,
			ItemTitle: audioCollection.title,
			ItemURL:   item.URL,
			Phase:     "fetching episode streams",
		}, err)

		return
	}

	// Prepare metadata for downloading episodes using the unified track download pipeline.
	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryPodcast,
		trackIDs:        sortedEpisodeIDs,
		tracksMetadata:  getPodcastsMetadataResponse.Tracks,
		chapterStreams:  episodeStreams,
		audioCollection: audioCollection,
	}

	// Download all episodes (supports concurrent downloads if configured).
	s.downloadTracks(ctx, metadata)
}

// addPodcastToAudioContainer adds a podcast to the audio container.
func (s *ServiceImpl) addPodcastToAudioContainer(
	ctx context.Context,
	podcastID string,
	podcasts map[string]*zvuk.Podcast,
) *audioCollection {
	// Retrieve the podcast from the metadata.
	podcast, ok := podcasts[podcastID]
	if !ok || podcast == nil {
		logger.Errorf(ctx, "Podcast with ID '%s' is not found", podcastID)

		return nil
	}

	logger.Infof(ctx, "Downloading podcast: %s by %s",
		podcast.Title, strings.Join(podcast.ArtistNames, ", "))
	logger.Debugf(
		ctx,
		"Podcast metadata: ID=%d, Category=%s, Episodes=%d, Explicit=%v",
		podcast.ID,
		podcast.Category,
		len(podcast.TrackIDs),
		podcast.Explicit,
	)

	// Generate tags for the podcast.
	podcastTags := s.fillPodcastTags(podcast)

	// Determine if this is a single-episode podcast without a dedicated folder.
	isSingleWithoutFolder := !s.cfg.CreateFolderForSingles && len(podcast.TrackIDs) == 1
	podcastFolderName := ""

	// Generate folder name for the podcast if it's not a single or if singles require folders.
	if !isSingleWithoutFolder {
		rawPodcastFolderName := s.templateManager.GetPodcastFolderName(ctx, podcastTags)
		podcastFolderName = s.truncateFolderName(ctx, "Podcast", utils.SanitizeFilename(rawPodcastFolderName))
	}

	// Create the podcast folder path.
	podcastPath := filepath.Join(s.cfg.OutputPath, podcastFolderName)

	// Create folder unless in dry-run mode.
	if !s.cfg.DryRun {
		err := os.MkdirAll(podcastPath, defaultFolderPermissions)
		if err != nil {
			logger.Errorf(ctx, "Failed to create podcast folder '%s': %v", podcastPath, err)

			return nil
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create podcast folder: %s", podcastPath)
	}

	// Download the podcast cover art (use UUID for temp filename to avoid concurrent overwrites).
	podcastCoverPath, podcastCoverTempPath := s.downloadPodcastCover(ctx, podcast.BigImageURL, podcastPath)

	// Save podcast description if available (use UUID for temp filename to avoid concurrent overwrites).
	var descriptionTempPath string
	if podcast.Description != "" {
		descriptionTempPath = s.savePodcastDescription(ctx, podcast.Description, podcastPath)
	}

	// Lock to ensure thread-safe access to the audio collections.
	s.audioCollectionsMutex.Lock()
	defer s.audioCollectionsMutex.Unlock()

	// Create and register the audio collection for the podcast.
	audioCollectionKey := ShortDownloadItem{
		Category: DownloadCategoryPodcast,
		ItemID:   podcastID,
	}
	audioCollection := &audioCollection{
		category:            DownloadCategoryPodcast,
		title:               podcast.Title,
		tags:                podcastTags,
		tracksPath:          podcastPath,
		coverPath:           podcastCoverPath,
		coverTempPath:       podcastCoverTempPath,
		descriptionTempPath: descriptionTempPath,
		trackIDs:            podcast.TrackIDs,
		tracksCount:         int64(len(podcast.TrackIDs)),
	}

	s.audioCollections[audioCollectionKey] = audioCollection

	return audioCollection
}

// downloadPodcastCover downloads the podcast cover.
func (s *ServiceImpl) downloadPodcastCover(ctx context.Context, bigImageURL, podcastPath string) (string, string) {
	return s.downloadCover(ctx, bigImageURL, podcastPath, "Podcast")
}

// savePodcastDescription saves the podcast description.
func (s *ServiceImpl) savePodcastDescription(ctx context.Context, description, podcastPath string) string {
	return s.saveDescription(ctx, description, podcastPath, "Podcast")
}

// finalizePodcastDescription renames the description file for single-episode podcasts.
func (s *ServiceImpl) finalizePodcastDescription(
	ctx context.Context,
	episodeIndex int64,
	audioCollection *audioCollection,
	episodeFilename string,
) {
	s.finalizeDescription(ctx, episodeIndex, audioCollection, episodeFilename)
}

// fillPodcastTags fills the podcast tags.
func (s *ServiceImpl) fillPodcastTags(podcast *zvuk.Podcast) map[string]string {
	// Determine genre: use podcast category if available, otherwise default to "Podcast".
	genreTag := "Podcast"
	if podcast.Category != "" {
		genreTag = podcast.Category
	}

	tags := map[string]string{
		"type":               "podcast",
		"podcastID":          strconv.FormatInt(podcast.ID, 10),
		"podcastTitle":       podcast.Title,
		"podcastAuthors":     strings.Join(podcast.ArtistNames, ", "),
		"podcastTrackCount":  strconv.FormatInt(int64(len(podcast.TrackIDs)), 10),
		"podcastDescription": podcast.Description,
		"podcastCategory":    podcast.Category,
		// Tag processor compatibility fields.
		"albumID":     strconv.FormatInt(podcast.ID, 10),
		"albumArtist": strings.Join(podcast.ArtistNames, ", "),
		"trackGenre":  genreTag,
	}

	// Add explicit flag if set.
	if podcast.Explicit {
		tags["podcastExplicit"] = "true"
	}

	return tags
}

// fillEpisodeTags fills episode-specific tags including publication date.
func (s *ServiceImpl) fillEpisodeTags(
	track *zvuk.Track,
	podcastTags map[string]string,
	episodeIndex int64,
) map[string]string {
	// Start with podcast tags as base.
	tags := make(map[string]string)
	maps.Copy(tags, podcastTags)

	// Parse publication date from track.Credits (where we stored it during parsing).
	publicationDate := s.parseEpisodePublicationDate(track.Credits)

	// Add episode-specific tags.
	tags["episodeID"] = strconv.FormatInt(track.ID, 10)
	tags["episodeTitle"] = track.Title
	tags["episodeDuration"] = strconv.FormatInt(track.Duration, 10)
	tags["episodeNumber"] = strconv.FormatInt(episodeIndex, 10)
	tags["episodeNumberPad"] = fmt.Sprintf("%02d", episodeIndex)
	tags["episodePublicationDate"] = publicationDate

	// Standard track tags for template compatibility.
	tags["trackID"] = strconv.FormatInt(track.ID, 10)
	tags["trackTitle"] = track.Title
	tags["trackNumber"] = strconv.FormatInt(episodeIndex, 10)
	tags["trackNumberPad"] = fmt.Sprintf("%02d", episodeIndex)
	tags["trackDuration"] = strconv.FormatInt(track.Duration, 10)

	return tags
}

// parseEpisodePublicationDate parses episode publication date to YYYY-MM-DD format.
func (s *ServiceImpl) parseEpisodePublicationDate(publicationDateISO string) string {
	if publicationDateISO == "" {
		return ""
	}

	// Parse ISO 8601 date format: "2020-05-04T00:00:00"
	parsedDate, err := time.Parse(time.RFC3339, publicationDateISO)
	if err != nil {
		// Try to extract YYYY-MM-DD from the beginning.
		if len(publicationDateISO) >= 10 {
			return publicationDateISO[:10]
		}

		return ""
	}

	return parsedDate.Format("2006-01-02")
}

// sortEpisodesByPosition sorts episode IDs by their position field from metadata.
// Episodes come in reverse chronological order by default, so we maintain that order.
func (s *ServiceImpl) sortEpisodesByPosition(
	episodesMetadata map[string]*zvuk.Track,
	episodeIDs []int64,
) []int64 {
	// Create a slice for sorting with position info.
	type episodeWithPosition struct {
		id       int64
		position int64
	}

	episodes := make([]episodeWithPosition, 0, len(episodeIDs))
	for i, id := range episodeIDs {
		idStr := strconv.FormatInt(id, 10)
		if episode, ok := episodesMetadata[idStr]; ok && episode != nil {
			episodes = append(episodes, episodeWithPosition{
				id: id,
				// Use array index as position since episodes come in order from API.
				position: int64(i),
			})
		}
	}

	// Sort by position.
	slices.SortFunc(episodes, func(a, b episodeWithPosition) int {
		return int(a.position - b.position)
	})

	// Extract sorted IDs.
	result := make([]int64, len(episodes))
	for i, ep := range episodes {
		result[i] = ep.id
	}

	return result
}
