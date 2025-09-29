package zvuk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDownloadCategory tests the DownloadCategory enum and String method.
func TestDownloadCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category DownloadCategory
		expected string
	}{
		{
			name:     "unknown category",
			category: DownloadCategoryUnknown,
			expected: "unknown",
		},
		{
			name:     "track category",
			category: DownloadCategoryTrack,
			expected: "track",
		},
		{
			name:     "album category",
			category: DownloadCategoryAlbum,
			expected: "album",
		},
		{
			name:     "playlist category",
			category: DownloadCategoryPlaylist,
			expected: "playlist",
		},
		{
			name:     "artist category",
			category: DownloadCategoryArtist,
			expected: "artist",
		},
		{
			name:     "invalid category",
			category: DownloadCategory(255), // Use a valid uint8 value
			expected: "unknown: 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.category.String())
		})
	}
}

// TestDownloadItem tests the DownloadItem structure.
func TestDownloadItem(t *testing.T) {
	t.Parallel()

	item := &DownloadItem{
		Category: DownloadCategoryTrack,
		URL:      "https://zvuk.com/track/123",
		ItemID:   "123",
	}

	assert.Equal(t, DownloadCategoryTrack, item.Category)
	assert.Equal(t, "https://zvuk.com/track/123", item.URL)
	assert.Equal(t, "123", item.ItemID)
}

// TestDownloadItem_GetShortVersion tests the GetShortVersion method.
func TestDownloadItem_GetShortVersion(t *testing.T) {
	t.Parallel()

	item := &DownloadItem{
		Category: DownloadCategoryTrack,
		URL:      "https://zvuk.com/track/789",
		ItemID:   "789",
	}

	shortItem := item.GetShortVersion()
	assert.Equal(t, DownloadCategoryTrack, shortItem.Category)
	assert.Equal(t, "789", shortItem.ItemID)
}

// TestShortDownloadItem tests the ShortDownloadItem structure.
func TestShortDownloadItem(t *testing.T) {
	t.Parallel()

	shortItem := &ShortDownloadItem{
		Category: DownloadCategoryAlbum,
		ItemID:   "456",
	}

	assert.Equal(t, DownloadCategoryAlbum, shortItem.Category)
	assert.Equal(t, "456", shortItem.ItemID)
}

// TestExtractDownloadItemsResponse tests the ExtractDownloadItemsResponse structure.
func TestExtractDownloadItemsResponse(t *testing.T) {
	t.Parallel()

	track := &DownloadItem{
		Category: DownloadCategoryTrack,
		URL:      "https://zvuk.com/track/123",
		ItemID:   "123",
	}

	album := &DownloadItem{
		Category: DownloadCategoryAlbum,
		URL:      "https://zvuk.com/release/456",
		ItemID:   "456",
	}

	artist := &DownloadItem{
		Category: DownloadCategoryArtist,
		URL:      "https://zvuk.com/artist/789",
		ItemID:   "789",
	}

	response := &ExtractDownloadItemsResponse{
		Tracks:          []*DownloadItem{track},
		StandaloneItems: []*DownloadItem{album},
		Artists:         []*DownloadItem{artist},
	}

	assert.Len(t, response.Tracks, 1)
	assert.Len(t, response.StandaloneItems, 1)
	assert.Len(t, response.Artists, 1)
	assert.Equal(t, DownloadCategoryTrack, response.Tracks[0].Category)
	assert.Equal(t, DownloadCategoryAlbum, response.StandaloneItems[0].Category)
	assert.Equal(t, DownloadCategoryArtist, response.Artists[0].Category)
}

// TestImageMetadata tests the imageMetadata structure.
func TestImageMetadata(t *testing.T) {
	t.Parallel()

	image := &imageMetadata{
		data:     []byte("test image data"),
		mimeType: "image/jpeg",
	}

	assert.Equal(t, []byte("test image data"), image.data)
	assert.Equal(t, "image/jpeg", image.mimeType)
}

// TestWriteTagsRequest tests the WriteTagsRequest structure.
func TestWriteTagsRequest(t *testing.T) {
	t.Parallel()

	tags := map[string]string{
		"title":  "Test Track",
		"artist": "Test Artist",
		"album":  "Test Album",
	}

	request := &WriteTagsRequest{
		TrackPath: "/path/to/file.mp3",
		TrackTags: tags,
		CoverPath: "/path/to/cover.jpg",
		Quality:   TrackQualityMP3Mid,
	}

	assert.Equal(t, "/path/to/file.mp3", request.TrackPath)
	assert.Equal(t, tags, request.TrackTags)
	assert.Equal(t, "Test Track", request.TrackTags["title"])
	assert.Equal(t, "Test Artist", request.TrackTags["artist"])
	assert.Equal(t, "Test Album", request.TrackTags["album"])
}

// TestAudioCollection tests the audioCollection structure.
func TestAudioCollection(t *testing.T) {
	t.Parallel()

	collection := &audioCollection{
		category:    DownloadCategoryTrack,
		title:       "Test Collection",
		tags:        make(map[string]string),
		tracksPath:  "/path/to/tracks",
		coverPath:   "/path/to/cover",
		trackIDs:    []int64{1, 2, 3},
		tracksCount: 3,
	}

	assert.Equal(t, DownloadCategoryTrack, collection.category)
	assert.Equal(t, "Test Collection", collection.title)
	assert.NotNil(t, collection.tags)
	assert.Equal(t, "/path/to/tracks", collection.tracksPath)
	assert.Equal(t, "/path/to/cover", collection.coverPath)
	assert.Len(t, collection.trackIDs, 3)
	assert.Equal(t, int64(3), collection.tracksCount)
}

// TestConstants tests the constants.
func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DownloadCategoryUnknown, DownloadCategory(0))
	assert.Equal(t, DownloadCategoryTrack, DownloadCategory(1))
	assert.Equal(t, DownloadCategoryAlbum, DownloadCategory(2))
	assert.Equal(t, DownloadCategoryPlaylist, DownloadCategory(3))
	assert.Equal(t, DownloadCategoryArtist, DownloadCategory(4))
}
