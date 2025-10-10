package zvuk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestNewTemplateManager tests the NewTemplateManager function.
func TestNewTemplateManager(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
	}

	manager := NewTemplateManager(ctx, cfg)
	assert.NotNil(t, manager)
	assert.Implements(t, (*TemplateManager)(nil), manager)
}

// TestNewTemplateManager_InvalidTemplate tests NewTemplateManager with invalid templates.
func TestNewTemplateManager_InvalidTemplate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.invalidTemplate", // invalid template
		AlbumFolderTemplate:      "{{.invalidTemplate", // invalid template
		PlaylistFilenameTemplate: "{{.invalidTemplate", // invalid template
	}

	manager := NewTemplateManager(ctx, cfg)
	assert.NotNil(t, manager)

	// Test track filename. with default template.
	trackTags := map[string]string{
		"trackNumberPad": "01",
		"trackTitle":     "Test Track",
	}
	result := manager.GetTrackFilename(ctx, false, trackTags, 1)
	t.Logf("Generated track filename: %s", result)
	// Should use default template.
	assert.NotEmpty(t, result)
	// Test album folder name with default template.
	albumTags := map[string]string{
		"releaseYear": "2023",
		"albumArtist": "Test Artist",
		"albumTitle":  "Test Album",
	}
	result = manager.GetAlbumFolderName(ctx, albumTags)
	t.Logf("Generated album folder name: %s", result)
	assert.NotEmpty(t, result) // Should use default template
}

// TestTemplateManagerImpl_DefaultTemplates tests with default templates.
func TestTemplateManagerImpl_DefaultTemplates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
		CreateFolderForSingles:   false,
	}

	templateManager := NewTemplateManager(ctx, cfg)

	// Test track filename.
	trackTags := map[string]string{
		"trackTitle":     "Test Track",
		"trackNumberPad": "01",
		"trackArtist":    "Test Artist",
	}
	filename := templateManager.GetTrackFilename(ctx, false, trackTags, 5) // Multiple tracks, not a single
	t.Logf("Generated filename: %s", filename)
	assert.NotEmpty(t, filename)
	assert.Contains(t, filename, "Test Track")
}

// TestTemplateManagerImpl_EdgeCases tests edge cases.
func TestTemplateManagerImpl_EdgeCases(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Test with valid config but empty tags.
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
		CreateFolderForSingles:   false,
	}
	manager := NewTemplateManager(ctx, cfg)
	assert.NotNil(t, manager)

	// Test track filename with empty tags.
	result := manager.GetTrackFilename(ctx, false, map[string]string{}, 1)
	t.Logf("Generated filename with empty tags: %s", result)
	assert.NotEmpty(t, result)

	// Test album folder name with empty tags.
	result = manager.GetAlbumFolderName(ctx, map[string]string{})
	t.Logf("Generated folder name with empty tags: %s", result)
	assert.NotEmpty(t, result)
}

// TestTemplateManagerImpl_GetAlbumFolderName tests the GetAlbumFolderName method.
func TestTemplateManagerImpl_GetAlbumFolderName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tags     map[string]string
		expected string
	}{
		{
			name: "complete album tags",
			tags: map[string]string{
				"releaseYear": "2023",
				"albumArtist": "Test Artist",
				"albumTitle":  "Test Album",
			},
			expected: "2023 - Test Artist - Test Album",
		},
		{
			name: "missing year",
			tags: map[string]string{
				"albumArtist": "Test Artist",
				"albumTitle":  "Test Album",
			},
			expected: " - Test Artist - Test Album",
		},
		{
			name: "missing artist",
			tags: map[string]string{
				"releaseYear": "2023",
				"albumTitle":  "Test Album",
			},
			expected: "2023 -  - Test Album",
		},
		{
			name: "missing title",
			tags: map[string]string{
				"releaseYear": "2023",
				"albumArtist": "Test Artist",
			},
			expected: "2023 - Test Artist - ",
		},
		{
			name:     "empty tags",
			tags:     map[string]string{},
			expected: " -  - ",
		},
		{
			name: "special characters in tags",
			tags: map[string]string{
				"releaseYear": "2023",
				"albumArtist": "Artist/With\\Special:Chars",
				"albumTitle":  "Album|With*Special?Chars",
			},
			expected: "2023 - Artist/With\\Special:Chars - Album|With*Special?Chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cfg := &config.Config{
				AlbumFolderTemplate: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
			}
			manager := NewTemplateManager(ctx, cfg)

			result := manager.GetAlbumFolderName(ctx, tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTemplateManagerImpl_GetTrackFilename tests the GetTrackFilename method.
func TestTemplateManagerImpl_GetTrackFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		isPlaylist  bool
		trackTags   map[string]string
		tracksCount int64
		expected    string
	}{
		{
			name:       "regular track",
			isPlaylist: false,
			trackTags: map[string]string{
				"trackNumberPad": "01",
				"trackTitle":     "Test Track",
			},
			tracksCount: 5,
			expected:    "01 - Test Track",
		},
		{
			name:       "playlist track",
			isPlaylist: true,
			trackTags: map[string]string{
				"trackNumberPad": "01",
				"trackArtist":    "Test Artist",
				"trackTitle":     "Test Track",
			},
			tracksCount: 5,
			expected:    "01 - Test Artist - Test Track",
		},
		{
			name:       "single track",
			isPlaylist: false,
			trackTags: map[string]string{
				"trackNumberPad": "01",
				"trackArtist":    "Test Artist",
				"trackTitle":     "Test Track",
			},
			tracksCount: 1,
			expected:    "01 - Test Artist - Test Track", // Uses playlist template for singles
		},
		{
			name:       "track with missing tags",
			isPlaylist: false,
			trackTags: map[string]string{
				"trackNumberPad": "01",
			},
			tracksCount: 5,
			expected:    "01 - ",
		},
		{
			name:        "empty tags",
			isPlaylist:  false,
			trackTags:   map[string]string{},
			tracksCount: 5,
			expected:    " - ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cfg := &config.Config{
				TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
				PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
				CreateFolderForSingles:   false,
			}
			manager := NewTemplateManager(ctx, cfg)

			result := manager.GetTrackFilename(ctx, tt.isPlaylist, tt.trackTags, tt.tracksCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTemplateManagerImpl_LargeNumbers tests with large track numbers.
func TestTemplateManagerImpl_LargeNumbers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate: "{{.trackNumberPad}} - {{.trackTitle}}",
	}
	manager := NewTemplateManager(ctx, cfg)

	trackTags := map[string]string{
		"trackNumberPad": "999",
		"trackTitle":     "Track 999",
	}

	result := manager.GetTrackFilename(ctx, false, trackTags, 1000)
	assert.Contains(t, result, "999")
	assert.Contains(t, result, "Track 999")
}

// TestTemplateManagerImpl_UnicodeCharacters tests with Unicode characters.
func TestTemplateManagerImpl_UnicodeCharacters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.trackTitle}}",
		AlbumFolderTemplate:      "{{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackTitle}}",
		CreateFolderForSingles:   true,
	}
	manager := NewTemplateManager(ctx, cfg)

	// Test Unicode characters in track title.
	trackTags := map[string]string{
		"trackTitle": "Тест Трек 🎵",
	}
	result := manager.GetTrackFilename(ctx, false, trackTags, 1)
	assert.Contains(t, result, "Тест Трек 🎵")

	// Test Unicode characters in album title.
	albumTags := map[string]string{
		"albumTitle": "Альбом 🎶",
	}
	result = manager.GetAlbumFolderName(ctx, albumTags)
	assert.Contains(t, result, "Альбом 🎶")
}
