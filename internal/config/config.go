package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"

	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Config holds all configuration settings.
type Config struct {
	// AuthToken is the authentication token for API access.
	AuthToken string `mapstructure:"auth_token"`
	// DownloadFormat specifies the audio quality/format (1=MP3 128k, 2=MP3 320k, 3=FLAC).
	DownloadFormat uint8 `mapstructure:"download_format"`
	// OutputPath is the directory path where downloaded files will be saved.
	OutputPath string `mapstructure:"output_path"`
	// TrackFilenameTemplate is the template for naming individual track files.
	TrackFilenameTemplate string `mapstructure:"track_filename_template"`
	// AlbumFolderTemplate is the template for naming album folders.
	AlbumFolderTemplate string `mapstructure:"album_folder_template"`
	// PlaylistFilenameTemplate is the template for naming playlist track files.
	PlaylistFilenameTemplate string `mapstructure:"playlist_filename_template"`
	// DownloadLyrics indicates whether to download lyrics for tracks.
	DownloadLyrics bool `mapstructure:"download_lyrics"`
	// ReplaceTracks indicates whether to replace existing track files.
	ReplaceTracks bool `mapstructure:"replace_tracks"`
	// ReplaceCovers indicates whether to replace existing cover art files.
	ReplaceCovers bool `mapstructure:"replace_covers"`
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
	// RetryAttemptsCount is the number of retry attempts for failed downloads.
	RetryAttemptsCount int64 `mapstructure:"retry_attempts_count"`
	// MaxDownloadPause is the maximum pause duration between downloads.
	MaxDownloadPause string `mapstructure:"max_download_pause"`
	// MinRetryPause is the minimum pause duration before retrying.
	MinRetryPause string `mapstructure:"min_retry_pause"`
	// MaxRetryPause is the maximum pause duration before retrying.
	MaxRetryPause string `mapstructure:"max_retry_pause"`
	// ZvukBaseURL is the base URL for the Zvuk API (set automatically).
	ZvukBaseURL string
	// ParsedDownloadSpeedLimit is the parsed download speed limit in bytes.
	ParsedDownloadSpeedLimit int64
	// ParsedLogLevel is the parsed zap log level.
	ParsedLogLevel zapcore.Level
	// ParsedMaxDownloadPause is the parsed maximum download pause duration.
	ParsedMaxDownloadPause time.Duration
	// ParsedMinRetryPause is the parsed minimum retry pause duration.
	ParsedMinRetryPause time.Duration
	// ParsedMaxRetryPause is the parsed maximum retry pause duration.
	ParsedMaxRetryPause time.Duration
}

const (
	// ZvukBaseURL is the base URL for the Zvuk service.
	ZvukBaseURL = "https://zvuk.com"

	// DefaultConfigFilename is the default name of the configuration file.
	DefaultConfigFilename = ".zvuk-grabber.yaml"

	// DefaultTrackFilenameTemplate is the default template for naming downloaded track files.
	DefaultTrackFilenameTemplate = "{{.trackNumberPad}} - {{.trackTitle}}"

	// DefaultAlbumFolderTemplate is the default template for naming folders for downloaded albums.
	DefaultAlbumFolderTemplate = "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"

	// DefaultPlaylistFilenameTemplate is the default template for naming downloaded track files from playlists.
	DefaultPlaylistFilenameTemplate = "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"

	// DefaultMaxLogLength is the default maximum size (in bytes) for log files.
	DefaultMaxLogLength = 1 * 1024 * 1024 // 1 MB

	// minDownloadFormat is the minimum valid download format value.
	minDownloadFormat = 1
	// maxDownloadFormat is the maximum valid download format value.
	maxDownloadFormat = 3
)

// Static error definitions for better error handling.
var (
	// ErrEmptyAuthToken indicates that the authentication token is missing.
	ErrEmptyAuthToken = errors.New("authentication token cannot be empty")
	// ErrInvalidFormat indicates that the download format is invalid.
	ErrInvalidFormat = errors.New("invalid format")
	// ErrUnknownLogLevel indicates that the log level is not recognized.
	ErrUnknownLogLevel = errors.New("unknown log level")
	// ErrInvalidRetryAttempts indicates that the retry attempts count is invalid.
	ErrInvalidRetryAttempts = errors.New("retry attempts count must a positive integer")
)

// LoadConfig loads configuration settings from a YAML file.
func LoadConfig(configFilename string) (*Config, error) {
	if configFilename == "" {
		configFilename = DefaultConfigFilename
	}

	viper.SetConfigFile(configFilename)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config from file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// ValidateConfig checks the configuration for validity and sets derived fields.
//
//nolint:cyclop // It's a validation function, not a complex one.
func ValidateConfig(cfg *Config) error {
	var (
		downloadSpeedLimit       = strings.TrimSpace(cfg.DownloadSpeedLimit)
		parsedDownloadSpeedLimit uint64
		err                      error
	)

	authToken := strings.TrimSpace(cfg.AuthToken)
	if authToken == "" {
		return ErrEmptyAuthToken
	}

	cfg.ZvukBaseURL = ZvukBaseURL

	if downloadSpeedLimit != "" && downloadSpeedLimit != "0" {
		parsedDownloadSpeedLimit, err = humanize.ParseBytes(downloadSpeedLimit)
		if err != nil {
			return fmt.Errorf("failed to parse download speed limit: %w", err)
		}
	}

	// Io.CopyN accepts only int64 so we transform it safely in order to use it later.
	cfg.ParsedDownloadSpeedLimit = utils.SafeUint64ToInt64(parsedDownloadSpeedLimit)

	if cfg.DownloadFormat < minDownloadFormat || cfg.DownloadFormat > maxDownloadFormat {
		return fmt.Errorf("%w: must be between %d and %d", ErrInvalidFormat, minDownloadFormat, maxDownloadFormat)
	}

	parsedLogLevel, isLogLevelCorrect := logger.ParseLogLevel(cfg.LogLevel)
	if !(isLogLevelCorrect) {
		return fmt.Errorf("%w: '%s'", ErrUnknownLogLevel, cfg.LogLevel)
	}

	cfg.ParsedLogLevel = parsedLogLevel

	if cfg.RetryAttemptsCount <= 0 {
		return ErrInvalidRetryAttempts
	}

	cfg.ParsedMaxDownloadPause, err = time.ParseDuration(cfg.MaxDownloadPause)
	if err != nil {
		return fmt.Errorf("failed to parse max download pause: %w", err)
	}

	cfg.ParsedMinRetryPause, err = time.ParseDuration(cfg.MinRetryPause)
	if err != nil {
		return fmt.Errorf("failed to parse min retry pause: %w", err)
	}

	cfg.ParsedMaxRetryPause, err = time.ParseDuration(cfg.MaxRetryPause)
	if err != nil {
		return fmt.Errorf("failed to parse max retry pause: %w", err)
	}

	return nil
}
