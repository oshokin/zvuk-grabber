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

	"github.com/oshokin/zvuk-grabber/internal/constants"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Config holds all configuration settings.
type Config struct {
	// AuthToken is the authentication token for API access.
	AuthToken string `mapstructure:"auth_token"`
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

	// minQuality is the minimum valid quality value.
	minQuality = 1
	// maxQuality is the maximum valid quality value.
	maxQuality = 3
)

// Static error definitions for better error handling.
var (
	// ErrEmptyAuthToken indicates that the authentication token is missing.
	ErrEmptyAuthToken = errors.New("authentication token cannot be empty")
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
	// ErrInvalidRetryAttempts indicates that the retry attempts count is invalid.
	ErrInvalidRetryAttempts = errors.New("retry attempts count must a positive integer")
	// ErrInvalidMaxDownloadPause indicates that the max download pause duration is invalid.
	ErrInvalidMaxDownloadPause = errors.New("max_download_pause must be positive")
	// ErrInvalidMinRetryPause indicates that the min retry pause duration is invalid.
	ErrInvalidMinRetryPause = errors.New("min_retry_pause must be positive")
	// ErrInvalidMaxRetryPause indicates that the max retry pause duration is invalid.
	ErrInvalidMaxRetryPause = errors.New("max_retry_pause must be positive")
	// ErrInvalidConcurrentDownloads indicates that the concurrent downloads count is invalid.
	ErrInvalidConcurrentDownloads = errors.New("max concurrent downloads must be a positive integer")
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
//nolint:funlen,gocognit,cyclop // Validation functions naturally have high complexity and length due to sequential checks.
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

	if cfg.Quality < minQuality || cfg.Quality > maxQuality {
		return fmt.Errorf("%w: must be between %d and %d", ErrInvalidQuality, minQuality, maxQuality)
	}

	// Validate min_quality if set (0 means no filtering).
	if cfg.MinQuality > 0 {
		if cfg.MinQuality < minQuality || cfg.MinQuality > maxQuality {
			return fmt.Errorf("%w: must be between %d and %d, or 0 to disable",
				ErrInvalidMinQuality, minQuality, maxQuality)
		}

		if cfg.MinQuality > cfg.Quality {
			return ErrMinQualityTooHigh
		}
	}

	// Parse min_duration if set (empty string means no filtering).
	if cfg.MinDuration != "" {
		cfg.ParsedMinDuration, err = time.ParseDuration(cfg.MinDuration)
		if err != nil {
			return fmt.Errorf("failed to parse min duration: %w", err)
		}

		if cfg.ParsedMinDuration <= 0 {
			return ErrInvalidMinDuration
		}
	}

	// Parse max_duration if set (empty string means no filtering).
	if cfg.MaxDuration != "" {
		cfg.ParsedMaxDuration, err = time.ParseDuration(cfg.MaxDuration)
		if err != nil {
			return fmt.Errorf("failed to parse max duration: %w", err)
		}

		if cfg.ParsedMaxDuration <= 0 {
			return ErrInvalidMaxDuration
		}

		// Validate that max_duration > min_duration if both are set.
		if cfg.MinDuration != "" && cfg.ParsedMaxDuration <= cfg.ParsedMinDuration {
			return ErrMaxDurationTooLow
		}
	}

	parsedLogLevel, isLogLevelCorrect := logger.ParseLogLevel(cfg.LogLevel)
	if !(isLogLevelCorrect) {
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

	if cfg.RetryAttemptsCount <= 0 {
		return ErrInvalidRetryAttempts
	}

	cfg.ParsedMaxDownloadPause, err = time.ParseDuration(cfg.MaxDownloadPause)
	if err != nil {
		return fmt.Errorf("failed to parse max download pause: %w", err)
	}

	if cfg.ParsedMaxDownloadPause <= 0 {
		return ErrInvalidMaxDownloadPause
	}

	cfg.ParsedMinRetryPause, err = time.ParseDuration(cfg.MinRetryPause)
	if err != nil {
		return fmt.Errorf("failed to parse min retry pause: %w", err)
	}

	if cfg.ParsedMinRetryPause <= 0 {
		return ErrInvalidMinRetryPause
	}

	cfg.ParsedMaxRetryPause, err = time.ParseDuration(cfg.MaxRetryPause)
	if err != nil {
		return fmt.Errorf("failed to parse max retry pause: %w", err)
	}

	if cfg.ParsedMaxRetryPause <= 0 {
		return ErrInvalidMaxRetryPause
	}

	if cfg.MaxConcurrentDownloads <= 0 {
		return ErrInvalidConcurrentDownloads
	}

	return nil
}

// SaveConfig saves the configuration to the file while preserving the original format and order.
func SaveConfig(cfg *Config) error {
	configFile := getConfigFilePath()

	// Read the original file content.
	originalContent, err := os.ReadFile(configFile)
	if err != nil {
		return handleMissingConfigFile(configFile, cfg.AuthToken, err)
	}

	// Parse YAML while preserving order using yaml.Node.
	var node yaml.Node
	if err = yaml.Unmarshal(originalContent, &node); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Update the auth_token value in the node tree.
	updateAuthTokenInNode(&node, cfg.AuthToken)

	// Marshal back to YAML (preserves order).
	newContent, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write the file back with preserved order.
	if err = os.WriteFile(configFile, newContent, constants.DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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
func handleMissingConfigFile(configFile, authToken string, err error) error {
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// File doesn't exist, create it with viper.
	viper.Set("auth_token", authToken)

	if err = viper.SafeWriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	return nil
}

// updateAuthTokenInNode updates the auth_token value in the YAML node tree.
func updateAuthTokenInNode(node *yaml.Node, authToken string) {
	// The root node is a document node, content[0] is the actual map.
	if len(node.Content) == 0 || node.Content[0].Kind != yaml.MappingNode {
		return
	}

	mapNode := node.Content[0]

	// Iterate through key-value pairs (stored as alternating nodes).
	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valueNode := mapNode.Content[i+1]

		if keyNode.Value == "auth_token" {
			// Update the value while preserving style.
			valueNode.Value = authToken

			// Ensure it's quoted if it contains special characters.
			if valueNode.Style == 0 {
				valueNode.Style = yaml.DoubleQuotedStyle
			}

			break
		}
	}
}
