package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
)

// Config holds all configuration settings.
type Config struct {
	AuthToken                string `mapstructure:"auth_token"`
	DownloadFormat           uint8  `mapstructure:"download_format"`
	OutputPath               string `mapstructure:"output_path"`
	TrackFilenameTemplate    string `mapstructure:"track_filename_template"`
	AlbumFolderTemplate      string `mapstructure:"album_folder_template"`
	PlaylistFilenameTemplate string `mapstructure:"playlist_filename_template"`
	DownloadLyrics           bool   `mapstructure:"download_lyrics"`
	ReplaceTracks            bool   `mapstructure:"replace_tracks"`
	ReplaceCovers            bool   `mapstructure:"replace_covers"`
	ReplaceLyrics            bool   `mapstructure:"replace_lyrics"`
	LogLevel                 string `mapstructure:"log_level"`
	DownloadSpeedLimit       string `mapstructure:"download_speed_limit"`
	CreateFolderForSingles   bool   `mapstructure:"create_folder_for_singles"`
	MaxFolderNameLength      int64  `mapstructure:"max_folder_name_length"`
	RetryAttemptsCount       int64  `mapstructure:"retry_attempts_count"`
	MaxDownloadPause         string `mapstructure:"max_download_pause"`
	MinRetryPause            string `mapstructure:"min_retry_pause"`
	MaxRetryPause            string `mapstructure:"max_retry_pause"`
	ZvukBaseURL              string
	ParsedDownloadSpeedLimit int64
	ParsedLogLevel           zapcore.Level
	ParsedMaxDownloadPause   time.Duration
	ParsedMinRetryPause      time.Duration
	ParsedMaxRetryPause      time.Duration
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

	minDownloadFormat = 1
	maxDownloadFormat = 3
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

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ValidateConfig checks the configuration for validity and sets derived fields.
func ValidateConfig(cfg *Config) error {
	var (
		downloadSpeedLimit       = strings.TrimSpace(cfg.DownloadSpeedLimit)
		parsedDownloadSpeedLimit uint64
		err                      error
	)

	authToken := strings.TrimSpace(cfg.AuthToken)
	if authToken == "" {
		return errors.New("authentication token cannot be empty")
	}

	cfg.ZvukBaseURL = ZvukBaseURL

	if downloadSpeedLimit != "" && downloadSpeedLimit != "0" {
		parsedDownloadSpeedLimit, err = humanize.ParseBytes(downloadSpeedLimit)
		if err != nil {
			return fmt.Errorf("failed to parse download speed limit: %w", err)
		}
	}

	// io.CopyN accepts only int64 so we transform it safely in order to use it later.
	cfg.ParsedDownloadSpeedLimit = utils.SafeUint64ToInt64(parsedDownloadSpeedLimit)

	if cfg.DownloadFormat < minDownloadFormat || cfg.DownloadFormat > maxDownloadFormat {
		return fmt.Errorf("format must be between %d and %d", minDownloadFormat, maxDownloadFormat)
	}

	parsedLogLevel, isLogLevelCorrect := logger.ParseLogLevel(cfg.LogLevel)
	if !(isLogLevelCorrect) {
		return fmt.Errorf("unknown log level: '%s'", cfg.LogLevel)
	}

	cfg.ParsedLogLevel = parsedLogLevel

	if cfg.RetryAttemptsCount <= 0 {
		return errors.New("retry attempts count must a positive integer")
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
