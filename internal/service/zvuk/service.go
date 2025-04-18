package zvuk

import (
	"context"
	"os"
	"strconv"
	"sync"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Service provides methods for downloading audio content from Zvuk URLs.
type Service interface {
	// DownloadURLs orchestrates the full download pipeline, from URL processing to file creation.
	DownloadURLs(ctx context.Context, urls []string)
}

// ServiceImpl implements audio download service with deduplication and metadata handling.
type ServiceImpl struct {
	cfg                   *config.Config
	zvukClient            zvuk.Client
	urlProcessor          URLProcessor
	templateManager       TemplateManager
	tagProcessor          TagProcessor
	audioCollections      map[ShortDownloadItem]*audioCollection
	audioCollectionsMutex *sync.Mutex
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
	}
}

// DownloadURLs orchestrates the full download pipeline, from URL processing to file creation.
func (s *ServiceImpl) DownloadURLs(ctx context.Context, urls []string) {
	// Ensure the output directory exists
	err := os.MkdirAll(s.cfg.OutputPath, defaultFolderPermissions)
	if err != nil {
		logger.Fatalf(ctx, "Failed to create output path: %v", err)
	}

	// Verify the user's subscription status before proceeding
	s.checkUserSubscription(ctx)

	// Extract and categorize download items from the provided URLs
	downloadItemsByCategories, err := s.urlProcessor.ExtractDownloadItems(ctx, urls)
	if err != nil {
		logger.Fatalf(ctx, "Failed to extract items to download: %v", err)

		return
	}

	logger.Info(ctx, "Starting download process")

	// Process albums and playlists first to maintain organizational structure
	standaloneItems := s.fetchAndDeduplicateStandaloneItems(ctx, downloadItemsByCategories)
	if len(standaloneItems) > 0 {
		s.downloadStandaloneItems(ctx, standaloneItems)
	}

	// Process individual tracks after collections to allow potential deduplication
	if len(downloadItemsByCategories.Tracks) > 0 {
		s.downloadTrackItems(ctx, downloadItemsByCategories.Tracks)
	}

	logger.Info(ctx, "Download process completed")
}

// fetchAndDeduplicateStandaloneItems processes artist URLs to fetch their albums and removes duplicate entries.
func (s *ServiceImpl) fetchAndDeduplicateStandaloneItems(
	ctx context.Context,
	items *ExtractDownloadItemsResponse,
) []*DownloadItem {
	standaloneItems := items.StandaloneItems

	// If artist URLs are present, fetch their albums and append them to the standalone items
	if len(items.Artists) > 0 {
		artistAlbums := s.fetchArtistAlbums(ctx, items.Artists)
		standaloneItems = append(standaloneItems, artistAlbums...)
		// Remove duplicate album entries that might exist in the original URLs
		standaloneItems = s.urlProcessor.DeduplicateDownloadItems(standaloneItems)
	}

	return standaloneItems
}

// downloadStandaloneItems handles the download of albums and playlists.
func (s *ServiceImpl) downloadStandaloneItems(ctx context.Context, items []*DownloadItem) {
	logger.Info(ctx, "Downloading albums and playlists")

	itemsCount := len(items)

	// Iterate through each item and download based on its category
	for index, item := range items {
		//nolint:exhaustive // All meaningful cases are explicitly handled; default covers unknown values.
		switch item.Category {
		case DownloadCategoryAlbum:
			logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)
			s.downloadAlbum(ctx, item.ItemID)
		case DownloadCategoryPlaylist:
			logger.Infof(ctx, "Downloading item: %v (%d / %d)", item, index+1, itemsCount)
			s.downloadPlaylist(ctx, item.ItemID)
		default:
			logger.Errorf(ctx, "Unknown URL category: %d", item.Category)
		}
	}
}

// downloadTrackItems handles the download of individual tracks.
func (s *ServiceImpl) downloadTrackItems(ctx context.Context, items []*DownloadItem) {
	logger.Info(ctx, "Downloading tracks")

	// Convert track IDs from strings to integers
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

	// Fetch metadata for the tracks
	tracksMetadata, err := s.zvukClient.GetTracksMetadata(ctx, trackIDs)
	if err != nil {
		logger.Errorf(ctx, "Failed to get track metadata: %v", err)

		return
	}

	// Fetch album and label metadata for the tracks
	fetchAlbumsDataFromTracksResponse, err := s.fetchAlbumsDataFromTracks(ctx, tracksMetadata)
	if err != nil {
		logger.Errorf(ctx, "Failed to fetch album and label metadata: %v", err)

		return
	}

	// Prepare metadata for downloading the tracks
	metadata := &downloadTracksMetadata{
		category:        DownloadCategoryTrack,
		trackIDs:        numericTrackIDs,
		tracksMetadata:  tracksMetadata,
		albumsMetadata:  fetchAlbumsDataFromTracksResponse.releases,
		albumsTags:      fetchAlbumsDataFromTracksResponse.releasesTags,
		labelsMetadata:  fetchAlbumsDataFromTracksResponse.labels,
		audioCollection: nil,
	}

	// Download the tracks
	s.downloadTracks(ctx, metadata)
}
