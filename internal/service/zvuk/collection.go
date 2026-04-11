package zvuk

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// BaseCollectionHandler provides default implementations for common collection logic.
type BaseCollectionHandler struct {
	// Category is the category of the collection.
	Category DownloadCategory
	// TemplateManager is the template manager.
	TemplateManager TemplateManager
	// SingleFolderHandling indicates if the collection should have a single folder.
	SingleFolderHandling bool
	// DescriptionSupport indicates if the collection should have description support.
	DescriptionSupport bool
}

// HasSingleFolderHandling returns true if the collection has single folder handling.
func (b *BaseCollectionHandler) HasSingleFolderHandling() bool {
	return b.SingleFolderHandling
}

// HasDescription returns true if the collection has description support.
func (b *BaseCollectionHandler) HasDescription() bool {
	return b.DescriptionSupport
}

func deriveCollectionTitle(tags map[string]string) string {
	collectionTitle := tags[TagCollectionTitle]
	if collectionTitle != "" {
		return collectionTitle
	}

	if v := tags[TagAlbumTitle]; v != "" {
		return v
	}

	if v := tags[TagPlaylistTitle]; v != "" {
		return v
	}

	if v := tags[TagAudiobookTitle]; v != "" {
		return v
	}

	if v := tags[TagPodcastTitle]; v != "" {
		return v
	}

	return ""
}

func handleDescription(
	ctx context.Context,
	s *ServiceImpl,
	in *registerCollectionCoreInput,
	itemPath string,
) (string, string) {
	if !in.DescriptionSupport {
		return "", ""
	}

	embeddableDescriptionPath, descriptionPath := s.saveDescription(
		ctx,
		in.Category,
		itemPath,
		in.Description,
		in.FirstTrackFilename,
	)

	if embeddableDescriptionPath == "" {
		return embeddableDescriptionPath, descriptionPath
	}

	if _, statErr := os.Stat(embeddableDescriptionPath); statErr != nil {
		return embeddableDescriptionPath, descriptionPath
	}

	content, readErr := os.ReadFile(embeddableDescriptionPath)
	if readErr != nil {
		logger.Warnf(
			ctx,
			"Failed to read existing description file '%s': %v",
			embeddableDescriptionPath,
			readErr,
		)

		return embeddableDescriptionPath, descriptionPath
	}

	switch in.Category {
	case DownloadCategoryAudiobook:
		in.Tags[TagAudiobookDescription] = string(content)
	case DownloadCategoryPodcast:
		in.Tags[TagPodcastDescription] = string(content)
	}

	logger.Infof(ctx, "Updated %s tags with description from existing file", in.Category.ToLowerCase())

	return embeddableDescriptionPath, descriptionPath
}

type folderHandler interface {
	HasSingleFolderHandling() bool
	GetFolderNameTemplate(ctx context.Context, tags map[string]string) string
	GetFirstTrackFilename(ctx context.Context, track *zvuk.Track, tags map[string]string, tracksCount int64) string
}

func determineItemFolderAndFilename(
	ctx context.Context,
	s *ServiceImpl,
	h folderHandler,
	category DownloadCategory,
	tracksCount int64,
	trackIDs []int64,
	tracksMetadata map[string]*zvuk.Track,
	itemTags map[string]string,
	title string,
) (string, string) {
	if !h.HasSingleFolderHandling() {
		return s.truncateFolderName(ctx, category, strings.TrimSpace(title)), ""
	}

	isSingleWithoutFolder := !s.cfg.CreateFolderForSingles && tracksCount == 1
	if !isSingleWithoutFolder {
		rawItemFolderName := h.GetFolderNameTemplate(ctx, itemTags)
		itemFolderName := s.getFolderNameAfterTemplateExecution(ctx, category, rawItemFolderName)

		return itemFolderName, ""
	}

	if tracksCount != 1 {
		return "", ""
	}

	trackID := strconv.FormatInt(trackIDs[0], 10)

	track, exists := tracksMetadata[trackID]
	if !exists {
		return "", ""
	}

	firstTrackFilename := h.GetFirstTrackFilename(ctx, track, itemTags, tracksCount)

	return "", firstTrackFilename
}

// GetFirstTrackFilename returns the first track filename for a collection.
func (b *BaseCollectionHandler) GetFirstTrackFilename(
	ctx context.Context,
	track *zvuk.Track,
	tags map[string]string,
	tracksCount int64,
) string {
	// For "single without folder" handling we don't yet have a persisted audioCollection,
	// but template execution still expects collection context (title/category/trackCount).
	collectionTitle := deriveCollectionTitle(tags)

	tempCollection := &audioCollection{
		category:    b.Category,
		title:       collectionTitle,
		tags:        tags,
		tracksCount: tracksCount,
	}

	trackTags := b.FillTrackTagsForTemplating(1, track, tempCollection, tags)

	return b.TemplateManager.GetTrackFilename(ctx, false, trackTags, tracksCount)
}

// FillTrackTagsForTemplating fills the track tags for a collection.
func (b *BaseCollectionHandler) FillTrackTagsForTemplating(
	trackNumber int64,
	track *zvuk.Track,
	audioCollection *audioCollection,
	collectionTags map[string]string,
) map[string]string {
	var result map[string]string

	if audioCollection.category == DownloadCategoryAudiobook ||
		audioCollection.category == DownloadCategoryPodcast {
		result = maps.Clone(audioCollection.tags)

		result[TagTrackGenre] = strings.Join(track.Genres, ", ")
	} else {
		result = make(map[string]string, len(collectionTags)+len(audioCollection.tags))
		maps.Copy(result, collectionTags)

		// Apply collection tags (if it's a playlist, these will override album-specific tags).
		maps.Copy(result, audioCollection.tags)
	}

	// Add track-specific fields.
	result[TagCollectionTitle] = audioCollection.title
	result[TagTrackArtist] = strings.Join(track.ArtistNames, ", ")
	result[TagTrackID] = strconv.FormatInt(track.ID, 10)
	result[TagTrackNumber] = strconv.FormatInt(trackNumber, 10)
	result[TagTrackNumberPad] = fmt.Sprintf("%0*d", trackNumberPaddingWidth, trackNumber)
	result[TagTrackTitle] = track.Title
	result[TagTrackCount] = strconv.FormatInt(audioCollection.tracksCount, 10)

	return result
}

// registerCollection registers a collection.
type registerCollectionCoreInput struct {
	Category           DownloadCategory
	ItemID             string
	Title              string
	TrackIDs           []int64
	Tags               map[string]string
	ItemFolderName     string
	FirstTrackFilename string
	CoverURL           string
	Description        string
	DescriptionSupport bool
}

// registerCollectionCore contains the shared, side-effecting parts of collection registration:
// folder creation, cover/description handling, and dedup registration in service state.
//
// It intentionally takes only plain values (no generics, no callback packs) to keep call sites readable.
func registerCollectionCore(
	ctx context.Context,
	s *ServiceImpl,
	in *registerCollectionCoreInput,
) *audioCollection {
	if in == nil {
		logger.Errorf(ctx, "Failed to register collection: input is nil")
		return nil
	}

	// Create the audio collection.
	audioCollection := &audioCollection{
		category:    in.Category,
		id:          in.ItemID,
		title:       in.Title,
		tags:        in.Tags,
		trackIDs:    in.TrackIDs,
		tracksCount: int64(len(in.TrackIDs)),
	}

	// Create the folder path for the item.
	itemPath := filepath.Join(s.cfg.OutputPath, in.ItemFolderName)

	// Create the folder for the item unless in dry-run mode.
	if !s.cfg.DryRun {
		err := os.MkdirAll(itemPath, defaultFolderPermissions)
		if err != nil {
			logger.Errorf(ctx, "Failed to create %s folder '%s': %v", in.Category.ToLowerCase(), itemPath, err)
			return nil
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create %s folder: %s", in.Category.ToLowerCase(), itemPath)
	}

	// Download cover.
	embeddableCoverPath, coverPath := s.downloadCover(ctx, in.Category, in.CoverURL, itemPath, in.FirstTrackFilename)

	// Handle description if applicable.
	embeddableDescriptionPath, descriptionPath := handleDescription(
		ctx,
		s,
		in,
		itemPath,
	)

	// Check if the audio collection already exists.
	s.audioCollectionsMutex.Lock()
	defer s.audioCollectionsMutex.Unlock()

	audioCollectionKey := ShortDownloadItem{
		Category: in.Category,
		ItemID:   in.ItemID,
	}
	if existing, isExist := s.audioCollections[audioCollectionKey]; isExist && existing != nil {
		return existing
	}

	// Set the paths for the audio collection.
	audioCollection.tracksPath = itemPath
	audioCollection.embeddableCoverPath = embeddableCoverPath
	audioCollection.coverPath = coverPath

	if in.DescriptionSupport {
		audioCollection.embeddableDescriptionPath = embeddableDescriptionPath
		audioCollection.descriptionPath = descriptionPath
	}

	// Register the audio collection.
	s.audioCollections[audioCollectionKey] = audioCollection

	return audioCollection
}

func registerAlbumCollection(
	ctx context.Context,
	s *ServiceImpl,
	albumID string,
	albums map[string]*zvuk.Release,
	tracksMetadata map[string]*zvuk.Track,
	isDownloadStartingBeingLogged bool,
) *audioCollection {
	item, ok := albums[albumID]
	if !ok || item == nil {
		logger.Errorf(ctx, "%s with ID '%s' is not found", DownloadCategoryAlbum.ToTitleCase(), albumID)
		return nil
	}

	h := s.albumHandler
	itemTags := h.FillTags(item)

	if isDownloadStartingBeingLogged {
		if msg := h.LogMessage(ctx, item, itemTags); msg != "" {
			logger.Infof(ctx, msg)
		}
	}

	title := h.GetTitle(item)
	trackIDs := h.GetTrackIDs(item)
	tracksCount := int64(len(trackIDs))

	itemFolderName, firstTrackFilename := determineItemFolderAndFilename(
		ctx,
		s,
		h,
		DownloadCategoryAlbum,
		tracksCount,
		trackIDs,
		tracksMetadata,
		itemTags,
		title,
	)

	return registerCollectionCore(ctx, s, &registerCollectionCoreInput{
		Category:           DownloadCategoryAlbum,
		ItemID:             albumID,
		Title:              title,
		TrackIDs:           trackIDs,
		Tags:               itemTags,
		ItemFolderName:     itemFolderName,
		FirstTrackFilename: firstTrackFilename,
		CoverURL:           h.GetCoverURL(item),
		Description:        "",
		DescriptionSupport: h.HasDescription(),
	})
}

func registerPlaylistCollection(
	ctx context.Context,
	s *ServiceImpl,
	playlistID string,
	playlists map[string]*zvuk.Playlist,
	tracksMetadata map[string]*zvuk.Track,
	isDownloadStartingBeingLogged bool,
) *audioCollection {
	_ = tracksMetadata // playlists never use "single without folder" logic

	item, ok := playlists[playlistID]
	if !ok || item == nil {
		logger.Errorf(ctx, "%s with ID '%s' is not found", DownloadCategoryPlaylist.ToTitleCase(), playlistID)
		return nil
	}

	h := s.playlistHandler
	itemTags := h.FillTags(item)

	if isDownloadStartingBeingLogged {
		if msg := h.LogMessage(ctx, item, itemTags); msg != "" {
			logger.Infof(ctx, msg)
		}
	}

	title := h.GetTitle(item)
	trackIDs := h.GetTrackIDs(item)
	itemFolderName := s.truncateFolderName(ctx, DownloadCategoryPlaylist, strings.TrimSpace(title))

	return registerCollectionCore(ctx, s, &registerCollectionCoreInput{
		Category:           DownloadCategoryPlaylist,
		ItemID:             playlistID,
		Title:              title,
		TrackIDs:           trackIDs,
		Tags:               itemTags,
		ItemFolderName:     itemFolderName,
		FirstTrackFilename: "",
		CoverURL:           h.GetCoverURL(item),
		Description:        "",
		DescriptionSupport: h.HasDescription(),
	})
}

func registerAudiobookCollection(
	ctx context.Context,
	s *ServiceImpl,
	audiobookID string,
	audiobooks map[string]*zvuk.Audiobook,
	tracksMetadata map[string]*zvuk.Track,
	isDownloadStartingBeingLogged bool,
) *audioCollection {
	item, ok := audiobooks[audiobookID]
	if !ok || item == nil {
		logger.Errorf(ctx, "%s with ID '%s' is not found", DownloadCategoryAudiobook.ToTitleCase(), audiobookID)
		return nil
	}

	h := s.audiobookHandler
	itemTags := h.FillTags(item)

	if isDownloadStartingBeingLogged {
		if msg := h.LogMessage(ctx, item, itemTags); msg != "" {
			logger.Infof(ctx, msg)
		}
	}

	title := h.GetTitle(item)
	trackIDs := h.GetTrackIDs(item)
	tracksCount := int64(len(trackIDs))

	itemFolderName, firstTrackFilename := determineItemFolderAndFilename(
		ctx,
		s,
		h,
		DownloadCategoryAudiobook,
		tracksCount,
		trackIDs,
		tracksMetadata,
		itemTags,
		title,
	)

	return registerCollectionCore(ctx, s, &registerCollectionCoreInput{
		Category:           DownloadCategoryAudiobook,
		ItemID:             audiobookID,
		Title:              title,
		TrackIDs:           trackIDs,
		Tags:               itemTags,
		ItemFolderName:     itemFolderName,
		FirstTrackFilename: firstTrackFilename,
		CoverURL:           h.GetCoverURL(item),
		Description:        h.GetDescription(item),
		DescriptionSupport: h.HasDescription(),
	})
}

func registerPodcastCollection(
	ctx context.Context,
	s *ServiceImpl,
	podcastID string,
	podcasts map[string]*zvuk.Podcast,
	tracksMetadata map[string]*zvuk.Track,
	isDownloadStartingBeingLogged bool,
) *audioCollection {
	item, ok := podcasts[podcastID]
	if !ok || item == nil {
		logger.Errorf(ctx, "%s with ID '%s' is not found", DownloadCategoryPodcast.ToTitleCase(), podcastID)
		return nil
	}

	h := s.podcastHandler
	itemTags := h.FillTags(item)

	if isDownloadStartingBeingLogged {
		if msg := h.LogMessage(ctx, item, itemTags); msg != "" {
			logger.Infof(ctx, msg)
		}
	}

	title := h.GetTitle(item)
	trackIDs := h.GetTrackIDs(item)
	tracksCount := int64(len(trackIDs))

	itemFolderName, firstTrackFilename := determineItemFolderAndFilename(
		ctx,
		s,
		h,
		DownloadCategoryPodcast,
		tracksCount,
		trackIDs,
		tracksMetadata,
		itemTags,
		title,
	)

	return registerCollectionCore(ctx, s, &registerCollectionCoreInput{
		Category:           DownloadCategoryPodcast,
		ItemID:             podcastID,
		Title:              title,
		TrackIDs:           trackIDs,
		Tags:               itemTags,
		ItemFolderName:     itemFolderName,
		FirstTrackFilename: firstTrackFilename,
		CoverURL:           h.GetCoverURL(item),
		Description:        h.GetDescription(item),
		DescriptionSupport: h.HasDescription(),
	})
}
