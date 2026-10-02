package zvuk

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/service/stats"
)

// Service provides methods for downloading audio content from Zvuk URLs.
type Service interface {
	// DownloadURLs orchestrates the full download pipeline, from URL processing to file creation.
	DownloadURLs(ctx context.Context, urls []string) error
	// PrintDownloadSummary prints a formatted summary of download statistics.
	PrintDownloadSummary(ctx context.Context)
}

// ServiceImpl implements audio download service with deduplication and metadata handling.
type ServiceImpl struct {
	// cfg contains the application configuration.
	cfg *config.Config
	// zvukClient is the client for interacting with Zvuk's API.
	zvukClient zvuk.Client
	// urlProcessor handles URL parsing and categorization.
	urlProcessor URLProcessor
	// templateManager generates filenames and folder names.
	templateManager media.TemplateManager
	// tagProcessor writes metadata tags to audio files.
	tagProcessor media.TagProcessor
	// audioCollections stores download collections indexed by item.
	audioCollections map[ShortDownloadItem]*audioCollection
	// audioCollectionsMutex protects concurrent access to audioCollections.
	audioCollectionsMutex *sync.Mutex
	// albumHandler is the collection handler for albums.
	albumHandler *AlbumCollectionHandler
	// playlistHandler is the collection handler for playlists.
	playlistHandler *PlaylistCollectionHandler
	// audiobookHandler is the collection handler for audiobooks.
	audiobookHandler *AudiobookCollectionHandler
	// podcastHandler is the collection handler for podcasts.
	podcastHandler *PodcastCollectionHandler
	// validator validates track constraints.
	validator *TrackValidator
	// stats tracks download statistics for the current session.
	stats *DownloadStatistics
	// statsMutex protects concurrent access to statistics.
	statsMutex sync.Mutex
	// pathLocks serializes writes to the same destination path.
	pathLocks *files.PathLocks
}

// NewService creates a download service instance with dependency-injected components.
func NewService(
	cfg *config.Config,
	zvukClient zvuk.Client,
	urlProcessor URLProcessor,
	templateManager media.TemplateManager,
	tagProcessor media.TagProcessor,
) Service {
	s := &ServiceImpl{
		cfg:                   cfg,
		zvukClient:            zvukClient,
		urlProcessor:          urlProcessor,
		templateManager:       templateManager,
		tagProcessor:          tagProcessor,
		audioCollections:      make(map[ShortDownloadItem]*audioCollection),
		audioCollectionsMutex: new(sync.Mutex),
		albumHandler:          NewAlbumCollectionHandler(templateManager),
		playlistHandler:       NewPlaylistCollectionHandler(templateManager),
		audiobookHandler:      NewAudiobookCollectionHandler(templateManager),
		podcastHandler:        NewPodcastCollectionHandler(templateManager),
		validator:             NewTrackValidator(cfg),
		stats:                 new(DownloadStatistics),
		pathLocks:             files.NewPathLocks(),
	}

	return s
}

// outputPath returns the effective output folder for Zvuk downloads.
//
//nolint:funcorder // Keep output path helper close to constructor and config wiring.
func (s *ServiceImpl) outputPath() string {
	return s.cfg.ResolveOutputPath(config.ProviderZvuk)
}

// DownloadURLs orchestrates the full download pipeline, from URL processing to file creation.
func (s *ServiceImpl) DownloadURLs(ctx context.Context, urls []string) (result error) {
	defer func() {
		snapshot := s.statsSnapshot()
		if result == nil && (snapshot.TracksFailed > 0 || len(snapshot.Errors) > 0) {
			result = fmt.Errorf(
				"%w (Zvuk): %d failed tracks, %d recorded errors",
				stats.ErrDownloadsFailed,
				snapshot.TracksFailed,
				len(snapshot.Errors),
			)
		}

		if ctx.Err() != nil {
			result = ctx.Err()
		}
	}()
	// Record start time and dry-run mode for statistics.
	s.statsMutex.Lock()
	s.stats.StartTime = time.Now()
	s.stats.IsDryRun = s.cfg.DryRun
	s.statsMutex.Unlock()

	// Ensure the output directory exists (skip in dry-run mode).
	outputPath := s.outputPath()
	if !s.cfg.DryRun {
		err := os.MkdirAll(outputPath, defaultFolderPermissions)
		if err != nil {
			logger.Errorf(ctx, "Failed to create output path: %v", err)
			return err
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create output directory: %s", outputPath)
	}

	// Verify the user's subscription status before proceeding.
	if err := s.checkUserSubscription(ctx); err != nil {
		return err
	}

	// Extract and categorize download items from the provided URLs.
	downloadItemsByCategories, err := s.urlProcessor.ExtractDownloadItems(ctx, urls)
	if err != nil {
		logger.Errorf(ctx, "Failed to extract items to download: %v", err)
		return err
	}

	logger.Info(ctx, "Starting Zvuk download process")

	// Process albums and playlists first to maintain organizational structure.
	standaloneItems := s.fetchAndDeduplicateStandaloneItems(ctx, downloadItemsByCategories)
	if len(standaloneItems) > 0 {
		s.downloadStandaloneItems(ctx, standaloneItems)
	}

	// Process individual tracks after collections to allow potential deduplication.
	if len(downloadItemsByCategories.Tracks) > 0 {
		s.downloadTrackItems(ctx, downloadItemsByCategories.Tracks)
	}

	logger.Info(ctx, "Zvuk download process completed")

	// Record end time for statistics.
	s.statsMutex.Lock()
	s.stats.EndTime = time.Now()
	s.statsMutex.Unlock()

	return nil
}

// fetchAndDeduplicateStandaloneItems processes artist URLs to fetch their albums and removes duplicate entries.
func (s *ServiceImpl) fetchAndDeduplicateStandaloneItems(
	ctx context.Context,
	items *ExtractDownloadItemsResponse,
) []*DownloadItem {
	standaloneItems := items.StandaloneItems

	// If artist URLs are present, fetch their albums and append them to the standalone items.
	if len(items.Artists) > 0 {
		artistAlbums := s.fetchArtistAlbums(ctx, items.Artists)
		standaloneItems = append(standaloneItems, artistAlbums...)
		// Remove duplicate album entries that might exist in the original URLs.
		standaloneItems = s.urlProcessor.DeduplicateDownloadItems(standaloneItems)
	}

	return standaloneItems
}

// downloadStandaloneItems handles the download of albums, playlists, audiobooks, and podcasts.
func (s *ServiceImpl) downloadStandaloneItems(ctx context.Context, items []*DownloadItem) {
	logger.Info(ctx, "Downloading albums, playlists, audiobooks, and podcasts")

	itemsCount := len(items)

	// Iterate through each item and download based on its category.
	for index, item := range items {
		// Check if context was canceled (CTRL+C pressed) - stop immediately.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check if the category is supported for downloading.
		if !item.Category.IsSupported() {
			logger.Errorf(ctx, "Unknown URL category: %d", item.Category)
			continue
		}

		// Download the collection.
		logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)

		s.downloadCollection(ctx, item)
	}
}

// downloadTrackItems handles the download of individual tracks.
func (s *ServiceImpl) downloadTrackItems(ctx context.Context, items []*DownloadItem) {
	logger.Info(ctx, "Downloading tracks")

	trackIDsToFetch, numericTrackIDs := s.prepareStandaloneTrackIDs(ctx, items)
	if len(trackIDsToFetch) == 0 {
		return
	}

	// Fetch metadata for the tracks.
	tracksMetadata, err := s.zvukClient.GetTracksMetadata(ctx, trackIDsToFetch)
	if err != nil {
		logger.Errorf(ctx, "Failed to get track metadata: %v", err)
		s.recordStandaloneTrackBatchError(trackIDsToFetch, "fetching track metadata", err)

		return
	}

	// Fetch album and label metadata for the tracks.
	fetchAlbumsDataFromTracksResponse, err := s.fetchAlbumsDataFromTracks(ctx, tracksMetadata)
	if err != nil {
		logger.Errorf(ctx, "Failed to fetch album and label metadata: %v", err)
		s.recordStandaloneTrackBatchError(trackIDsToFetch, "fetching album and label metadata", err)

		return
	}

	// Prepare metadata for downloading the tracks.
	metadata := &downloadTracksMetadata{
		category:       DownloadCategoryTrack,
		trackIDs:       numericTrackIDs,
		tracksMetadata: tracksMetadata,
		albumsMetadata: fetchAlbumsDataFromTracksResponse.releases,
		albumsTags:     fetchAlbumsDataFromTracksResponse.releasesTags,
		labelsMetadata: fetchAlbumsDataFromTracksResponse.labels,
	}

	// Download the tracks.
	s.downloadTracks(ctx, metadata)
}

// prepareStandaloneTrackIDs parses, deduplicates, and filters standalone track download items.
func (s *ServiceImpl) prepareStandaloneTrackIDs(ctx context.Context, items []*DownloadItem) ([]string, []int64) {
	items = s.urlProcessor.DeduplicateDownloadItems(items)

	numericTrackIDs := make([]int64, 0, len(items))
	trackIDsToFetch := make([]string, 0, len(items))
	registeredCollectionTrackIDs := s.getRegisteredCollectionTrackIDs()

	for _, item := range items {
		trackIDString := item.ItemID

		trackID, err := strconv.ParseInt(trackIDString, 10, 64)
		if err != nil {
			logger.Errorf(ctx, "Failed to parse track ID '%s': %v", trackIDString, err)
			s.recordStandaloneTrackError(
				trackIDString,
				"parsing track ID",
				fmt.Errorf("invalid track ID '%s': %w", trackIDString, err),
			)

			continue
		}

		if _, isExist := registeredCollectionTrackIDs[trackID]; isExist {
			logger.Infof(ctx,
				"Track ID '%s' is already covered by previously processed collections, skipping standalone download",
				trackIDString)
			s.incrementTrackSkipped(SkipReasonExists)

			continue
		}

		numericTrackIDs = append(numericTrackIDs, trackID)
		trackIDsToFetch = append(trackIDsToFetch, trackIDString)
	}

	return trackIDsToFetch, numericTrackIDs
}

// recordStandaloneTrackBatchError records the same error for each track ID in the batch.
func (s *ServiceImpl) recordStandaloneTrackBatchError(trackIDs []string, phase string, err error) {
	for _, trackID := range trackIDs {
		s.recordStandaloneTrackError(trackID, phase, err)
	}
}

// recordStandaloneTrackError records a download error for a standalone track.
func (s *ServiceImpl) recordStandaloneTrackError(trackID, phase string, err error) {
	s.recordError(&DownloadError{
		Category:       DownloadCategoryTrack,
		ItemID:         trackID,
		ItemTitle:      "Standalone track",
		ParentCategory: DownloadCategoryTrack,
		ParentID:       "standalone-tracks",
		ParentTitle:    "standalone track URLs",
		Phase:          phase,
		Error:          err,
	})
}

// getRegisteredCollectionTrackIDs returns track IDs already covered by registered collections.
func (s *ServiceImpl) getRegisteredCollectionTrackIDs() map[int64]struct{} {
	result := make(map[int64]struct{})

	s.audioCollectionsMutex.Lock()
	defer s.audioCollectionsMutex.Unlock()

	for _, collection := range s.audioCollections {
		if collection == nil {
			continue
		}

		for _, trackID := range collection.trackIDs {
			result[trackID] = struct{}{}
		}
	}

	return result
}
