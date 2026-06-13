package zvuk

import (
	"context"
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

// folderHandler defines folder naming behavior for collections.
type folderHandler interface {
	// HasSingleFolderHandling reports whether tracks are stored in a per-collection folder.
	HasSingleFolderHandling() bool
	// GetFolderNameTemplate returns the folder name template for the collection.
	GetFolderNameTemplate(ctx context.Context, tags map[string]string) string
	// GetFirstTrackFilename returns the filename for a single-track collection without a folder.
	GetFirstTrackFilename(ctx context.Context, track *zvuk.Track, tags map[string]string, tracksCount int64) string
}

// registerCollection registers a collection.
type registerCollectionCoreInput struct {
	// Category is the type of content being registered.
	Category DownloadCategory
	// ItemID is the unique identifier of the collection item.
	ItemID string
	// Title is the human-readable collection title.
	Title string
	// TrackIDs is the ordered list of track IDs in the collection.
	TrackIDs []int64
	// Tags contains metadata key-value pairs for the collection.
	Tags map[string]string
	// ItemFolderName is the resolved output folder name for the collection.
	ItemFolderName string
	// FirstTrackFilename is the filename used when a single track has no folder.
	FirstTrackFilename string
	// CoverURL is the remote URL of the collection cover art.
	CoverURL string
	// Description is the collection description text.
	Description string
	// DescriptionSupport indicates whether description files are supported.
	DescriptionSupport bool
}

// collectionHandler defines type-specific collection metadata extraction.
type collectionHandler[T any] interface {
	folderHandler
	// GetCategory returns the download category for this handler.
	GetCategory() DownloadCategory
	// HasDescription reports whether the collection supports descriptions.
	HasDescription() bool
	// FillTags builds metadata tags from the collection item.
	FillTags(item *T) map[string]string
	// LogMessage returns the download start log message for the collection.
	LogMessage(ctx context.Context, item *T, tags map[string]string) string
	// GetTitle returns the display title of the collection item.
	GetTitle(item *T) string
	// GetTrackIDs returns the track IDs belonging to the collection.
	GetTrackIDs(item *T) []int64
	// GetCoverURL returns the cover art URL for the collection.
	GetCoverURL(item *T) string
	// GetDescription returns the description text for the collection.
	GetDescription(item *T) string
}

// newBaseCollectionHandler creates a BaseCollectionHandler with the given settings.
func newBaseCollectionHandler(
	category DownloadCategory,
	templateManager TemplateManager,
	singleFolderHandling bool,
	descriptionSupport bool,
) BaseCollectionHandler {
	return BaseCollectionHandler{
		Category:             category,
		TemplateManager:      templateManager,
		SingleFolderHandling: singleFolderHandling,
		DescriptionSupport:   descriptionSupport,
	}
}

// HasSingleFolderHandling returns true if the collection has single folder handling.
func (b *BaseCollectionHandler) HasSingleFolderHandling() bool {
	return b.SingleFolderHandling
}

// HasDescription returns true if the collection has description support.
func (b *BaseCollectionHandler) HasDescription() bool {
	return b.DescriptionSupport
}

// GetCategory returns the collection download category.
func (b *BaseCollectionHandler) GetCategory() DownloadCategory {
	return b.Category
}

// deriveCollectionTitle picks the best display title from collection tags.
func deriveCollectionTitle(tags map[string]string) string {
	for _, key := range []string{TagCollectionTitle, TagAlbumTitle, TagPlaylistTitle, TagAudiobookTitle, TagPodcastTitle} {
		if title := tags[key]; title != "" {
			return title
		}
	}

	return ""
}

// handleDescription saves and embeds the collection description when supported.
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

	content, err := os.ReadFile(embeddableDescriptionPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf(ctx, "Failed to read existing description file '%s': %v", embeddableDescriptionPath, err)
		}

		return embeddableDescriptionPath, descriptionPath
	}

	switch in.Category {
	case DownloadCategoryAudiobook:
		in.Tags[TagAudiobookDescription] = string(content)
	case DownloadCategoryPodcast:
		in.Tags[TagPodcastDescription] = string(content)
	}

	logger.Infof(ctx, "Updated %s tags with description from existing file", in.Category.String())

	return embeddableDescriptionPath, descriptionPath
}

// determineItemFolderAndFilename resolves the output folder and first track filename.
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

	if audioCollection.category.IsChapterCollection() {
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
	fillCommonTrackTags(result, trackNumber, track)
	result[TagTrackCount] = strconv.FormatInt(audioCollection.tracksCount, 10)

	return result
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
			logger.Errorf(ctx, "Failed to create %s folder '%s': %v", in.Category.String(), itemPath, err)
			return nil
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create %s folder: %s", in.Category.String(), itemPath)
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

// registerCollection registers a typed collection and returns its audio collection context.
func registerCollection[T any](
	ctx context.Context,
	s *ServiceImpl,
	itemID string,
	items map[string]*T,
	tracksMetadata map[string]*zvuk.Track,
	logDownloadStart bool,
	h collectionHandler[T],
) *audioCollection {
	item, ok := items[itemID]
	if !ok || item == nil {
		logger.Errorf(ctx, "%s with ID '%s' is not found", h.GetCategory().ToTitleCase(), itemID)

		return nil
	}

	tags := h.FillTags(item)
	if logDownloadStart {
		if msg := h.LogMessage(ctx, item, tags); msg != "" {
			logger.Infof(ctx, msg)
		}
	}

	title := h.GetTitle(item)
	trackIDs := h.GetTrackIDs(item)
	folderName, firstTrackFilename := determineItemFolderAndFilename(
		ctx,
		s,
		h,
		h.GetCategory(),
		int64(len(trackIDs)),
		trackIDs,
		tracksMetadata,
		tags,
		title,
	)

	return registerCollectionCore(ctx, s, &registerCollectionCoreInput{
		Category:           h.GetCategory(),
		ItemID:             itemID,
		Title:              title,
		TrackIDs:           trackIDs,
		Tags:               tags,
		ItemFolderName:     folderName,
		FirstTrackFilename: firstTrackFilename,
		CoverURL:           h.GetCoverURL(item),
		Description:        h.GetDescription(item),
		DescriptionSupport: h.HasDescription(),
	})
}
