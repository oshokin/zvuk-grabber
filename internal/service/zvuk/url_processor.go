package zvuk

//go:generate $MOCKGEN -source=url_processor.go -destination=mocks/url_processor_mock.go

import (
	"context"
	"regexp"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// URLProcessor defines the interface for processing URLs and extracting downloadable items.
type URLProcessor interface {
	// ExtractDownloadItems processes a list of URLs and categorizes them into tracks, standalone items, and artists.
	ExtractDownloadItems(ctx context.Context, urls []string) (*ExtractDownloadItemsResponse, error)
	// DeduplicateDownloadItems removes duplicate DownloadItems based on their category and ItemID.
	DeduplicateDownloadItems(items []*DownloadItem) []*DownloadItem
}

// ExtractDownloadItemsResponse represents the result of processing URLs.
// It categorizes the URLs into tracks, standalone items, and artists.
type ExtractDownloadItemsResponse struct {
	// Tracks contains individual track download items.
	Tracks []*DownloadItem
	// StandaloneItems contains album and playlist download items.
	StandaloneItems []*DownloadItem
	// Artists contains artist discography download items.
	Artists []*DownloadItem
}

// URLProcessorImpl implements the URLProcessor interface.
type URLProcessorImpl struct{}

// defaultTextExtension is the default file extension for text files.
const defaultTextExtension = ".txt"

// categoriesByPatterns maps URL patterns to download categories.
//
//nolint:gochecknoglobals,lll // This is a justified global variable: immutable data, performance optimization, and reusability.
var categoriesByPatterns = []struct {
	// Pattern is the regex pattern to match URLs.
	Pattern *regexp.Regexp
	// Category is the download category for matched URLs.
	Category DownloadCategory
}{
	{regexp.MustCompile(`/track/(?<ID>\d+)$`), DownloadCategoryTrack},
	{regexp.MustCompile(`/release/(?<ID>\d+)$`), DownloadCategoryAlbum},
	{regexp.MustCompile(`/playlist/(?<ID>\d+)$`), DownloadCategoryPlaylist},
	{regexp.MustCompile(`/artist/(?<ID>\d+)$`), DownloadCategoryArtist},
}

// NewURLProcessor creates and returns a new instance of URLProcessorImpl.
func NewURLProcessor() URLProcessor {
	return &URLProcessorImpl{}
}

// ExtractDownloadItems processes a list of URLs and categorizes them into tracks, standalone items, and artists.
func (up *URLProcessorImpl) ExtractDownloadItems(
	ctx context.Context,
	urls []string,
) (*ExtractDownloadItemsResponse, error) {
	// Process and flatten URLs to handle text files containing multiple URLs.
	urls, err := up.processAndFlattenURLs(urls)
	if err != nil {
		return nil, err
	}

	var (
		tracks          []*DownloadItem
		standaloneItems []*DownloadItem
		artists         = make([]*DownloadItem, 0, len(urls))
		parsedURLs      = make(map[string]struct{}, len(urls))
	)

	// Iterate through each URL and categorize it.
	for _, url := range urls {
		// Skip already parsed URLs to avoid duplicates.
		if _, ok := parsedURLs[url]; ok {
			continue
		}

		// Parse the URL into a DownloadItem.
		item := up.parseDownloadItem(url)
		parsedURLs[url] = struct{}{}

		// Categorize the item based on its type.
		switch item.Category {
		case DownloadCategoryTrack:
			tracks = append(tracks, item)
		case DownloadCategoryAlbum, DownloadCategoryPlaylist:
			standaloneItems = append(standaloneItems, item)
		case DownloadCategoryArtist:
			artists = append(artists, item)
		case DownloadCategoryUnknown:
			logger.Warnf(ctx, "Unknown URL: %s", url)
		}
	}

	// Return the categorized items.
	return &ExtractDownloadItemsResponse{
		Tracks:          tracks,
		StandaloneItems: standaloneItems,
		Artists:         artists,
	}, nil
}

// DeduplicateDownloadItems removes duplicate DownloadItems based on their category and ItemID.
func (up *URLProcessorImpl) DeduplicateDownloadItems(items []*DownloadItem) []*DownloadItem {
	// Use a map to track unique items.
	uniqueItems := make(map[ShortDownloadItem]struct{}, len(items))
	result := make([]*DownloadItem, 0, len(items))

	// Iterate through items and add only unique ones to the result.
	for _, item := range items {
		key := ShortDownloadItem{Category: item.Category, ItemID: item.ItemID}
		if _, ok := uniqueItems[key]; ok {
			continue
		}

		uniqueItems[key] = struct{}{}

		result = append(result, item)
	}

	return result
}

func (up *URLProcessorImpl) parseDownloadItem(url string) *DownloadItem {
	// Match the URL against each pattern to determine its category.
	for _, p := range categoriesByPatterns {
		if itemID := utils.ExtractNamedGroup(p.Pattern, "ID", url); itemID != "" {
			return &DownloadItem{Category: p.Category, URL: url, ItemID: itemID}
		}
	}

	// If no pattern matches, return an item with an unknown category.
	return &DownloadItem{
		Category: DownloadCategoryUnknown,
		URL:      url,
		ItemID:   "",
	}
}

func (up *URLProcessorImpl) processAndFlattenURLs(urls []string) ([]string, error) {
	var (
		// Track processed URLs.
		processedSet = make(map[string]struct{})
		// Track processed text files.
		processedTextFiles = make(map[string]struct{})
		// Store the final list of URLs.
		processedURLs []string
	)

	// Iterate through each URL.
	for _, url := range urls {
		// If the URL is not a text file, add it directly to the processed list.
		if !strings.HasSuffix(url, defaultTextExtension) {
			if _, ok := processedSet[url]; ok {
				continue
			}

			processedSet[url] = struct{}{}

			processedURLs = append(processedURLs, url)

			continue
		}

		// Skip already processed text files.
		if _, exists := processedTextFiles[url]; exists {
			continue
		}

		// Read unique lines from the text file.
		lines, err := utils.ReadUniqueLinesFromFile(url)
		if err != nil {
			return nil, err
		}

		// Add each line (URL) from the text file to the processed list.
		for _, line := range lines {
			if _, ok := processedSet[line]; ok {
				continue
			}

			processedSet[line] = struct{}{}

			processedURLs = append(processedURLs, line)
		}

		// Mark the text file as processed.
		processedTextFiles[url] = struct{}{}
	}

	return processedURLs, nil
}
