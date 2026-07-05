package media

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestNewTemplateManager verifies NewTemplateManager returns a configured template manager.
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

// TestTemplateManager_GetTrackFilename verifies track filename rendering for albums and playlists.
func TestTemplateManager_GetTrackFilename(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
		CreateFolderForSingles:   false,
	}

	manager := NewTemplateManager(ctx, cfg)
	tags := map[string]string{
		TagTrackTitle:     "Test Track",
		TagTrackNumberPad: "01",
		TagTrackArtist:    "Test Artist",
	}

	name := manager.GetTrackFilename(ctx, false, tags, 10)
	assert.Equal(t, "01 - Test Track", name)

	playlistName := manager.GetTrackFilename(ctx, true, tags, 10)
	assert.Equal(t, "01 - Test Artist - Test Track", playlistName)
}

// TestTemplateManager_GetAlbumFolderName verifies album folder name rendering from templates.
func TestTemplateManager_GetAlbumFolderName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
	}

	manager := NewTemplateManager(ctx, cfg)
	tags := map[string]string{
		TagReleaseYear: "2024",
		TagAlbumArtist: "Artist",
		TagAlbumTitle:  "Album",
	}

	folder := manager.GetAlbumFolderName(ctx, tags)
	assert.Equal(t, "2024 - Artist - Album", folder)
}
