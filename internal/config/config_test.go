package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// TestConfigStruct tests the Config struct fields.
func TestConfigStruct(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		AuthToken:                "test_token",
		DownloadFormat:           2,
		OutputPath:               "/tmp/downloads",
		TrackFilenameTemplate:    "{{.trackNumberPad}} - {{.trackTitle}}",
		AlbumFolderTemplate:      "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}",
		PlaylistFilenameTemplate: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}",
		DownloadLyrics:           true,
		ReplaceTracks:            false,
		ReplaceCovers:            false,
		ReplaceLyrics:            false,
		LogLevel:                 "info",
		DownloadSpeedLimit:       "1MB",
		CreateFolderForSingles:   true,
		MaxFolderNameLength:      100,
		RetryAttemptsCount:       3,
		MaxDownloadPause:         "5s",
		MinRetryPause:            "1s",
		MaxRetryPause:            "3s",
	}

	assert.Equal(t, "test_token", cfg.AuthToken)
	assert.Equal(t, uint8(2), cfg.DownloadFormat)
	assert.Equal(t, "/tmp/downloads", cfg.OutputPath)
	assert.Equal(t, "{{.trackNumberPad}} - {{.trackTitle}}", cfg.TrackFilenameTemplate)
	assert.Equal(t, "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}", cfg.AlbumFolderTemplate)
	assert.Equal(t, "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}", cfg.PlaylistFilenameTemplate)
	assert.True(t, cfg.DownloadLyrics)
	assert.False(t, cfg.ReplaceTracks)
	assert.False(t, cfg.ReplaceCovers)
	assert.False(t, cfg.ReplaceLyrics)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "1MB", cfg.DownloadSpeedLimit)
	assert.True(t, cfg.CreateFolderForSingles)
	assert.Equal(t, int64(100), cfg.MaxFolderNameLength)
	assert.Equal(t, int64(3), cfg.RetryAttemptsCount)
	assert.Equal(t, "5s", cfg.MaxDownloadPause)
	assert.Equal(t, "1s", cfg.MinRetryPause)
	assert.Equal(t, "3s", cfg.MaxRetryPause)
}

// TestConstants tests the constants.
func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1024*1024, DefaultMaxLogLength)
	assert.Equal(t, 1, minDownloadFormat)
	assert.Equal(t, 3, maxDownloadFormat)
}

// TestLoadConfig tests the LoadConfig function.
//
//nolint:funlen, tparallel // It's a test function and it's not parallel to avoid race conditions.
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
download_format: 2
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
				err := os.WriteFile(configPath, []byte(tt.configContent), 0o644) //nolint:gosec // It's a test file.

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
				assert.Equal(t, uint8(2), cfg.DownloadFormat)
			}
		})
	}
}

// TestValidateConfig tests the ValidateConfig function.
//
//nolint:funlen, tparallel // It's a test function and it's not parallel to avoid race conditions.
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
				AuthToken:          "valid_token",
				DownloadFormat:     2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: false,
		},
		{
			name: "empty auth token",
			config: &Config{
				AuthToken:          "",
				DownloadFormat:     2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "authentication token cannot be empty",
		},
		{
			name: "whitespace auth token",
			config: &Config{
				AuthToken:          "   ",
				DownloadFormat:     2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "authentication token cannot be empty",
		},
		{
			name: "invalid download format - too low",
			config: &Config{
				AuthToken:          "valid_token",
				DownloadFormat:     0,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "invalid format: must be between",
		},
		{
			name: "invalid download format - too high",
			config: &Config{
				AuthToken:          "valid_token",
				DownloadFormat:     4,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "invalid format: must be between",
		},
		{
			name: "invalid log level",
			config: &Config{
				AuthToken:          "valid_token",
				DownloadFormat:     2,
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
				DownloadFormat:     2,
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
				AuthToken:          "valid_token",
				DownloadFormat:     2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "invalid",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "failed to parse max download pause:",
		},
		{
			name: "invalid min retry pause",
			config: &Config{
				AuthToken:          "valid_token",
				DownloadFormat:     2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "invalid",
				MaxRetryPause:      "3s",
			},
			expectError: true,
			errorMsg:    "failed to parse min retry pause:",
		},
		{
			name: "invalid max retry pause",
			config: &Config{
				AuthToken:          "valid_token",
				DownloadFormat:     2,
				DownloadSpeedLimit: "1MB",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "invalid",
			},
			expectError: true,
			errorMsg:    "failed to parse max retry pause:",
		},
		{
			name: "invalid download speed limit",
			config: &Config{
				AuthToken:          "valid_token",
				DownloadFormat:     2,
				DownloadSpeedLimit: "invalid",
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
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
//nolint:funlen, tparallel // It's a test function and it's not parallel to avoid race conditions.
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
				AuthToken:          "valid_token",
				DownloadFormat:     2,
				DownloadSpeedLimit: tt.speedLimit,
				LogLevel:           "info",
				RetryAttemptsCount: 3,
				MaxDownloadPause:   "5s",
				MinRetryPause:      "1s",
				MaxRetryPause:      "3s",
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
