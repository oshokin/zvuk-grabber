package media

//go:generate $MOCKGEN -source=template_manager.go -destination=mocks/template_manager_mock.go

import (
	"bytes"
	"context"
	"html"
	"html/template"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
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
	// trackFilenameTemplates contains templates for regular track filenames.
	trackFilenameTemplates templatePair
	// albumFolderTemplates contains templates for album folder names.
	albumFolderTemplates templatePair
	// playlistFilenameTemplates contains templates for playlist track filenames.
	playlistFilenameTemplates templatePair
	// audiobookFolderTemplates contains templates for audiobook folder names.
	audiobookFolderTemplates templatePair
	// audiobookChapterFilenameTemplates contains templates for audiobook chapter filenames.
	audiobookChapterFilenameTemplates templatePair
	// podcastFolderTemplates contains templates for podcast folder names.
	podcastFolderTemplates templatePair
	// podcastEpisodeFilenameTemplates contains templates for podcast episode filenames.
	podcastEpisodeFilenameTemplates templatePair
}

// templatePair stores a configurable template together with its guaranteed-valid fallback.
type templatePair struct {
	// custom is the user-configured template, nil when using the default.
	custom *template.Template
	// fallback is the guaranteed-valid template used when custom parsing fails.
	fallback *template.Template
}

// NewTemplateManager creates and returns a new instance of TemplateManagerImpl.
// It initializes templates from the configuration and falls back to default templates if parsing fails.
func NewTemplateManager(ctx context.Context, cfg *config.Config) TemplateManager {
	return &TemplateManagerImpl{
		cfg: cfg,
		trackFilenameTemplates: newTemplatePair(
			ctx,
			"trackFilenameTemplate",
			cfg.TrackFilenameTemplate,
			config.DefaultTrackFilenameTemplate,
			"Failed to parse track filename template, using default: %v",
		),
		albumFolderTemplates: newTemplatePair(
			ctx,
			"albumFolderTemplate",
			cfg.AlbumFolderTemplate,
			config.DefaultAlbumFolderTemplate,
			"Failed to parse album folder template, using default: %v",
		),
		playlistFilenameTemplates: newTemplatePair(
			ctx,
			"playlistFilenameTemplate",
			cfg.PlaylistFilenameTemplate,
			config.DefaultPlaylistFilenameTemplate,
			"Failed to parse playlist filename template, using default: %v",
		),
		audiobookFolderTemplates: newTemplatePair(
			ctx,
			"audiobookFolderTemplate",
			cfg.AudiobookFolderTemplate,
			config.DefaultAudiobookFolderTemplate,
			"Failed to parse audiobook folder template, using default: %v",
		),
		audiobookChapterFilenameTemplates: newTemplatePair(
			ctx,
			"audiobookChapterFilenameTemplate",
			cfg.AudiobookChapterFilenameTemplate,
			config.DefaultAudiobookChapterFilenameTemplate,
			"Failed to parse audiobook chapter filename template, using default: %v",
		),
		podcastFolderTemplates: newTemplatePair(
			ctx,
			"podcastFolderTemplate",
			cfg.PodcastFolderTemplate,
			config.DefaultPodcastFolderTemplate,
			"Failed to parse podcast folder template, using default: %v",
		),
		podcastEpisodeFilenameTemplates: newTemplatePair(
			ctx,
			"podcastEpisodeFilenameTemplate",
			cfg.PodcastEpisodeFilenameTemplate,
			config.DefaultPodcastEpisodeFilenameTemplate,
			"Failed to parse podcast episode filename template, using default: %v",
		),
	}
}

// render executes a configured template and transparently falls back to the default template on error.
func (pair *templatePair) render(ctx context.Context, data any, executeErrorMessage string) string {
	selected := pair.custom
	if selected == nil {
		selected = pair.fallback
	}

	var buffer bytes.Buffer
	if err := selected.Execute(&buffer, data); err != nil && selected != pair.fallback {
		logger.Errorf(ctx, executeErrorMessage, err)
		buffer.Reset()
		_ = pair.fallback.Execute(&buffer, data) //nolint:errcheck // The fallback template is always valid.
	}

	return html.UnescapeString(buffer.String())
}

// newTemplatePair parses a configurable template and prepares its guaranteed-valid fallback.
func newTemplatePair(
	ctx context.Context,
	name string,
	configuredValue string,
	defaultValue string,
	parseErrorMessage string,
) templatePair {
	fallback := template.Must(template.New("default" + name).Parse(defaultValue))

	custom, err := template.New(name).Parse(configuredValue)
	if err != nil {
		logger.Errorf(ctx, parseErrorMessage, err)
	}

	return templatePair{
		custom:   custom,
		fallback: fallback,
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
	isSingleWithoutFolder := !s.cfg.CreateFolderForSingles && tracksCount == 1

	templates := s.trackFilenameTemplates
	if isPlaylist || isSingleWithoutFolder {
		templates = s.playlistFilenameTemplates
	}

	return templates.render(ctx, trackTags, "Failed to execute template, using default: %v")
}

// GetAlbumFolderName generates a folder name for an album based on its tags.
func (s *TemplateManagerImpl) GetAlbumFolderName(ctx context.Context, tags map[string]string) string {
	sanitizedTags := make(map[string]string, len(tags))
	for key, value := range tags {
		sanitizedTags[key] = utils.SanitizeFilename(value)
	}

	return s.albumFolderTemplates.render(
		ctx,
		sanitizedTags,
		"Failed to execute template, default album folder template is being used. Error: %v",
	)
}

// GetAudiobookFolderName generates a folder name for an audiobook based on its tags.
func (s *TemplateManagerImpl) GetAudiobookFolderName(ctx context.Context, tags map[string]string) string {
	return s.audiobookFolderTemplates.render(
		ctx,
		tags,
		"Failed to execute audiobook folder template, using default. Error: %v",
	)
}

// GetAudiobookChapterFilename generates a filename for an audiobook chapter based on its tags.
func (s *TemplateManagerImpl) GetAudiobookChapterFilename(
	ctx context.Context,
	chapterTags map[string]string,
	_ int64,
) string {
	return s.audiobookChapterFilenameTemplates.render(
		ctx,
		chapterTags,
		"Failed to execute audiobook chapter template, using default: %v",
	)
}

// GetPodcastFolderName generates a folder name for a podcast based on its tags.
func (s *TemplateManagerImpl) GetPodcastFolderName(ctx context.Context, tags map[string]string) string {
	return s.podcastFolderTemplates.render(
		ctx,
		tags,
		"Failed to execute podcast folder template, using default. Error: %v",
	)
}

// GetPodcastEpisodeFilename generates a filename for a podcast episode based on its tags.
func (s *TemplateManagerImpl) GetPodcastEpisodeFilename(
	ctx context.Context,
	episodeTags map[string]string,
	_ int64,
) string {
	return s.podcastEpisodeFilenameTemplates.render(
		ctx,
		episodeTags,
		"Failed to execute podcast episode template, using default: %v",
	)
}
