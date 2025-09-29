package zvuk

//go:generate $MOCKGEN -source=template_manager.go -destination=mocks/template_manager_mock.go

import (
	"bytes"
	"context"
	"html"
	"html/template"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TemplateManager defines the interface for managing templates used to generate filenames and folder names.
type TemplateManager interface {
	// GetTrackFilename generates a filename for a track based on its tags and context.
	// If the track is part of a playlist or a single without a folder, it uses a different template.
	GetTrackFilename(ctx context.Context, isPlaylist bool, trackTags map[string]string, tracksCount int64) string

	// GetAlbumFolderName generates a folder name for an album based on its tags.
	GetAlbumFolderName(ctx context.Context, tags map[string]string) string
}

// TemplateManagerImpl implements the TemplateManager interface.
type TemplateManagerImpl struct {
	cfg                             *config.Config
	trackFilenameTemplate           *template.Template
	albumFolderTemplate             *template.Template
	playlistFilenameTemplate        *template.Template
	defaultTrackFilenameTemplate    *template.Template
	defaultAlbumFolderTemplate      *template.Template
	defaultPlaylistFilenameTemplate *template.Template
}

// NewTemplateManager creates and returns a new instance of TemplateManagerImpl.
// It initializes templates from the configuration and falls back to default templates if parsing fails.
func NewTemplateManager(ctx context.Context, cfg *config.Config) TemplateManager {
	// Initialize default templates.
	defaultTrackFilenameTemplate := template.Must(
		template.New("defaultTrackFilenameTemplate").Parse(config.DefaultTrackFilenameTemplate))
	defaultAlbumFolderTemplate := template.Must(
		template.New("defaultAlbumFolderTemplate").Parse(config.DefaultAlbumFolderTemplate))
	defaultPlaylistFilenameTemplate := template.Must(
		template.New("defaultPlaylistFilenameTemplate").Parse(config.DefaultPlaylistFilenameTemplate))

	// Parse custom templates from the configuration.
	trackFilenameTemplate, err := template.New("trackFilenameTemplate").Parse(cfg.TrackFilenameTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse track filename template, using default: %v", err)
	}

	albumFolderTemplate, err := template.New("albumFolderTemplate").Parse(cfg.AlbumFolderTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse album folder template, using default: %v", err)
	}

	playlistFilenameTemplate, err := template.New("playlistFilenameTemplate").Parse(cfg.PlaylistFilenameTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse playlist filename template, using default: %v", err)
	}

	return &TemplateManagerImpl{
		cfg:                             cfg,
		trackFilenameTemplate:           trackFilenameTemplate,
		albumFolderTemplate:             albumFolderTemplate,
		playlistFilenameTemplate:        playlistFilenameTemplate,
		defaultTrackFilenameTemplate:    defaultTrackFilenameTemplate,
		defaultAlbumFolderTemplate:      defaultAlbumFolderTemplate,
		defaultPlaylistFilenameTemplate: defaultPlaylistFilenameTemplate,
	}
}

// GetTrackFilename generates a filename for a track based on its tags and context.
// If the track is part of a playlist or a single without a folder, it uses a different template.
func (s *TemplateManagerImpl) GetTrackFilename(
	ctx context.Context,
	isPlaylist bool,
	trackTags map[string]string,
	tracksCount int64,
) string {
	// Determine if the track is a single and should not have its own folder.
	isSingleWithoutFolder := !s.cfg.CreateFolderForSingles && tracksCount == 1

	// Select the appropriate template based on whether the track is part of a playlist or a single without a folder.
	textBuilder, defaultTextBuilder := s.trackFilenameTemplate, s.defaultTrackFilenameTemplate
	if isPlaylist || isSingleWithoutFolder {
		textBuilder, defaultTextBuilder = s.playlistFilenameTemplate, s.defaultPlaylistFilenameTemplate
	}

	// Execute the selected template with the track tags.
	var buffer bytes.Buffer
	if textBuilder != nil {
		if err := textBuilder.Execute(&buffer, trackTags); err != nil {
			logger.Errorf(ctx, "Failed to execute template, using default: %v", err)

			// Fall back to the default template if execution fails.
			buffer.Reset()
			_ = defaultTextBuilder.Execute(&buffer, trackTags)
		}
	} else {
		// Use default template if custom template is nil.
		_ = defaultTextBuilder.Execute(&buffer, trackTags)
	}

	// Unescape HTML entities in the generated filename.
	return html.UnescapeString(buffer.String())
}

// GetAlbumFolderName generates a folder name for an album based on its tags.
func (s *TemplateManagerImpl) GetAlbumFolderName(ctx context.Context, tags map[string]string) string {
	var (
		textBuilder = s.albumFolderTemplate
		buffer      bytes.Buffer
	)

	// Execute the template with the album tags.
	if textBuilder != nil {
		err := textBuilder.Execute(&buffer, tags)
		if err != nil {
			logger.Errorf(
				ctx,
				"Failed to execute template, default album folder template is being used. Error: %v",
				err,
			)

			// Fall back to the default template if execution fails.
			buffer.Reset()

			textBuilder = s.defaultAlbumFolderTemplate
			_ = textBuilder.Execute(&buffer, tags)
		}
	} else {
		// Use default template if custom template is nil.
		textBuilder = s.defaultAlbumFolderTemplate
		_ = textBuilder.Execute(&buffer, tags)
	}

	// Unescape HTML entities in the generated folder name.
	return html.UnescapeString(buffer.String())
}
