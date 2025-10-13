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

	// GetAudiobookFolderName generates a folder name for an audiobook based on its tags.
	GetAudiobookFolderName(ctx context.Context, tags map[string]string) string

	// GetAudiobookChapterFilename generates a filename for an audiobook chapter based on its tags.
	GetAudiobookChapterFilename(ctx context.Context, chapterTags map[string]string, totalChapters int64) string

	// GetPodcastFolderName generates a folder name for a podcast based on its tags.
	GetPodcastFolderName(ctx context.Context, tags map[string]string) string

	// GetPodcastEpisodeFilename generates a filename for a podcast episode based on its tags.
	GetPodcastEpisodeFilename(ctx context.Context, episodeTags map[string]string, totalEpisodes int64) string
}

// TemplateManagerImpl implements the TemplateManager interface.
type TemplateManagerImpl struct {
	// cfg contains the application configuration.
	cfg *config.Config
	// trackFilenameTemplate is the template for track filenames.
	trackFilenameTemplate *template.Template
	// albumFolderTemplate is the template for album folder names.
	albumFolderTemplate *template.Template
	// playlistFilenameTemplate is the template for playlist track filenames.
	playlistFilenameTemplate *template.Template
	// audiobookFolderTemplate is the template for audiobook folder names.
	audiobookFolderTemplate *template.Template
	// audiobookChapterFilenameTemplate is the template for audiobook chapter filenames.
	audiobookChapterFilenameTemplate *template.Template
	// podcastFolderTemplate is the template for podcast folder names.
	podcastFolderTemplate *template.Template
	// podcastEpisodeFilenameTemplate is the template for podcast episode filenames.
	podcastEpisodeFilenameTemplate *template.Template
	// defaultTrackFilenameTemplate is the fallback template for track filenames.
	defaultTrackFilenameTemplate *template.Template
	// defaultAlbumFolderTemplate is the fallback template for album folder names.
	defaultAlbumFolderTemplate *template.Template
	// defaultPlaylistFilenameTemplate is the fallback template for playlist track filenames.
	defaultPlaylistFilenameTemplate *template.Template
	// defaultAudiobookFolderTemplate is the fallback template for audiobook folder names.
	defaultAudiobookFolderTemplate *template.Template
	// defaultAudiobookChapterFilenameTemplate is the fallback template for audiobook chapter filenames.
	defaultAudiobookChapterFilenameTemplate *template.Template
	// defaultPodcastFolderTemplate is the fallback template for podcast folder names.
	defaultPodcastFolderTemplate *template.Template
	// defaultPodcastEpisodeFilenameTemplate is the fallback template for podcast episode filenames.
	defaultPodcastEpisodeFilenameTemplate *template.Template
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
	defaultAudiobookFolderTemplate := template.Must(
		template.New("defaultAudiobookFolderTemplate").Parse(config.DefaultAudiobookFolderTemplate))
	defaultAudiobookChapterFilenameTemplate := template.Must(
		template.New("defaultAudiobookChapterFilenameTemplate").Parse(config.DefaultAudiobookChapterFilenameTemplate))
	defaultPodcastFolderTemplate := template.Must(
		template.New("defaultPodcastFolderTemplate").Parse(config.DefaultPodcastFolderTemplate))
	defaultPodcastEpisodeFilenameTemplate := template.Must(
		template.New("defaultPodcastEpisodeFilenameTemplate").Parse(config.DefaultPodcastEpisodeFilenameTemplate))

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

	audiobookFolderTemplate, err := template.New("audiobookFolderTemplate").Parse(cfg.AudiobookFolderTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse audiobook folder template, using default: %v", err)
	}

	audiobookChapterFilenameTemplate, err := template.New("audiobookChapterFilenameTemplate").
		Parse(cfg.AudiobookChapterFilenameTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse audiobook chapter filename template, using default: %v", err)
	}

	podcastFolderTemplate, err := template.New("podcastFolderTemplate").Parse(cfg.PodcastFolderTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse podcast folder template, using default: %v", err)
	}

	podcastEpisodeFilenameTemplate, err := template.New("podcastEpisodeFilenameTemplate").
		Parse(cfg.PodcastEpisodeFilenameTemplate)
	if err != nil {
		logger.Errorf(ctx, "Failed to parse podcast episode filename template, using default: %v", err)
	}

	return &TemplateManagerImpl{
		cfg:                                     cfg,
		trackFilenameTemplate:                   trackFilenameTemplate,
		albumFolderTemplate:                     albumFolderTemplate,
		playlistFilenameTemplate:                playlistFilenameTemplate,
		audiobookFolderTemplate:                 audiobookFolderTemplate,
		audiobookChapterFilenameTemplate:        audiobookChapterFilenameTemplate,
		podcastFolderTemplate:                   podcastFolderTemplate,
		podcastEpisodeFilenameTemplate:          podcastEpisodeFilenameTemplate,
		defaultTrackFilenameTemplate:            defaultTrackFilenameTemplate,
		defaultAlbumFolderTemplate:              defaultAlbumFolderTemplate,
		defaultPlaylistFilenameTemplate:         defaultPlaylistFilenameTemplate,
		defaultAudiobookFolderTemplate:          defaultAudiobookFolderTemplate,
		defaultAudiobookChapterFilenameTemplate: defaultAudiobookChapterFilenameTemplate,
		defaultPodcastFolderTemplate:            defaultPodcastFolderTemplate,
		defaultPodcastEpisodeFilenameTemplate:   defaultPodcastEpisodeFilenameTemplate,
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
			_ = defaultTextBuilder.Execute(&buffer, trackTags) //nolint:errcheck // Default template is always valid.
		}
	} else {
		// Use default template if custom template is nil.
		_ = defaultTextBuilder.Execute(&buffer, trackTags) //nolint:errcheck // Default template is always valid.
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
			_ = textBuilder.Execute(&buffer, tags) //nolint:errcheck // Default template is always valid.
		}
	} else {
		// Use default template if custom template is nil.
		textBuilder = s.defaultAlbumFolderTemplate
		_ = textBuilder.Execute(&buffer, tags) //nolint:errcheck // Default template is always valid.
	}

	// Unescape HTML entities in the generated folder name.
	return html.UnescapeString(buffer.String())
}

// GetAudiobookFolderName generates a folder name for an audiobook based on its tags.
func (s *TemplateManagerImpl) GetAudiobookFolderName(ctx context.Context, tags map[string]string) string {
	var (
		textBuilder = s.audiobookFolderTemplate
		buffer      bytes.Buffer
	)

	// Execute the template with the audiobook tags.
	if textBuilder != nil {
		err := textBuilder.Execute(&buffer, tags)
		if err != nil {
			logger.Errorf(
				ctx,
				"Failed to execute audiobook folder template, using default. Error: %v",
				err,
			)

			// Fall back to the default template if execution fails.
			buffer.Reset()

			textBuilder = s.defaultAudiobookFolderTemplate
			_ = textBuilder.Execute(&buffer, tags) //nolint:errcheck // Default template is always valid.
		}
	} else {
		// Use default template if custom template is nil.
		textBuilder = s.defaultAudiobookFolderTemplate
		_ = textBuilder.Execute(&buffer, tags) //nolint:errcheck // Default template is always valid.
	}

	// Unescape HTML entities in the generated folder name.
	return html.UnescapeString(buffer.String())
}

// GetAudiobookChapterFilename generates a filename for an audiobook chapter based on its tags.
func (s *TemplateManagerImpl) GetAudiobookChapterFilename(
	ctx context.Context,
	chapterTags map[string]string,
	totalChapters int64,
) string {
	var (
		textBuilder        = s.audiobookChapterFilenameTemplate
		defaultTextBuilder = s.defaultAudiobookChapterFilenameTemplate
		buffer             bytes.Buffer
	)

	if textBuilder != nil {
		if err := textBuilder.Execute(&buffer, chapterTags); err != nil {
			logger.Errorf(ctx, "Failed to execute audiobook chapter template, using default: %v", err)
			buffer.Reset()
			_ = defaultTextBuilder.Execute(&buffer, chapterTags) //nolint:errcheck // Default template is always valid.
		}
	} else {
		_ = defaultTextBuilder.Execute(&buffer, chapterTags) //nolint:errcheck // Default template is always valid.
	}

	// Unescape HTML entities in the generated filename.
	return html.UnescapeString(buffer.String())
}

// GetPodcastFolderName generates a folder name for a podcast based on its tags.
func (s *TemplateManagerImpl) GetPodcastFolderName(ctx context.Context, tags map[string]string) string {
	var (
		textBuilder = s.podcastFolderTemplate
		buffer      bytes.Buffer
	)

	// Execute the template with the podcast tags.
	if textBuilder != nil {
		err := textBuilder.Execute(&buffer, tags)
		if err != nil {
			logger.Errorf(
				ctx,
				"Failed to execute podcast folder template, using default. Error: %v",
				err,
			)

			// Fall back to the default template if execution fails.
			buffer.Reset()

			textBuilder = s.defaultPodcastFolderTemplate
			_ = textBuilder.Execute(&buffer, tags) //nolint:errcheck // Default template is always valid.
		}
	} else {
		// Use default template if custom template is nil.
		textBuilder = s.defaultPodcastFolderTemplate
		_ = textBuilder.Execute(&buffer, tags) //nolint:errcheck // Default template is always valid.
	}

	// Unescape HTML entities in the generated folder name.
	return html.UnescapeString(buffer.String())
}

// GetPodcastEpisodeFilename generates a filename for a podcast episode based on its tags.
func (s *TemplateManagerImpl) GetPodcastEpisodeFilename(
	ctx context.Context,
	episodeTags map[string]string,
	totalEpisodes int64,
) string {
	var (
		textBuilder        = s.podcastEpisodeFilenameTemplate
		defaultTextBuilder = s.defaultPodcastEpisodeFilenameTemplate
		buffer             bytes.Buffer
	)

	if textBuilder != nil {
		if err := textBuilder.Execute(&buffer, episodeTags); err != nil {
			logger.Errorf(ctx, "Failed to execute podcast episode template, using default: %v", err)
			buffer.Reset()
			_ = defaultTextBuilder.Execute(&buffer, episodeTags) //nolint:errcheck // Default template is always valid.
		}
	} else {
		_ = defaultTextBuilder.Execute(&buffer, episodeTags) //nolint:errcheck // Default template is always valid.
	}

	// Unescape HTML entities in the generated filename.
	return html.UnescapeString(buffer.String())
}
