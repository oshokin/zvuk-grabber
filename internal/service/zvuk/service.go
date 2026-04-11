package zvuk

//go:generate $MOCKGEN -source=service.go -destination=mocks/service_mock.go

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// Service provides methods for downloading audio content from Zvuk URLs.
type Service interface {
	// DownloadURLs orchestrates the full download pipeline, from URL processing to file creation.
	DownloadURLs(ctx context.Context, urls []string)
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
	templateManager TemplateManager
	// tagProcessor writes metadata tags to audio files.
	tagProcessor TagProcessor
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
	// filePathLocks serializes writes to the same destination path.
	filePathLocks map[string]*pathLock
	// filePathLocksMutex protects concurrent access to filePathLocks.
	filePathLocksMutex sync.Mutex
}

// NewService creates a download service instance with dependency-injected components.
func NewService(
	cfg *config.Config,
	zvukClient zvuk.Client,
	urlProcessor URLProcessor,
	templateManager TemplateManager,
	tagProcessor TagProcessor,
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
		filePathLocks:         make(map[string]*pathLock),
	}

	return s
}

// DownloadURLs orchestrates the full download pipeline, from URL processing to file creation.
func (s *ServiceImpl) DownloadURLs(ctx context.Context, urls []string) {
	// Record start time and dry-run mode for statistics.
	s.statsMutex.Lock()
	s.stats.StartTime = time.Now()
	s.stats.IsDryRun = s.cfg.DryRun
	s.statsMutex.Unlock()

	// Ensure the output directory exists (skip in dry-run mode).
	if !s.cfg.DryRun {
		err := os.MkdirAll(s.cfg.OutputPath, defaultFolderPermissions)
		if err != nil {
			logger.Errorf(ctx, "Failed to create output path: %v", err)
			return
		}
	} else {
		logger.Infof(ctx, "[DRY-RUN] Would create output directory: %s", s.cfg.OutputPath)
	}

	// Verify the user's subscription status before proceeding.
	s.checkUserSubscription(ctx)

	// Extract and categorize download items from the provided URLs.
	downloadItemsByCategories, err := s.urlProcessor.ExtractDownloadItems(ctx, urls)
	if err != nil {
		logger.Errorf(ctx, "Failed to extract items to download: %v", err)
		return
	}

	logger.Info(ctx, "Starting download process")

	// Process albums and playlists first to maintain organizational structure.
	standaloneItems := s.fetchAndDeduplicateStandaloneItems(ctx, downloadItemsByCategories)
	if len(standaloneItems) > 0 {
		s.downloadStandaloneItems(ctx, standaloneItems)
	}

	// Process individual tracks after collections to allow potential deduplication.
	if len(downloadItemsByCategories.Tracks) > 0 {
		s.downloadTrackItems(ctx, downloadItemsByCategories.Tracks)
	}

	logger.Info(ctx, "Download process completed")

	// Record end time for statistics.
	s.statsMutex.Lock()
	s.stats.EndTime = time.Now()
	s.statsMutex.Unlock()
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

func (s *ServiceImpl) recordStandaloneTrackBatchError(trackIDs []string, phase string, err error) {
	for _, trackID := range trackIDs {
		s.recordStandaloneTrackError(trackID, phase, err)
	}
}

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
