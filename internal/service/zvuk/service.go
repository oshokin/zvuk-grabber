package zvuk

//go:generate $MOCKGEN -source=service.go -destination=mocks/service_mock.go

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
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
	// stats tracks download statistics for the current session.
	stats *DownloadStatistics
	// statsMutex protects concurrent access to statistics.
	statsMutex *sync.Mutex
}

// NewService creates a download service instance with dependency-injected components.
func NewService(
	cfg *config.Config,
	zvukClient zvuk.Client,
	urlProcessor URLProcessor,
	templateManager TemplateManager,
	tagProcessor TagProcessor,
) Service {
	return &ServiceImpl{
		cfg:                   cfg,
		zvukClient:            zvukClient,
		urlProcessor:          urlProcessor,
		templateManager:       templateManager,
		tagProcessor:          tagProcessor,
		audioCollections:      make(map[ShortDownloadItem]*audioCollection),
		audioCollectionsMutex: new(sync.Mutex),
		stats:                 new(DownloadStatistics),
		statsMutex:            new(sync.Mutex),
	}
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

		//nolint:exhaustive // All meaningful cases are explicitly handled; default covers unknown values.
		switch item.Category {
		case DownloadCategoryAlbum:
			logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)
			s.downloadAlbum(ctx, item)
		case DownloadCategoryPlaylist:
			logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)
			s.downloadPlaylist(ctx, item)
		case DownloadCategoryAudiobook:
			logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)
			s.downloadAudiobook(ctx, item)
		case DownloadCategoryPodcast:
			logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)
			s.downloadPodcast(ctx, item)
		default:
			logger.Errorf(ctx, "Unknown URL category: %d", item.Category)
		}
	}
}

// downloadTrackItems handles the download of individual tracks.
func (s *ServiceImpl) downloadTrackItems(ctx context.Context, items []*DownloadItem) {
	logger.Info(ctx, "Downloading tracks")

	// Convert track IDs from strings to integers.
	numericTrackIDs := make([]int64, 0, len(items))
	trackIDs := utils.Map(items, func(v *DownloadItem) string { return v.ItemID })

	for _, trackIDString := range trackIDs {
		trackID, err := strconv.ParseInt(trackIDString, 10, 64)
		if err != nil {
			logger.Errorf(ctx, "Failed to parse track ID '%s': %v", trackID, err)

			return
		}

		numericTrackIDs = append(numericTrackIDs, trackID)
	}

	// Fetch metadata for the tracks.
	tracksMetadata, err := s.zvukClient.GetTracksMetadata(ctx, trackIDs)
	if err != nil {
		logger.Errorf(ctx, "Failed to get track metadata: %v", err)

		return
	}

	// Fetch album and label metadata for the tracks.
	fetchAlbumsDataFromTracksResponse, err := s.fetchAlbumsDataFromTracks(ctx, tracksMetadata)
	if err != nil {
		logger.Errorf(ctx, "Failed to fetch album and label metadata: %v", err)

		return
	}

	// Prepare metadata for downloading the tracks.
	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryTrack,
		trackIDs:        numericTrackIDs,
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  fetchAlbumsDataFromTracksResponse.releases,
		albumsTags:      fetchAlbumsDataFromTracksResponse.releasesTags,
		labelsMetadata:  fetchAlbumsDataFromTracksResponse.labels,
		audioCollection: nil,
	}

	// Download the tracks.
	s.downloadTracks(ctx, metadata)
}
