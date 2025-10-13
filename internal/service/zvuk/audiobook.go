package zvuk

import (
	"context"
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

// downloadAudiobook downloads an audiobook and its chapters.
func (s *ServiceImpl) downloadAudiobook(ctx context.Context, item *DownloadItem) {
	audiobookID := item.ItemID

	// Fetch audiobook metadata (chapters) via GraphQL.
	getAudiobooksMetadataResponse, err := s.zvukClient.GetAudiobooksMetadata(ctx, []string{audiobookID})
	if err != nil {
		logger.Errorf(ctx, "Failed to get metadata for audiobook with ID '%s': %v", audiobookID, err)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryAudiobook,
			ItemID:    audiobookID,
			ItemTitle: "Audiobook ID: " + audiobookID,
			ItemURL:   item.URL,
			Phase:     "fetching audiobook metadata",
		}, err)

		return
	}

	// Register the audiobook (create folder, download cover).
	audioCollection := s.addAudiobookToAudioContainer(ctx, audiobookID, getAudiobooksMetadataResponse.Audiobooks)
	if audioCollection == nil {
		return
	}

	// Sort chapters by position to ensure correct playback order.
	sortedChapterIDs := s.sortChaptersByPosition(getAudiobooksMetadataResponse.Tracks, audioCollection.trackIDs)

	// Fetch chapter stream URLs via mediaContents GraphQL query.
	chapterIDs := make([]string, len(sortedChapterIDs))
	for i, trackID := range sortedChapterIDs {
		chapterIDs[i] = strconv.FormatInt(trackID, 10)
	}

	chapterStreams, err := s.zvukClient.GetChapterStreamMetadata(ctx, chapterIDs)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chapter stream URLs: %v", err)
		s.recordError(&ErrorContext{
			Category:  DownloadCategoryAudiobook,
			ItemID:    audiobookID,
			ItemTitle: audioCollection.title,
			ItemURL:   item.URL,
			Phase:     "fetching chapter streams",
		}, err)

		return
	}

	// Prepare metadata for downloading chapters using the unified track download pipeline.
	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryAudiobook,
		trackIDs:        sortedChapterIDs,
		tracksMetadata:  getAudiobooksMetadataResponse.Tracks,
		chapterStreams:  chapterStreams,
		audioCollection: audioCollection,
	}

	// Download all chapters (supports concurrent downloads if configured).
	s.downloadTracks(ctx, metadata)
}

// addAudiobookToAudioContainer adds an audiobook to the audio container.
func (s *ServiceImpl) addAudiobookToAudioContainer(
	ctx context.Context,
	audiobookID string,
	audiobooks map[string]*zvuk.Audiobook,
) *audioCollection {
	// Retrieve the audiobook from the metadata.
	audiobook, ok := audiobooks[audiobookID]
	if !ok || audiobook == nil {
		logger.Errorf(ctx, "Audiobook with ID '%s' is not found", audiobookID)

		return nil
	}

	logger.Infof(ctx, "Downloading audiobook: %s by %s",
		audiobook.Title, strings.Join(audiobook.ArtistNames, ", "))
	logger.Debugf(
		ctx,
		"Audiobook metadata: ID=%d, Publisher=%s, Copyright=%s, Performers=%s, Genres=%s, Chapters=%d, Duration=%ds, PubDate=%s, AgeLimit=%d",
		audiobook.ID,
		audiobook.PublisherBrand,
		audiobook.Copyright,
		strings.Join(audiobook.PerformerNames, ", "),
		strings.Join(audiobook.Genres, ", "),
		len(audiobook.TrackIDs),
		audiobook.FullDuration,
		audiobook.PublicationDate,
		audiobook.AgeLimit,
	)

	// Generate tags for the audiobook.
	audiobookTags := s.fillAudiobookTags(audiobook)

	// Determine if this is a single-chapter audiobook without a dedicated folder.
	isSingleWithoutFolder := !s.cfg.CreateFolderForSingles && len(audiobook.TrackIDs) == 1
	audiobookFolderName := ""

	// Generate folder name for the audiobook if it's not a single or if singles require folders.
	if !isSingleWithoutFolder {
		rawAudiobookFolderName := s.templateManager.GetAudiobookFolderName(ctx, audiobookTags)
		audiobookFolderName = s.truncateFolderName(ctx, "Audiobook", utils.SanitizeFilename(rawAudiobookFolderName))
	}

	// Create the audiobook folder path.
	audiobookPath := filepath.Join(s.cfg.OutputPath, audiobookFolderName)

	// Create folder unless in dry-run mode.
	if !s.cfg.DryRun {
		err := os.MkdirAll(audiobookPath, defaultFolderPermissions)
		if err != nil {
			logger.Errorf(ctx, "Failed to create audiobook folder '%s': %v", audiobookPath, err)

			return nil
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create audiobook folder: %s", audiobookPath)
	}

	// Download the audiobook cover art (use UUID for temp filename to avoid concurrent overwrites).
	audiobookCoverPath, audiobookCoverTempPath := s.downloadAudiobookCover(ctx, audiobook.BigImageURL, audiobookPath)

	// Save audiobook description if available (use UUID for temp filename to avoid concurrent overwrites).
	var descriptionTempPath string
	if audiobook.Description != "" {
		descriptionTempPath = s.saveAudiobookDescription(ctx, audiobook.Description, audiobookPath)
	}

	// Lock to ensure thread-safe access to the audio collections.
	s.audioCollectionsMutex.Lock()
	defer s.audioCollectionsMutex.Unlock()

	// Create and register the audio collection for the audiobook.
	audioCollectionKey := ShortDownloadItem{
		Category: DownloadCategoryAudiobook,
		ItemID:   audiobookID,
	}
	audioCollection := &audioCollection{
		category:            DownloadCategoryAudiobook,
		title:               audiobook.Title,
		tags:                audiobookTags,
		tracksPath:          audiobookPath,
		coverPath:           audiobookCoverPath,
		coverTempPath:       audiobookCoverTempPath,
		descriptionTempPath: descriptionTempPath,
		trackIDs:            audiobook.TrackIDs,
		tracksCount:         int64(len(audiobook.TrackIDs)),
	}

	s.audioCollections[audioCollectionKey] = audioCollection

	return audioCollection
}

// downloadAudiobookCover downloads the audiobook cover.
func (s *ServiceImpl) downloadAudiobookCover(ctx context.Context, bigImageURL, audiobookPath string) (string, string) {
	return s.downloadCover(ctx, bigImageURL, audiobookPath, "Audiobook")
}

// saveAudiobookDescription saves the audiobook description.
func (s *ServiceImpl) saveAudiobookDescription(ctx context.Context, description, audiobookPath string) string {
	return s.saveDescription(ctx, description, audiobookPath, "Audiobook")
}

// finalizeAudiobookDescription renames the description file for single-chapter audiobooks.
func (s *ServiceImpl) finalizeAudiobookDescription(
	ctx context.Context,
	chapterIndex int64,
	audioCollection *audioCollection,
	chapterFilename string,
) {
	s.finalizeDescription(ctx, chapterIndex, audioCollection, chapterFilename)
}

// parseAudiobookPublicationYear parses the publication year from an ISO 8601 date string.
func (s *ServiceImpl) parseAudiobookPublicationYear(publicationDate string) string {
	// Parse ISO 8601 date format: "2023-12-05T09:59:51.549544+00:00"
	if publicationDate == "" {
		return defaultUnknownYear
	}

	parsedDate, err := time.Parse(time.RFC3339, publicationDate)
	if err != nil {
		// Try to extract just the year from the beginning.
		if len(publicationDate) >= 4 {
			return publicationDate[:4]
		}

		return defaultUnknownYear
	}

	return strconv.Itoa(parsedDate.Year())
}

// parseAudiobookPublicationDate parses the publication date from an ISO 8601 date string to "YYYY-MM-DD" format.
func (s *ServiceImpl) parseAudiobookPublicationDate(publicationDate string) string {
	// Parse ISO 8601 date format: "2023-12-05T09:59:51.549544+00:00" to "2023-12-05"
	if publicationDate == "" {
		return ""
	}

	parsedDate, err := time.Parse(time.RFC3339, publicationDate)
	if err != nil {
		// Try to extract YYYY-MM-DD from the beginning.
		if len(publicationDate) >= 10 {
			return publicationDate[:10]
		}

		return ""
	}

	return parsedDate.Format("2006-01-02")
}

// fillAudiobookTags fills the audiobook tags.
func (s *ServiceImpl) fillAudiobookTags(audiobook *zvuk.Audiobook) map[string]string {
	// Parse publication date and extract year.
	publishYear := s.parseAudiobookPublicationYear(audiobook.PublicationDate)
	releaseDate := s.parseAudiobookPublicationDate(audiobook.PublicationDate)

	// Determine genre: use audiobook genres if available, otherwise default to "Audiobook".
	genreTag := "Audiobook"
	if len(audiobook.Genres) > 0 {
		genreTag = strings.Join(audiobook.Genres, ", ")
	}

	tags := map[string]string{
		"type":                   "audiobook",
		"audiobookID":            strconv.FormatInt(audiobook.ID, 10),
		"audiobookTitle":         audiobook.Title,
		"audiobookAuthors":       strings.Join(audiobook.ArtistNames, ", "),
		"audiobookTrackCount":    strconv.FormatInt(int64(len(audiobook.TrackIDs)), 10),
		"audiobookPublisher":     audiobook.PublisherBrand,
		"audiobookPublisherName": audiobook.PublisherName,
		"audiobookCopyright":     audiobook.Copyright,
		"audiobookDescription":   audiobook.Description,
		"audiobookGenres":        strings.Join(audiobook.Genres, ", "),
		"publishYear":            publishYear,
		"releaseDate":            releaseDate,
		// Tag processor compatibility fields.
		"releaseYear": publishYear, // Tag processor expects this for YEAR tag.
		"albumID":     strconv.FormatInt(audiobook.ID, 10),
		"albumArtist": strings.Join(audiobook.ArtistNames, ", "),
		"trackGenre":  genreTag,
		"recordLabel": audiobook.PublisherBrand,
	}

	// Add publication date if available.
	if audiobook.PublicationDate != "" {
		tags["audiobookPublicationDate"] = audiobook.PublicationDate
	}

	// Add performers if available.
	if len(audiobook.PerformerNames) > 0 {
		tags["audiobookPerformers"] = strings.Join(audiobook.PerformerNames, ", ")
	}

	// Add age limit if set.
	if audiobook.AgeLimit > 0 {
		tags["audiobookAgeLimit"] = strconv.FormatInt(audiobook.AgeLimit, 10)
	}

	// Add full duration if available.
	if audiobook.FullDuration > 0 {
		tags["audiobookDuration"] = strconv.FormatInt(audiobook.FullDuration, 10)
	}

	return tags
}

// sortChaptersByPosition sorts chapter IDs by their position field from metadata.
func (s *ServiceImpl) sortChaptersByPosition(
	chaptersMetadata map[string]*zvuk.Track,
	chapterIDs []int64,
) []int64 {
	// Create a slice for sorting with position info.
	type chapterWithPosition struct {
		id       int64
		position int64
	}

	chapters := make([]chapterWithPosition, 0, len(chapterIDs))
	for _, id := range chapterIDs {
		idStr := strconv.FormatInt(id, 10)
		if chapter, ok := chaptersMetadata[idStr]; ok && chapter != nil {
			chapters = append(chapters, chapterWithPosition{
				id:       id,
				position: chapter.Position,
			})
		}
	}

	// Sort by position.
	slices.SortFunc(chapters, func(a, b chapterWithPosition) int {
		return int(a.position - b.position)
	})

	// Extract sorted IDs.
	result := make([]int64, len(chapters))
	for i, ch := range chapters {
		result[i] = ch.id
	}

	return result
}
