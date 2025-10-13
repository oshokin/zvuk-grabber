package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/oshokin/zvuk-grabber/internal/constants"
)

// TestConfigStruct tests the Config struct fields.
func TestConfigStruct(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		AuthToken:                "test_token",
		Quality:                  2,
		OutputPath:               "/tmp/downloads",
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
		DownloadLyrics:           true,
		ReplaceTracks:            false,
		ReplaceCovers:            false,
		ReplaceDescriptions:      false,
		ReplaceLyrics:            false,
		LogLevel:                 "info",
		DownloadSpeedLimit:       "1MB",
		CreateFolderForSingles:   true,
		MaxFolderNameLength:      100,
		RetryAttemptsCount:       3,
		MaxDownloadPause:         "5s",
		MinRetryPause:            "1s",
		MaxRetryPause:            "3s",
		MaxConcurrentDownloads:   1,
	}

	assert.Equal(t, "test_token", cfg.AuthToken)
	assert.Equal(t, uint8(2), cfg.Quality)
	assert.Equal(t, "/tmp/downloads", cfg.OutputPath)
	assert.Equal(t, "{{.trackNumberPad}} - {{.trackTitle}}", cfg.TrackFilenameTemplate)
	assert.Equal(t, "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}", cfg.AlbumFolderTemplate)
	assert.Equal(t, "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}", cfg.PlaylistFilenameTemplate)
	assert.True(t, cfg.DownloadLyrics)
	assert.False(t, cfg.ReplaceTracks)
	assert.False(t, cfg.ReplaceCovers)
	assert.False(t, cfg.ReplaceDescriptions)
	assert.False(t, cfg.ReplaceLyrics)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "1MB", cfg.DownloadSpeedLimit)
	assert.True(t, cfg.CreateFolderForSingles)
	assert.Equal(t, int64(100), cfg.MaxFolderNameLength)
	assert.Equal(t, int64(3), cfg.RetryAttemptsCount)
	assert.Equal(t, "5s", cfg.MaxDownloadPause)
	assert.Equal(t, "1s", cfg.MinRetryPause)
	assert.Equal(t, "3s", cfg.MaxRetryPause)
	assert.Equal(t, int64(1), cfg.MaxConcurrentDownloads)
}

// TestConstants tests the constants.
func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1024*1024, DefaultMaxLogLength)
	assert.Equal(t, 1, minQuality)
	assert.Equal(t, 3, maxQuality)
}

// TestLoadConfig tests the LoadConfig function.
//
//nolint:tparallel // It's a test function and it's not parallel to avoid race conditions.
func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configFilename string
		configContent  string
		expectError    bool
		expectedError  string
	}{
		{
			name:           "valid config file",
			configFilename: "valid_config.yaml",
			configContent: `
auth_token: "test_token"
quality: 2
min_quality: 0
output_path: "/tmp/downloads"
track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
download_lyrics: true
replace_tracks: false
replace_covers: false
replace_lyrics: false
log_level: "info"
download_speed_limit: "1MB"
create_folder_for_singles: true
max_folder_name_length: 100
retry_attempts_count: 3
max_download_pause: "5s"
min_retry_pause: "1s"
max_retry_pause: "3s"
`,
			expectError: false,
		},
		{
			name:           "non-existent file",
			configFilename: "non_existent.yaml",
			expectError:    true,
			expectedError:  "failed to read config from file",
		},
		{
			name:           "invalid yaml",
			configFilename: "invalid.yaml",
			configContent: `
invalid: yaml: content: [unclosed
`,
			expectError:   true,
			expectedError: "failed to read config from file",
		},
		{
			name:           "empty filename uses default",
			configFilename: "",
			expectError:    true,
			expectedError:  "failed to read config from file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test.
			var (
				tempDir    = t.TempDir()
				configPath string
			)

			switch {
			case tt.configContent != "":
				configPath = filepath.Join(tempDir, tt.configFilename)
				err := os.WriteFile(configPath, []byte(tt.configContent), constants.DefaultFilePermissions)

				require.NoError(t, err)
			case tt.configFilename != "":
				configPath = filepath.Join(tempDir, tt.configFilename)
			default:
				// For empty filename test, use a non-existent file path.
				configPath = filepath.Join(tempDir, "non_existent.yaml")
			}

			cfg, err := LoadConfig(configPath)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, "test_token", cfg.AuthToken)
				assert.Equal(t, uint8(2), cfg.Quality)
			}
		})
	}
}

// TestValidateConfig tests the ValidateConfig function.
//
//nolint:tparallel // It's a test function and it's not parallel to avoid race conditions.
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                2,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: false,
		},
		{
			name: "empty auth token",
			config: &Config{
				AuthToken:              "",
				Quality:                2,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "authentication token cannot be empty",
		},
		{
			name: "whitespace auth token",
			config: &Config{
				AuthToken:              "   ",
				Quality:                2,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "authentication token cannot be empty",
		},
		{
			name: "invalid quality - too low",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                0,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "invalid quality: must be between",
		},
		{
			name: "invalid quality - too high",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                4,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "invalid quality: must be between",
		},
		{
			name: "invalid log level",
			config: &Config{
				AuthToken:          "valid_token",
				Quality:            2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "invalid",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "unknown log level:",
		},
		{
			name: "invalid retry attempts count",
			config: &Config{
				AuthToken:          "valid_token",
				Quality:            2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 0,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "retry attempts count must a positive integer",
		},
		{
			name: "invalid max download pause",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                2,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "invalid",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "failed to parse max download pause:",
		},
		{
			name: "invalid min retry pause",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                2,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "invalid",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "failed to parse min retry pause:",
		},
		{
			name: "invalid max retry pause",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                2,
				DownloadSpeedLimit:     "1MB",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "invalid",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "failed to parse max retry pause:",
		},
		{
			name: "invalid download speed limit",
			config: &Config{
				AuthToken:              "valid_token",
				Quality:                2,
				DownloadSpeedLimit:     "invalid",
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			},
			expectError: true,
			errorMsg:    "failed to parse download speed limit:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateConfig(tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				// Check that parsed values are set correctly.
				assert.Equal(t, zapcore.InfoLevel, tt.config.ParsedLogLevel)
			}
		})
	}
}

// TestValidateConfig_DownloadSpeedLimit tests download speed limit validation.
//
//nolint:tparallel // It's a test function and it's not parallel to avoid race conditions.
func TestValidateConfig_DownloadSpeedLimit(t *testing.T) {
	tests := []struct {
		name          string
		speedLimit    string
		expectedBytes int64
		expectError   bool
	}{
		{
			name:          "empty limit",
			speedLimit:    "",
			expectedBytes: 0,
			expectError:   false,
		},
		{
			name:          "zero limit",
			speedLimit:    "0",
			expectedBytes: 0,
			expectError:   false,
		},
		{
			name:          "1KB limit",
			speedLimit:    "1KB",
			expectedBytes: 1000,
			expectError:   false,
		},
		{
			name:          "1MB limit",
			speedLimit:    "1MB",
			expectedBytes: 1000000,
			expectError:   false,
		},
		{
			name:          "1GB limit",
			speedLimit:    "1GB",
			expectedBytes: 1000000000,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{
				AuthToken:              "valid_token",
				Quality:                2,
				DownloadSpeedLimit:     tt.speedLimit,
				LogLevel:               "info",
				RetryAttemptsCount:     3,
				MaxDownloadPause:       "5s",
				MinRetryPause:          "1s",
				MaxRetryPause:          "3s",
				MaxConcurrentDownloads: 1,
			}

			err := ValidateConfig(config)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBytes, config.ParsedDownloadSpeedLimit)
			}
		})
	}
}

// TestConfigValidation_MinQuality tests min_quality validation rules.
func TestConfigValidation_MinQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		quality       uint8
		minQuality    uint8
		expectError   bool
		errorContains string
	}{
		{
			name:        "min_quality 0 is valid (disabled)",
			quality:     3,
			minQuality:  0,
			expectError: false,
		},
		{
			name:        "min_quality equals quality",
			quality:     3,
			minQuality:  3,
			expectError: false,
		},
		{
			name:        "min_quality less than quality",
			quality:     3,
			minQuality:  2,
			expectError: false,
		},
		{
			name:          "min_quality too high (4)",
			quality:       3,
			minQuality:    4,
			expectError:   true,
			errorContains: "invalid min_quality",
		},
		{
			name:          "min_quality greater than quality",
			quality:       2,
			minQuality:    3,
			expectError:   true,
			errorContains: "min_quality cannot be higher than quality",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				AuthToken:                "valid_token",
				Quality:                  tt.quality,
				MinQuality:               tt.minQuality,
				OutputPath:               "/tmp",
				TrackFilenameTemplate:    "{{.trackTitle}}",
				AlbumFolderTemplate:      "{{.albumTitle}}",
				PlaylistFilenameTemplate: "{{.trackTitle}}",
				LogLevel:                 "info",
				DownloadSpeedLimit:       "",
				RetryAttemptsCount:       1,
				MaxDownloadPause:         "1s",
				MinRetryPause:            "1s",
				MaxRetryPause:            "5s",
				MaxConcurrentDownloads:   1,
			}

			err := ValidateConfig(cfg)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfigValidation_DurationSettings tests min_duration and max_duration validation.
func TestConfigValidation_DurationSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		minDuration   string
		maxDuration   string
		expectError   bool
		errorContains string
	}{
		{
			name:        "No duration filtering (both empty)",
			minDuration: "",
			maxDuration: "",
			expectError: false,
		},
		{
			name:        "Only min_duration set",
			minDuration: "30s",
			maxDuration: "",
			expectError: false,
		},
		{
			name:        "Only max_duration set",
			minDuration: "",
			maxDuration: "10m",
			expectError: false,
		},
		{
			name:        "Both set with valid range",
			minDuration: "30s",
			maxDuration: "10m",
			expectError: false,
		},
		{
			name:        "Both set with 1s difference",
			minDuration: "1m",
			maxDuration: "1m1s",
			expectError: false,
		},
		{
			name:          "Invalid min_duration format",
			minDuration:   "invalid",
			maxDuration:   "",
			expectError:   true,
			errorContains: "failed to parse min duration",
		},
		{
			name:          "Invalid max_duration format",
			minDuration:   "",
			maxDuration:   "notaduration",
			expectError:   true,
			errorContains: "failed to parse max duration",
		},
		{
			name:          "Negative min_duration",
			minDuration:   "-30s",
			maxDuration:   "",
			expectError:   true,
			errorContains: "min_duration must be positive",
		},
		{
			name:          "Zero max_duration",
			minDuration:   "",
			maxDuration:   "0s",
			expectError:   true,
			errorContains: "max_duration must be positive",
		},
		{
			name:          "Negative max_duration",
			minDuration:   "",
			maxDuration:   "-1m",
			expectError:   true,
			errorContains: "max_duration must be positive",
		},
		{
			name:          "max_duration equals min_duration",
			minDuration:   "5m",
			maxDuration:   "5m",
			expectError:   true,
			errorContains: "max_duration must be greater than min_duration",
		},
		{
			name:          "max_duration less than min_duration",
			minDuration:   "10m",
			maxDuration:   "5m",
			expectError:   true,
			errorContains: "max_duration must be greater than min_duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				AuthToken:                "valid_token",
				Quality:                  2,
				MinQuality:               0,
				MinDuration:              tt.minDuration,
				MaxDuration:              tt.maxDuration,
				OutputPath:               "/tmp",
				TrackFilenameTemplate:    "{{.trackTitle}}",
				AlbumFolderTemplate:      "{{.albumTitle}}",
				PlaylistFilenameTemplate: "{{.trackTitle}}",
				LogLevel:                 "info",
				DownloadSpeedLimit:       "",
				RetryAttemptsCount:       1,
				MaxDownloadPause:         "1s",
				MinRetryPause:            "1s",
				MaxRetryPause:            "5s",
				MaxConcurrentDownloads:   1,
			}

			err := ValidateConfig(cfg)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)

				// Verify parsed values if set.
				if tt.minDuration != "" {
					expected, parseErr := time.ParseDuration(tt.minDuration)
					require.NoError(t, parseErr, "Test duration string should be valid")
					assert.Equal(t, expected, cfg.ParsedMinDuration)
				}

				if tt.maxDuration != "" {
					expected, parseErr := time.ParseDuration(tt.maxDuration)
					require.NoError(t, parseErr, "Test duration string should be valid")
					assert.Equal(t, expected, cfg.ParsedMaxDuration)
				}
			}
		})
	}
}

// TestConfigValidation_PauseDurations tests validation of all pause/retry duration settings.
func TestConfigValidation_PauseDurations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		maxDownloadPause string
		minRetryPause    string
		maxRetryPause    string
		expectError      bool
		errorContains    string
	}{
		{
			name:             "Valid durations",
			maxDownloadPause: "2s",
			minRetryPause:    "1s",
			maxRetryPause:    "5s",
			expectError:      false,
		},
		{
			name:             "Zero max_download_pause",
			maxDownloadPause: "0s",
			minRetryPause:    "1s",
			maxRetryPause:    "5s",
			expectError:      true,
			errorContains:    "max_download_pause must be positive",
		},
		{
			name:             "Negative max_download_pause",
			maxDownloadPause: "-1s",
			minRetryPause:    "1s",
			maxRetryPause:    "5s",
			expectError:      true,
			errorContains:    "max_download_pause must be positive",
		},
		{
			name:             "Zero min_retry_pause",
			maxDownloadPause: "2s",
			minRetryPause:    "0s",
			maxRetryPause:    "5s",
			expectError:      true,
			errorContains:    "min_retry_pause must be positive",
		},
		{
			name:             "Negative min_retry_pause",
			maxDownloadPause: "2s",
			minRetryPause:    "-1s",
			maxRetryPause:    "5s",
			expectError:      true,
			errorContains:    "min_retry_pause must be positive",
		},
		{
			name:             "Zero max_retry_pause",
			maxDownloadPause: "2s",
			minRetryPause:    "1s",
			maxRetryPause:    "0s",
			expectError:      true,
			errorContains:    "max_retry_pause must be positive",
		},
		{
			name:             "Negative max_retry_pause",
			maxDownloadPause: "2s",
			minRetryPause:    "1s",
			maxRetryPause:    "-5s",
			expectError:      true,
			errorContains:    "max_retry_pause must be positive",
		},
		{
			name:             "Invalid max_download_pause format",
			maxDownloadPause: "invalid",
			minRetryPause:    "1s",
			maxRetryPause:    "5s",
			expectError:      true,
			errorContains:    "failed to parse max download pause",
		},
		{
			name:             "Invalid min_retry_pause format",
			maxDownloadPause: "2s",
			minRetryPause:    "notaduration",
			maxRetryPause:    "5s",
			expectError:      true,
			errorContains:    "failed to parse min retry pause",
		},
		{
			name:             "Invalid max_retry_pause format",
			maxDownloadPause: "2s",
			minRetryPause:    "1s",
			maxRetryPause:    "xyz",
			expectError:      true,
			errorContains:    "failed to parse max retry pause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				AuthToken:                "valid_token",
				Quality:                  2,
				MinQuality:               0,
				OutputPath:               "/tmp",
				TrackFilenameTemplate:    "{{.trackTitle}}",
				AlbumFolderTemplate:      "{{.albumTitle}}",
				PlaylistFilenameTemplate: "{{.trackTitle}}",
				LogLevel:                 "info",
				DownloadSpeedLimit:       "",
				RetryAttemptsCount:       1,
				MaxDownloadPause:         tt.maxDownloadPause,
				MinRetryPause:            tt.minRetryPause,
				MaxRetryPause:            tt.maxRetryPause,
				MaxConcurrentDownloads:   1,
			}

			err := ValidateConfig(cfg)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)

				// Verify parsed values.
				expectedMaxDownload, parseErr := time.ParseDuration(tt.maxDownloadPause)
				require.NoError(t, parseErr)
				expectedMinRetry, parseErr := time.ParseDuration(tt.minRetryPause)
				require.NoError(t, parseErr)
				expectedMaxRetry, parseErr := time.ParseDuration(tt.maxRetryPause)
				require.NoError(t, parseErr)

				assert.Equal(t, expectedMaxDownload, cfg.ParsedMaxDownloadPause)
				assert.Equal(t, expectedMinRetry, cfg.ParsedMinRetryPause)
				assert.Equal(t, expectedMaxRetry, cfg.ParsedMaxRetryPause)
			}
		})
	}
}
