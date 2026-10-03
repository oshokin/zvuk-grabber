package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Config holds all configuration settings.
type Config struct {
	// ZvukDownloadHTTP configures the dedicated audio download client.
	ZvukDownloadHTTP *DownloadHTTPConfig `mapstructure:"zvuk_download_http"`
	// YandexMusicDownloadHTTP configures Yandex audio transfers independently of API calls.
	YandexMusicDownloadHTTP *DownloadHTTPConfig `mapstructure:"yandex_music_download_http"`
	// ZvukAuthToken is the authentication token for Zvuk API access.
	ZvukAuthToken string `mapstructure:"zvuk_auth_token"`
	// YandexMusicToken is the OAuth token for Yandex Music API access.
	YandexMusicToken string `mapstructure:"yandex_music_token"`
	// Quality specifies the preferred audio quality (1=MP3 128k, 2=MP3 320k, 3=FLAC).
	Quality uint8 `mapstructure:"quality"`
	// MinQuality specifies the minimum acceptable quality (1=MP3 128k, 2=MP3 320k, 3=FLAC).
	// Tracks below this quality will be skipped. Set to 0 to disable filtering.
	MinQuality uint8 `mapstructure:"min_quality"`
	// MinDuration specifies the minimum acceptable track duration (e.g., "30s", "1m").
	// Tracks shorter than this will be skipped. Empty string disables filtering.
	MinDuration string `mapstructure:"min_duration"`
	// MaxDuration specifies the maximum acceptable track duration (e.g., "10m", "1h").
	// Tracks longer than this will be skipped. Empty string disables filtering.
	MaxDuration string `mapstructure:"max_duration"`
	// OutputPath is the directory path where downloaded files will be saved.
	OutputPath string `mapstructure:"output_path"`
	// GroupByProvider controls whether provider subfolders are created under output_path.
	GroupByProvider bool `mapstructure:"group_by_provider"`
	// TrackFilenameTemplate is the template for naming individual track files.
	TrackFilenameTemplate string `mapstructure:"track_filename_template"`
	// AlbumFolderTemplate is the template for naming album folders.
	AlbumFolderTemplate string `mapstructure:"album_folder_template"`
	// PlaylistFilenameTemplate is the template for naming playlist track files.
	PlaylistFilenameTemplate string `mapstructure:"playlist_filename_template"`
	// AudiobookFolderTemplate is the template for naming audiobook folders.
	AudiobookFolderTemplate string `mapstructure:"audiobook_folder_template"`
	// AudiobookChapterFilenameTemplate is the template for naming audiobook chapter files.
	AudiobookChapterFilenameTemplate string `mapstructure:"audiobook_chapter_filename_template"`
	// PodcastFolderTemplate is the template for naming podcast folders.
	PodcastFolderTemplate string `mapstructure:"podcast_folder_template"`
	// PodcastEpisodeFilenameTemplate is the template for naming podcast episode files.
	PodcastEpisodeFilenameTemplate string `mapstructure:"podcast_episode_filename_template"`
	// DownloadLyrics indicates whether to download lyrics for tracks.
	DownloadLyrics bool `mapstructure:"download_lyrics"`
	// ReplaceTracks indicates whether to replace existing track files.
	ReplaceTracks bool `mapstructure:"replace_tracks"`
	// ReplaceCovers indicates whether to replace existing cover art files.
	ReplaceCovers bool `mapstructure:"replace_covers"`
	// ReplaceDescriptions indicates whether to replace existing description files.
	ReplaceDescriptions bool `mapstructure:"replace_descriptions"`
	// ReplaceLyrics indicates whether to replace existing lyrics files.
	ReplaceLyrics bool `mapstructure:"replace_lyrics"`
	// LogLevel specifies the logging verbosity level.
	LogLevel string `mapstructure:"log_level"`
	// DownloadSpeedLimit sets the maximum download speed (e.g., "1MB", "500KB").
	DownloadSpeedLimit string `mapstructure:"download_speed_limit"`
	// CreateFolderForSingles indicates whether to create folders for single tracks.
	CreateFolderForSingles bool `mapstructure:"create_folder_for_singles"`
	// MaxFolderNameLength is the maximum length for folder names.
	MaxFolderNameLength int64 `mapstructure:"max_folder_name_length"`
	// APIRetryAttemptsCount is the number of attempts for provider API requests.
	APIRetryAttemptsCount int64 `mapstructure:"api_retry_attempts_count"`
	// MaxDownloadPause is the maximum pause duration between downloads.
	MaxDownloadPause string `mapstructure:"max_download_pause"`
	// APIMinRetryPause is the minimum pause duration before retrying an API request.
	APIMinRetryPause string `mapstructure:"api_min_retry_pause"`
	// APIMaxRetryPause is the maximum pause duration before retrying an API request.
	APIMaxRetryPause string `mapstructure:"api_max_retry_pause"`
	// MaxConcurrentDownloads is the maximum number of tracks to download simultaneously.
	MaxConcurrentDownloads int64 `mapstructure:"max_concurrent_downloads"`
	// ZvukBaseURL is the base URL for the Zvuk API (set automatically).
	ZvukBaseURL string
	// DryRun indicates whether to preview downloads without actually downloading files.
	DryRun bool
	// ParsedMinDuration is the parsed minimum track duration.
	ParsedMinDuration time.Duration
	// ParsedMaxDuration is the parsed maximum track duration.
	ParsedMaxDuration time.Duration
	// ParsedDownloadSpeedLimit is the parsed download speed limit in bytes.
	ParsedDownloadSpeedLimit int64
	// ParsedLogLevel is the parsed zap log level.
	ParsedLogLevel zapcore.Level
	// ParsedMaxDownloadPause is the parsed maximum download pause duration.
	ParsedMaxDownloadPause time.Duration
	// ParsedAPIMinRetryPause is the parsed minimum API retry pause duration.
	ParsedAPIMinRetryPause time.Duration
	// ParsedAPIMaxRetryPause is the parsed maximum API retry pause duration.
	ParsedAPIMaxRetryPause time.Duration
}

const (
	// ZvukBaseURL is the base URL for the Zvuk service.
	ZvukBaseURL = "https://zvuk.com"

	// DefaultConfigFilename is the default name of the configuration file.
	DefaultConfigFilename = ".zvuk-grabber.yaml"

	// DefaultQuality is the preferred download quality (3=FLAC with MP3 fallback).
	DefaultQuality = 3
	// DefaultMinQuality disables quality filtering by default.
	DefaultMinQuality = 0
	// DefaultMinDuration disables minimum duration filtering.
	DefaultMinDuration = ""
	// DefaultMaxDuration disables maximum duration filtering.
	DefaultMaxDuration = ""

	// DefaultOutputPath is the base directory where downloads are saved.
	DefaultOutputPath = "zvuk-grabber-downloads"

	// DefaultGroupByProvider controls provider subfolder grouping under output_path.
	DefaultGroupByProvider = true

	// DefaultDownloadLyrics enables lyrics downloading when available.
	DefaultDownloadLyrics = true
	// DefaultReplaceTracks keeps existing track files by default.
	DefaultReplaceTracks = false
	// DefaultReplaceCovers keeps existing covers by default.
	DefaultReplaceCovers = false
	// DefaultReplaceDescriptions keeps existing descriptions by default.
	DefaultReplaceDescriptions = false
	// DefaultReplaceLyrics keeps existing lyrics by default.
	DefaultReplaceLyrics = false
	// DefaultLogLevel defines default logging verbosity.
	DefaultLogLevel = "info"
	// DefaultDownloadSpeedLimit means unlimited speed.
	DefaultDownloadSpeedLimit = ""
	// DefaultCreateFolderForSingles keeps singles in root by default.
	DefaultCreateFolderForSingles = false
	// DefaultMaxFolderNameLength limits generated folder names.
	DefaultMaxFolderNameLength int64 = 100
	// DefaultAPIRetryAttemptsCount is the default provider API retry count.
	DefaultAPIRetryAttemptsCount int64 = 5
	// DefaultMaxDownloadPause controls pause between downloads.
	DefaultMaxDownloadPause = "2s"
	// DefaultAPIMinRetryPause controls the minimum API retry pause.
	DefaultAPIMinRetryPause = "3s"
	// DefaultAPIMaxRetryPause controls the maximum API retry pause.
	DefaultAPIMaxRetryPause = "7s"
	// DefaultMaxConcurrentDownloads defaults to safe sequential mode.
	DefaultMaxConcurrentDownloads int64 = 1

	// DefaultTrackFilenameTemplate is the default template for naming downloaded track files.
	DefaultTrackFilenameTemplate = "{{.trackNumberPad}} - {{.trackTitle}}"

	// DefaultAlbumFolderTemplate is the default template for naming folders for downloaded albums.
	DefaultAlbumFolderTemplate = "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"

	// DefaultPlaylistFilenameTemplate is the default template for naming downloaded track files from playlists.
	DefaultPlaylistFilenameTemplate = "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"

	// DefaultAudiobookFolderTemplate is the default template for naming audiobook folders.
	DefaultAudiobookFolderTemplate = "{{.publishYear}} - {{.audiobookAuthors}} - {{.audiobookTitle}}"

	// DefaultAudiobookChapterFilenameTemplate is the default template for naming audiobook chapter files.
	DefaultAudiobookChapterFilenameTemplate = "{{.trackNumberPad}} - {{.trackTitle}}"

	// DefaultPodcastFolderTemplate is the default template for naming podcast folders.
	DefaultPodcastFolderTemplate = "{{.podcastAuthors}} - {{.podcastTitle}}"

	// DefaultPodcastEpisodeFilenameTemplate is the default template for naming podcast episode files.
	DefaultPodcastEpisodeFilenameTemplate = "{{.episodePublicationDate}} - {{.trackTitle}}"

	// DefaultMaxLogLength is the default maximum size (in bytes) for log files.
	DefaultMaxLogLength = 1 * 1024 * 1024 // 1 MB

	// minQuality is the minimum valid quality value.
	minQuality = 1
	// maxQuality is the maximum valid quality value.
	maxQuality = 3

	// configKeyZvukAuthToken is the YAML config key for the Zvuk authentication token.
	//
	//nolint:gosec // These are YAML keys, not credentials.
	configKeyZvukAuthToken = "zvuk_auth_token"
	// configKeyYandexMusicToken is the YAML config key for the Yandex Music OAuth token.
	//
	//nolint:gosec // These are YAML keys, not credentials.
	configKeyYandexMusicToken = "yandex_music_token"
	// configKeyAPIRetryAttemptsCount is the YAML key for provider API retry attempts.
	configKeyAPIRetryAttemptsCount = "api_retry_attempts_count"
	// configKeyAPIMinRetryPause is the YAML key for the minimum API retry pause.
	configKeyAPIMinRetryPause = "api_min_retry_pause"
	// configKeyAPIMaxRetryPause is the YAML key for the maximum API retry pause.
	configKeyAPIMaxRetryPause = "api_max_retry_pause"
	// yamlStringTag is the YAML scalar tag used when writing quoted string values.
	yamlStringTag = "!!str"
)

// Static error definitions for better error handling.
var (
	// ErrEmptyZvukAuthToken indicates that the Zvuk authentication token is missing.
	ErrEmptyZvukAuthToken = errors.New("zvuk_auth_token cannot be empty")
	// ErrEmptyYandexMusicToken indicates that the Yandex Music authentication token is missing.
	ErrEmptyYandexMusicToken = errors.New("yandex_music_token cannot be empty")
	// ErrInvalidQuality indicates that the quality setting is invalid.
	ErrInvalidQuality = errors.New("invalid quality")
	// ErrInvalidMinQuality indicates that the minimum quality setting is invalid.
	ErrInvalidMinQuality = errors.New("invalid min_quality")
	// ErrMinQualityTooHigh indicates that min_quality is higher than quality.
	ErrMinQualityTooHigh = errors.New("min_quality cannot be higher than quality")
	// ErrInvalidMinDuration indicates that the minimum duration setting is invalid.
	ErrInvalidMinDuration = errors.New("min_duration must be positive")
	// ErrInvalidMaxDuration indicates that the maximum duration setting is invalid.
	ErrInvalidMaxDuration = errors.New("max_duration must be positive")
	// ErrMaxDurationTooLow indicates that max_duration is not greater than min_duration.
	ErrMaxDurationTooLow = errors.New("max_duration must be greater than min_duration")
	// ErrUnknownLogLevel indicates that the log level is not recognized.
	ErrUnknownLogLevel = errors.New("unknown log level")
	// ErrInvalidAPIRetryAttempts indicates that the API retry attempts count is invalid.
	ErrInvalidAPIRetryAttempts = errors.New("api_retry_attempts_count must be a positive integer")
	// ErrInvalidMaxDownloadPause indicates that the max download pause duration is invalid.
	ErrInvalidMaxDownloadPause = errors.New("max_download_pause must be positive")
	// ErrInvalidAPIMinRetryPause indicates that the API min retry pause duration is invalid.
	ErrInvalidAPIMinRetryPause = errors.New("api_min_retry_pause must be positive")
	// ErrInvalidAPIMaxRetryPause indicates that the API max retry pause duration is invalid.
	ErrInvalidAPIMaxRetryPause = errors.New("api_max_retry_pause must be positive")
	// ErrInvalidConcurrentDownloads indicates that the concurrent downloads count is invalid.
	ErrInvalidConcurrentDownloads = errors.New("max concurrent downloads must be a positive integer")
)

// DefaultConfig returns a fully-populated config with safe defaults and empty tokens.
func DefaultConfig() *Config {
	return &Config{
		ZvukDownloadHTTP:                 DefaultDownloadHTTPConfig(),
		YandexMusicDownloadHTTP:          DefaultDownloadHTTPConfig(),
		ZvukAuthToken:                    "",
		YandexMusicToken:                 "",
		Quality:                          DefaultQuality,
		MinQuality:                       DefaultMinQuality,
		MinDuration:                      DefaultMinDuration,
		MaxDuration:                      DefaultMaxDuration,
		OutputPath:                       DefaultOutputPath,
		GroupByProvider:                  DefaultGroupByProvider,
		TrackFilenameTemplate:            DefaultTrackFilenameTemplate,
		AlbumFolderTemplate:              DefaultAlbumFolderTemplate,
		PlaylistFilenameTemplate:         DefaultPlaylistFilenameTemplate,
		AudiobookFolderTemplate:          DefaultAudiobookFolderTemplate,
		AudiobookChapterFilenameTemplate: DefaultAudiobookChapterFilenameTemplate,
		PodcastFolderTemplate:            DefaultPodcastFolderTemplate,
		PodcastEpisodeFilenameTemplate:   DefaultPodcastEpisodeFilenameTemplate,
		DownloadLyrics:                   DefaultDownloadLyrics,
		ReplaceTracks:                    DefaultReplaceTracks,
		ReplaceCovers:                    DefaultReplaceCovers,
		ReplaceDescriptions:              DefaultReplaceDescriptions,
		ReplaceLyrics:                    DefaultReplaceLyrics,
		LogLevel:                         DefaultLogLevel,
		DownloadSpeedLimit:               DefaultDownloadSpeedLimit,
		CreateFolderForSingles:           DefaultCreateFolderForSingles,
		MaxFolderNameLength:              DefaultMaxFolderNameLength,
		APIRetryAttemptsCount:            DefaultAPIRetryAttemptsCount,
		MaxDownloadPause:                 DefaultMaxDownloadPause,
		APIMinRetryPause:                 DefaultAPIMinRetryPause,
		APIMaxRetryPause:                 DefaultAPIMaxRetryPause,
		MaxConcurrentDownloads:           DefaultMaxConcurrentDownloads,
	}
}

// Clone returns a detached copy, including nested download HTTP settings.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	return &Config{
		ZvukDownloadHTTP:                 c.ZvukDownloadHTTP.Clone(),
		YandexMusicDownloadHTTP:          c.YandexMusicDownloadHTTP.Clone(),
		ZvukAuthToken:                    c.ZvukAuthToken,
		YandexMusicToken:                 c.YandexMusicToken,
		Quality:                          c.Quality,
		MinQuality:                       c.MinQuality,
		MinDuration:                      c.MinDuration,
		MaxDuration:                      c.MaxDuration,
		OutputPath:                       c.OutputPath,
		GroupByProvider:                  c.GroupByProvider,
		TrackFilenameTemplate:            c.TrackFilenameTemplate,
		AlbumFolderTemplate:              c.AlbumFolderTemplate,
		PlaylistFilenameTemplate:         c.PlaylistFilenameTemplate,
		AudiobookFolderTemplate:          c.AudiobookFolderTemplate,
		AudiobookChapterFilenameTemplate: c.AudiobookChapterFilenameTemplate,
		PodcastFolderTemplate:            c.PodcastFolderTemplate,
		PodcastEpisodeFilenameTemplate:   c.PodcastEpisodeFilenameTemplate,
		DownloadLyrics:                   c.DownloadLyrics,
		ReplaceTracks:                    c.ReplaceTracks,
		ReplaceCovers:                    c.ReplaceCovers,
		ReplaceDescriptions:              c.ReplaceDescriptions,
		ReplaceLyrics:                    c.ReplaceLyrics,
		LogLevel:                         c.LogLevel,
		DownloadSpeedLimit:               c.DownloadSpeedLimit,
		CreateFolderForSingles:           c.CreateFolderForSingles,
		MaxFolderNameLength:              c.MaxFolderNameLength,
		APIRetryAttemptsCount:            c.APIRetryAttemptsCount,
		MaxDownloadPause:                 c.MaxDownloadPause,
		APIMinRetryPause:                 c.APIMinRetryPause,
		APIMaxRetryPause:                 c.APIMaxRetryPause,
		MaxConcurrentDownloads:           c.MaxConcurrentDownloads,
		ZvukBaseURL:                      c.ZvukBaseURL,
		DryRun:                           c.DryRun,
		ParsedMinDuration:                c.ParsedMinDuration,
		ParsedMaxDuration:                c.ParsedMaxDuration,
		ParsedDownloadSpeedLimit:         c.ParsedDownloadSpeedLimit,
		ParsedLogLevel:                   c.ParsedLogLevel,
		ParsedMaxDownloadPause:           c.ParsedMaxDownloadPause,
		ParsedAPIMinRetryPause:           c.ParsedAPIMinRetryPause,
		ParsedAPIMaxRetryPause:           c.ParsedAPIMaxRetryPause,
	}
}

// LoadConfig loads configuration settings from a YAML file.
func LoadConfig(configFilename string) (*Config, error) {
	if configFilename == "" {
		configFilename = DefaultConfigFilename
	}

	viper.Reset()
	viper.SetConfigFile(configFilename)
	applyViperDefaults()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError

		if !errors.As(err, &configFileNotFoundError) && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config from file: %w", err)
		}
	}

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.YandexMusicDownloadHTTP == nil {
		cfg.YandexMusicDownloadHTTP = DefaultDownloadHTTPConfig()
	}

	if cfg.ZvukDownloadHTTP == nil {
		cfg.ZvukDownloadHTTP = DefaultDownloadHTTPConfig()
	}

	if err := cfg.ZvukDownloadHTTP.parseReceiveBuffer(); err != nil {
		return nil, fmt.Errorf("zvuk_download_http: %w", err)
	}

	if err := cfg.YandexMusicDownloadHTTP.parseReceiveBuffer(); err != nil {
		return nil, fmt.Errorf("yandex_music_download_http: %w", err)
	}

	return &cfg, nil
}

// ValidateConfig checks the configuration for validity and sets derived fields.
//
//nolint:funlen,gocognit // Validation functions naturally have high complexity and length due to sequential checks.
func ValidateConfig(cfg *Config) error {
	var (
		downloadSpeedLimit       = strings.TrimSpace(cfg.DownloadSpeedLimit)
		parsedDownloadSpeedLimit uint64
		err                      error
	)

	if cfg.YandexMusicDownloadHTTP == nil {
		cfg.YandexMusicDownloadHTTP = DefaultDownloadHTTPConfig()
	}

	if cfg.ZvukDownloadHTTP == nil {
		cfg.ZvukDownloadHTTP = DefaultDownloadHTTPConfig()
	}

	if err = cfg.ZvukDownloadHTTP.Validate(); err != nil {
		return fmt.Errorf("zvuk_download_http: %w", err)
	}

	if err = cfg.YandexMusicDownloadHTTP.Validate(); err != nil {
		return fmt.Errorf("yandex_music_download_http: %w", err)
	}

	cfg.ZvukBaseURL = ZvukBaseURL
	if strings.TrimSpace(cfg.OutputPath) == "" {
		cfg.OutputPath = DefaultOutputPath
	}

	if !isValidQuality(cfg.Quality) {
		return fmt.Errorf("%w: must be between %d and %d", ErrInvalidQuality, minQuality, maxQuality)
	}

	// Validate min_quality if set (0 means no filtering).
	if cfg.MinQuality > 0 {
		if !isValidQuality(cfg.MinQuality) {
			return fmt.Errorf("%w: must be between %d and %d, or 0 to disable",
				ErrInvalidMinQuality, minQuality, maxQuality)
		}

		if cfg.MinQuality > cfg.Quality {
			return ErrMinQualityTooHigh
		}
	}

	cfg.ParsedMinDuration, err = parseOptionalPositiveDuration(cfg.MinDuration, "min duration", ErrInvalidMinDuration)
	if err != nil {
		return err
	}

	cfg.ParsedMaxDuration, err = parseOptionalPositiveDuration(cfg.MaxDuration, "max duration", ErrInvalidMaxDuration)
	if err != nil {
		return err
	}

	if cfg.MinDuration != "" && cfg.MaxDuration != "" && cfg.ParsedMaxDuration <= cfg.ParsedMinDuration {
		return ErrMaxDurationTooLow
	}

	parsedLogLevel, isLogLevelCorrect := logger.ParseLogLevel(cfg.LogLevel)
	if !isLogLevelCorrect {
		return fmt.Errorf("%w: '%s'", ErrUnknownLogLevel, cfg.LogLevel)
	}

	cfg.ParsedLogLevel = parsedLogLevel

	if downloadSpeedLimit != "" && downloadSpeedLimit != "0" {
		parsedDownloadSpeedLimit, err = humanize.ParseBytes(downloadSpeedLimit)
		if err != nil {
			return fmt.Errorf("failed to parse download speed limit: %w", err)
		}
	}

	// io.CopyN accepts only int64 so we transform it safely in order to use it later.
	cfg.ParsedDownloadSpeedLimit = utils.SafeUint64ToInt64(parsedDownloadSpeedLimit)

	if cfg.APIRetryAttemptsCount <= 0 {
		return ErrInvalidAPIRetryAttempts
	}

	cfg.ParsedMaxDownloadPause, err = parsePositiveDuration(
		cfg.MaxDownloadPause,
		"max download pause",
		ErrInvalidMaxDownloadPause,
	)
	if err != nil {
		return err
	}

	cfg.ParsedAPIMinRetryPause, err = parsePositiveDuration(
		cfg.APIMinRetryPause,
		configKeyAPIMinRetryPause,
		ErrInvalidAPIMinRetryPause,
	)
	if err != nil {
		return err
	}

	cfg.ParsedAPIMaxRetryPause, err = parsePositiveDuration(
		cfg.APIMaxRetryPause,
		configKeyAPIMaxRetryPause,
		ErrInvalidAPIMaxRetryPause,
	)
	if err != nil {
		return err
	}

	if cfg.MaxConcurrentDownloads <= 0 {
		return ErrInvalidConcurrentDownloads
	}

	return nil
}

// SaveConfig saves the configuration to the file while preserving the original format and order.
func SaveConfig(cfg *Config) error {
	return SaveConfigFields(cfg, map[string]string{
		configKeyZvukAuthToken:    cfg.ZvukAuthToken,
		configKeyYandexMusicToken: cfg.YandexMusicToken,
	})
}

// SaveConfigFields saves selected sensitive fields while preserving YAML order where possible.
func SaveConfigFields(cfg *Config, fields map[string]string) error {
	configFile := getConfigFilePath()

	// Read the original file content.
	originalContent, err := os.ReadFile(configFile)
	if err != nil {
		return handleMissingConfigFile(configFile, cfg, err)
	}

	// Parse YAML while preserving order using yaml.Node.
	var node yaml.Node

	if err = yaml.Unmarshal(originalContent, &node); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	updateFieldsInNode(&node, fields)

	// Marshal back to YAML (preserves order).
	newContent, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Config contains access tokens, so keep it private to the current user.
	if err = os.WriteFile(configFile, newContent, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyViperDefaults registers default configuration values with viper.
func applyViperDefaults() {
	defaults := DefaultConfig()
	setDownloadHTTPDefaults(viperPrefixDownloadHTTP, defaults.ZvukDownloadHTTP)
	setDownloadHTTPDefaults(viperPrefixYandexDownloadHTTP, defaults.YandexMusicDownloadHTTP)

	viper.SetDefault(configKeyZvukAuthToken, defaults.ZvukAuthToken)
	viper.SetDefault(configKeyYandexMusicToken, defaults.YandexMusicToken)
	viper.SetDefault("quality", defaults.Quality)
	viper.SetDefault("min_quality", defaults.MinQuality)
	viper.SetDefault("min_duration", defaults.MinDuration)
	viper.SetDefault("max_duration", defaults.MaxDuration)
	viper.SetDefault("output_path", defaults.OutputPath)
	viper.SetDefault("group_by_provider", defaults.GroupByProvider)
	viper.SetDefault("track_filename_template", defaults.TrackFilenameTemplate)
	viper.SetDefault("album_folder_template", defaults.AlbumFolderTemplate)
	viper.SetDefault("playlist_filename_template", defaults.PlaylistFilenameTemplate)
	viper.SetDefault("audiobook_folder_template", defaults.AudiobookFolderTemplate)
	viper.SetDefault("audiobook_chapter_filename_template", defaults.AudiobookChapterFilenameTemplate)
	viper.SetDefault("podcast_folder_template", defaults.PodcastFolderTemplate)
	viper.SetDefault("podcast_episode_filename_template", defaults.PodcastEpisodeFilenameTemplate)
	viper.SetDefault("download_lyrics", defaults.DownloadLyrics)
	viper.SetDefault("replace_tracks", defaults.ReplaceTracks)
	viper.SetDefault("replace_covers", defaults.ReplaceCovers)
	viper.SetDefault("replace_descriptions", defaults.ReplaceDescriptions)
	viper.SetDefault("replace_lyrics", defaults.ReplaceLyrics)
	viper.SetDefault("log_level", defaults.LogLevel)
	viper.SetDefault("download_speed_limit", defaults.DownloadSpeedLimit)
	viper.SetDefault("create_folder_for_singles", defaults.CreateFolderForSingles)
	viper.SetDefault("max_folder_name_length", defaults.MaxFolderNameLength)
	viper.SetDefault(configKeyAPIRetryAttemptsCount, defaults.APIRetryAttemptsCount)
	viper.SetDefault("max_download_pause", defaults.MaxDownloadPause)
	viper.SetDefault(configKeyAPIMinRetryPause, defaults.APIMinRetryPause)
	viper.SetDefault(configKeyAPIMaxRetryPause, defaults.APIMaxRetryPause)
	viper.SetDefault("max_concurrent_downloads", defaults.MaxConcurrentDownloads)
}

// isValidQuality reports whether the quality value is within the supported range.
func isValidQuality(quality uint8) bool {
	return quality >= minQuality && quality <= maxQuality
}

// parsePositiveDuration parses a non-empty duration string and validates it is positive.
func parsePositiveDuration(raw, name string, invalidErr error) (time.Duration, error) {
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s: %w", name, err)
	}

	if parsed <= 0 {
		return 0, invalidErr
	}

	return parsed, nil
}

// parseOptionalPositiveDuration parses an optional duration string, returning zero when empty.
func parseOptionalPositiveDuration(raw, name string, invalidErr error) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}

	return parsePositiveDuration(raw, name, invalidErr)
}

// getConfigFilePath returns the config file path from viper or the default.
func getConfigFilePath() string {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return DefaultConfigFilename
	}

	return configFile
}

// handleMissingConfigFile creates a new config file if it doesn't exist.
func handleMissingConfigFile(configFile string, cfg *Config, err error) error {
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	defaultCfg := DefaultConfig()
	if cfg != nil {
		defaultCfg.ZvukAuthToken = cfg.ZvukAuthToken
		defaultCfg.YandexMusicToken = cfg.YandexMusicToken
	}

	content := renderConfigContent(defaultCfg)

	if err = os.WriteFile(configFile, content, 0o600); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	return nil
}

// renderConfigContent renders the default YAML configuration file contents.
//
//nolint:funlen // YAML dump lists every config field.
func renderConfigContent(cfg *Config) []byte {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	httpCfg := cfg.ZvukDownloadHTTP
	if httpCfg == nil {
		httpCfg = DefaultDownloadHTTPConfig()
	}

	yandexHTTP := cfg.YandexMusicDownloadHTTP
	if yandexHTTP == nil {
		yandexHTTP = DefaultDownloadHTTPConfig()
	}

	return fmt.Appendf(
		nil,
		`zvuk_auth_token: %q
yandex_music_token: %q
quality: %d
min_quality: %d
min_duration: %q
max_duration: %q
output_path: %q
group_by_provider: %t
track_filename_template: %q
album_folder_template: %q
playlist_filename_template: %q
audiobook_folder_template: %q
audiobook_chapter_filename_template: %q
podcast_folder_template: %q
podcast_episode_filename_template: %q
download_lyrics: %t
replace_tracks: %t
replace_covers: %t
replace_descriptions: %t
replace_lyrics: %t
log_level: %q
download_speed_limit: %q
create_folder_for_singles: %t
max_folder_name_length: %d
api_retry_attempts_count: %d
max_download_pause: %q
api_min_retry_pause: %q
api_max_retry_pause: %q
max_concurrent_downloads: %d
zvuk_download_http:
  timeout: %q
  dial_timeout: %q
  tls_handshake_timeout: %q
  response_header_timeout: %q
  read_idle_timeout: %q
  receive_buffer: %q
  http1_only: %t
  max_retries: %d
  retry_initial_delay: %q
  retry_max_delay: %q
  resume: %t
yandex_music_download_http:
  timeout: %q
  dial_timeout: %q
  tls_handshake_timeout: %q
  response_header_timeout: %q
  read_idle_timeout: %q
  receive_buffer: %q
  http1_only: %t
  max_retries: %d
  retry_initial_delay: %q
  retry_max_delay: %q
  resume: %t
`,
		cfg.ZvukAuthToken,
		cfg.YandexMusicToken,
		cfg.Quality,
		cfg.MinQuality,
		cfg.MinDuration,
		cfg.MaxDuration,
		cfg.OutputPath,
		cfg.GroupByProvider,
		cfg.TrackFilenameTemplate,
		cfg.AlbumFolderTemplate,
		cfg.PlaylistFilenameTemplate,
		cfg.AudiobookFolderTemplate,
		cfg.AudiobookChapterFilenameTemplate,
		cfg.PodcastFolderTemplate,
		cfg.PodcastEpisodeFilenameTemplate,
		cfg.DownloadLyrics,
		cfg.ReplaceTracks,
		cfg.ReplaceCovers,
		cfg.ReplaceDescriptions,
		cfg.ReplaceLyrics,
		cfg.LogLevel,
		cfg.DownloadSpeedLimit,
		cfg.CreateFolderForSingles,
		cfg.MaxFolderNameLength,
		cfg.APIRetryAttemptsCount,
		cfg.MaxDownloadPause,
		cfg.APIMinRetryPause,
		cfg.APIMaxRetryPause,
		cfg.MaxConcurrentDownloads,
		httpCfg.Timeout.String(),
		httpCfg.DialTimeout.String(),
		httpCfg.TLSHandshakeTimeout.String(),
		httpCfg.ResponseHeaderTimeout.String(),
		httpCfg.ReadIdleTimeout.String(),
		httpCfg.ReceiveBuffer,
		httpCfg.HTTP1Only,
		httpCfg.MaxRetries,
		httpCfg.RetryInitialDelay.String(),
		httpCfg.RetryMaxDelay.String(),
		httpCfg.Resume,
		yandexHTTP.Timeout.String(),
		yandexHTTP.DialTimeout.String(),
		yandexHTTP.TLSHandshakeTimeout.String(),
		yandexHTTP.ResponseHeaderTimeout.String(),
		yandexHTTP.ReadIdleTimeout.String(),
		yandexHTTP.ReceiveBuffer,
		yandexHTTP.HTTP1Only,
		yandexHTTP.MaxRetries,
		yandexHTTP.RetryInitialDelay.String(),
		yandexHTTP.RetryMaxDelay.String(),
		yandexHTTP.Resume,
	)
}

// updateFieldsInNode updates or appends scalar string values in the YAML node tree.
func updateFieldsInNode(node *yaml.Node, fields map[string]string) {
	// The root node is a document node, content[0] is the actual map.
	if len(node.Content) == 0 || node.Content[0].Kind != yaml.MappingNode {
		return
	}

	mapNode := node.Content[0]
	seen := make(map[string]struct{}, len(fields))

	// Iterate through key-value pairs (stored as alternating nodes).
	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valueNode := mapNode.Content[i+1]

		value, ok := fields[keyNode.Value]
		if !ok {
			continue
		}

		valueNode.Value = value

		valueNode.Tag = yamlStringTag
		if valueNode.Style == 0 {
			valueNode.Style = yaml.DoubleQuotedStyle
		}

		seen[keyNode.Value] = struct{}{}
	}

	for key, value := range fields {
		if _, ok := seen[key]; ok {
			continue
		}

		mapNode.Content = append(mapNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   yamlStringTag,
			Value: key,
		}, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   yamlStringTag,
			Value: value,
			Style: yaml.DoubleQuotedStyle,
		})
	}
}
