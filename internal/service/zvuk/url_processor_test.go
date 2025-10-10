package zvuk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewURLProcessor tests the NewURLProcessor function.
func TestNewURLProcessor(t *testing.T) {
	t.Parallel()

	processor := NewURLProcessor()
	assert.NotNil(t, processor)
	assert.Implements(t, (*URLProcessor)(nil), processor)
}

// TestURLPatterns tests URL pattern matching.
func TestURLPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected DownloadCategory
	}{
		{
			name:     "track URL",
			url:      "https://zvuk.com/track/123",
			expected: DownloadCategoryTrack,
		},
		{
			name:     "release URL",
			url:      "https://zvuk.com/release/456",
			expected: DownloadCategoryAlbum,
		},
		{
			name:     "playlist URL",
			url:      "https://zvuk.com/playlist/789",
			expected: DownloadCategoryPlaylist,
		},
		{
			name:     "artist URL",
			url:      "https://zvuk.com/artist/101",
			expected: DownloadCategoryArtist,
		},
		{
			name:     "URL with trailing slash",
			url:      "https://zvuk.com/track/123/",
			expected: DownloadCategoryUnknown, // Doesn't match due to trailing slash
		},
		{
			name:     "URL with additional path",
			url:      "https://zvuk.com/track/123/details",
			expected: DownloadCategoryUnknown, // Doesn't match due to additional path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewURLProcessor()
			ctx := context.Background()

			result, err := processor.ExtractDownloadItems(ctx, []string{tt.url})
			require.NoError(t, err)
			assert.NotNil(t, result)

			switch tt.expected {
			case DownloadCategoryTrack:
				assert.Len(t, result.Tracks, 1)
				assert.Equal(t, tt.expected, result.Tracks[0].Category)
			case DownloadCategoryAlbum, DownloadCategoryPlaylist:
				assert.Len(t, result.StandaloneItems, 1)
				assert.Equal(t, tt.expected, result.StandaloneItems[0].Category)
			case DownloadCategoryArtist:
				assert.Len(t, result.Artists, 1)
				assert.Equal(t, tt.expected, result.Artists[0].Category)
			default:
				// Unknown category - should not appear in any result slice.
				assert.Empty(t, result.Tracks)
				assert.Empty(t, result.StandaloneItems)
				assert.Empty(t, result.Artists)
			}
		})
	}
}

// TestURLProcessorImpl_DeduplicateDownloadItems tests the DeduplicateDownloadItems method.
func TestURLProcessorImpl_DeduplicateDownloadItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		items    []*DownloadItem
		expected []*DownloadItem
	}{
		{
			name:     "empty items",
			items:    []*DownloadItem{},
			expected: []*DownloadItem{},
		},
		{
			name: "no duplicates",
			items: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryAlbum, ItemID: "2"},
			},
			expected: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryAlbum, ItemID: "2"},
			},
		},
		{
			name: "with duplicates",
			items: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryAlbum, ItemID: "2"},
			},
			expected: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryAlbum, ItemID: "2"},
			},
		},
		{
			name: "same category different IDs",
			items: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryTrack, ItemID: "2"},
			},
			expected: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryTrack, ItemID: "2"},
			},
		},
		{
			name: "different categories same ID",
			items: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryAlbum, ItemID: "1"},
			},
			expected: []*DownloadItem{
				{Category: DownloadCategoryTrack, ItemID: "1"},
				{Category: DownloadCategoryAlbum, ItemID: "1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewURLProcessor()
			result := processor.DeduplicateDownloadItems(tt.items)
			assert.Len(t, result, len(tt.expected))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.Category, result[i].Category)
				assert.Equal(t, expected.ItemID, result[i].ItemID)
			}
		})
	}
}

// TestURLProcessorImpl_ExtractDownloadItems tests the ExtractDownloadItems method.
func TestURLProcessorImpl_ExtractDownloadItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		urls     []string
		expected *ExtractDownloadItemsResponse
	}{
		{
			name: "empty URLs",
			urls: []string{},
			expected: &ExtractDownloadItemsResponse{
				Tracks:          []*DownloadItem{},
				StandaloneItems: []*DownloadItem{},
				Artists:         []*DownloadItem{},
			},
		},
		{
			name: "track URLs",
			urls: []string{
				"https://zvuk.com/track/123",
				"https://zvuk.com/track/456",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks: []*DownloadItem{
					{Category: DownloadCategoryTrack, URL: "https://zvuk.com/track/123", ItemID: "123"},
					{Category: DownloadCategoryTrack, URL: "https://zvuk.com/track/456", ItemID: "456"},
				},
				StandaloneItems: []*DownloadItem{},
				Artists:         []*DownloadItem{},
			},
		},
		{
			name: "album URLs",
			urls: []string{
				"https://zvuk.com/release/123",
				"https://zvuk.com/release/456",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks: []*DownloadItem{},
				StandaloneItems: []*DownloadItem{
					{Category: DownloadCategoryAlbum, URL: "https://zvuk.com/release/123", ItemID: "123"},
					{Category: DownloadCategoryAlbum, URL: "https://zvuk.com/release/456", ItemID: "456"},
				},
				Artists: []*DownloadItem{},
			},
		},
		{
			name: "playlist URLs",
			urls: []string{
				"https://zvuk.com/playlist/123",
				"https://zvuk.com/playlist/456",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks: []*DownloadItem{},
				StandaloneItems: []*DownloadItem{
					{Category: DownloadCategoryPlaylist, URL: "https://zvuk.com/playlist/123", ItemID: "123"},
					{Category: DownloadCategoryPlaylist, URL: "https://zvuk.com/playlist/456", ItemID: "456"},
				},
				Artists: []*DownloadItem{},
			},
		},
		{
			name: "artist URLs",
			urls: []string{
				"https://zvuk.com/artist/123",
				"https://zvuk.com/artist/456",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks:          []*DownloadItem{},
				StandaloneItems: []*DownloadItem{},
				Artists: []*DownloadItem{
					{Category: DownloadCategoryArtist, URL: "https://zvuk.com/artist/123", ItemID: "123"},
					{Category: DownloadCategoryArtist, URL: "https://zvuk.com/artist/456", ItemID: "456"},
				},
			},
		},
		{
			name: "mixed URLs",
			urls: []string{
				"https://zvuk.com/track/123",
				"https://zvuk.com/release/456",
				"https://zvuk.com/playlist/789",
				"https://zvuk.com/artist/101",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks: []*DownloadItem{
					{Category: DownloadCategoryTrack, URL: "https://zvuk.com/track/123", ItemID: "123"},
				},
				StandaloneItems: []*DownloadItem{
					{Category: DownloadCategoryAlbum, URL: "https://zvuk.com/release/456", ItemID: "456"},
					{Category: DownloadCategoryPlaylist, URL: "https://zvuk.com/playlist/789", ItemID: "789"},
				},
				Artists: []*DownloadItem{
					{Category: DownloadCategoryArtist, URL: "https://zvuk.com/artist/101", ItemID: "101"},
				},
			},
		},
		{
			name: "unknown URLs",
			urls: []string{
				"https://zvuk.com/unknown/123",
				"https://example.com/invalid/path",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks:          []*DownloadItem{},
				StandaloneItems: []*DownloadItem{},
				Artists:         []*DownloadItem{},
			},
		},
		{
			name: "URLs with query parameters",
			urls: []string{
				"https://zvuk.com/track/123?param=value",
				"https://zvuk.com/release/456?utm_source=test",
			},
			expected: &ExtractDownloadItemsResponse{
				Tracks:          []*DownloadItem{},
				StandaloneItems: []*DownloadItem{},
				Artists:         []*DownloadItem{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewURLProcessor()
			ctx := context.Background()

			result, err := processor.ExtractDownloadItems(ctx, tt.urls)
			require.NoError(t, err)
			assert.NotNil(t, result)

			assert.Len(t, result.Tracks, len(tt.expected.Tracks))
			assert.Len(t, result.StandaloneItems, len(tt.expected.StandaloneItems))
			assert.Len(t, result.Artists, len(tt.expected.Artists))

			// Check tracks.
			for i, expectedTrack := range tt.expected.Tracks {
				assert.Equal(t, expectedTrack.Category, result.Tracks[i].Category)
				assert.Equal(t, expectedTrack.ItemID, result.Tracks[i].ItemID)
			}

			// Check standalone items.
			for i, expectedItem := range tt.expected.StandaloneItems {
				assert.Equal(t, expectedItem.Category, result.StandaloneItems[i].Category)
				assert.Equal(t, expectedItem.ItemID, result.StandaloneItems[i].ItemID)
			}

			// Check artists.
			for i, expectedArtist := range tt.expected.Artists {
				assert.Equal(t, expectedArtist.Category, result.Artists[i].Category)
				assert.Equal(t, expectedArtist.ItemID, result.Artists[i].ItemID)
			}
		})
	}
}
