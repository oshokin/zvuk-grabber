package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/constants"
)

const testBaseConfigContent = `
auth_token: "config_token"
quality: 1
min_quality: 0
output_path: "/config/output"
download_lyrics: false
download_speed_limit: "500KB"
log_level: "info"
track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
replace_tracks: false
replace_covers: false
replace_lyrics: false
create_folder_for_singles: false
max_folder_name_length: 100
retry_attempts_count: 3
max_download_pause: "5s"
min_retry_pause: "1s"
max_retry_pause: "3s"
max_concurrent_downloads: 1
`

// TestFlagOverrides tests that command-line flags correctly override configuration file values.
//
//nolint:funlen,nolintlint,tparallel // It's a comprehensive integration test. Cannot run in parallel due to Viper global state.
func TestFlagOverrides(t *testing.T) {
	tests := []struct {
		name           string
		flags          map[string]any
		expectedConfig func(*testing.T, *config.Config)
	}{
		{
			name:  "no flags - use config values",
			flags: map[string]any{},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "quality flag only - override quality",
			flags: map[string]any{
				"quality": 2,
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(2), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "output flag only - override output path",
			flags: map[string]any{
				"output": "/flag/output",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/flag/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "lyrics flag only - override lyrics",
			flags: map[string]any{
				"lyrics": true,
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "speed-limit flag only - override speed limit",
			flags: map[string]any{
				"speed-limit": "1MB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "1MB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "all flags - override everything",
			flags: map[string]any{
				"quality":     3,
				"output":      "/all/flags/output",
				"lyrics":      true,
				"speed-limit": "2MB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(3), cfg.Quality)
				assert.Equal(t, "/all/flags/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "2MB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "quality and output flags - partial override",
			flags: map[string]any{
				"quality": 2,
				"output":  "/partial/output",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(2), cfg.Quality)
				assert.Equal(t, "/partial/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "quality and lyrics flags - partial override",
			flags: map[string]any{
				"quality": 3,
				"lyrics":  true,
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(3), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "output and speed-limit flags - partial override",
			flags: map[string]any{
				"output":      "/speed/output",
				"speed-limit": "3MB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/speed/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "3MB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "lyrics and speed-limit flags - partial override",
			flags: map[string]any{
				"lyrics":      true,
				"speed-limit": "750KB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "750KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "quality, output, and lyrics flags - triple override",
			flags: map[string]any{
				"quality": 2,
				"output":  "/triple/output",
				"lyrics":  true,
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(2), cfg.Quality)
				assert.Equal(t, "/triple/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "500KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "quality, output, and speed-limit flags - triple override",
			flags: map[string]any{
				"quality":     1,
				"output":      "/speed-triple/output",
				"speed-limit": "1.5MB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/speed-triple/output", cfg.OutputPath)
				assert.False(t, cfg.DownloadLyrics)
				assert.Equal(t, "1.5MB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "quality, lyrics, and speed-limit flags - triple override",
			flags: map[string]any{
				"quality":     3,
				"lyrics":      true,
				"speed-limit": "2.5MB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(3), cfg.Quality)
				assert.Equal(t, "/config/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "2.5MB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "output, lyrics, and speed-limit flags - triple override",
			flags: map[string]any{
				"output":      "/another-triple/output",
				"lyrics":      true,
				"speed-limit": "100KB",
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, uint8(1), cfg.Quality)
				assert.Equal(t, "/another-triple/output", cfg.OutputPath)
				assert.True(t, cfg.DownloadLyrics)
				assert.Equal(t, "100KB", cfg.DownloadSpeedLimit)
			},
		},
		{
			name: "lyrics false flag - explicit false override",
			flags: map[string]any{
				"lyrics": false,
			},
			expectedConfig: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.False(t, cfg.DownloadLyrics)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")

			err := os.WriteFile(
				configPath,
				[]byte(testBaseConfigContent),
				constants.DefaultFilePermissions,
			) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Load configuration.
			cfg, err := config.LoadConfig(configPath)
			require.NoError(t, err)

			// Create a test command with flags.
			testCmd := &cobra.Command{
				Use: "test",
			}

			// Add the same flags as root command.
			testCmd.Flags().IntP("quality", "q", 1, "audio quality")
			testCmd.Flags().StringP("output", "o", "", "output directory")
			testCmd.Flags().BoolP("lyrics", "l", false, "include lyrics")
			testCmd.Flags().StringP("speed-limit", "s", "", "download speed limit")

			// Set flag values.
			for flagName, flagValue := range tt.flags {
				var setErr error

				switch v := flagValue.(type) {
				case int:
					setErr = testCmd.Flags().Set(flagName, string(rune(v+'0')))
				case string:
					setErr = testCmd.Flags().Set(flagName, v)
				case bool:
					if v {
						setErr = testCmd.Flags().Set(flagName, "true")
					} else {
						setErr = testCmd.Flags().Set(flagName, "false")
					}
				}

				require.NoError(t, setErr, "failed to set flag %s", flagName)
			}

			// Bind flags to config.
			err = bindFlagsToConfig(testCmd.Flags(), cfg)
			require.NoError(t, err)

			// Verify expectations.
			tt.expectedConfig(t, cfg)
		})
	}
}

// TestFlagOverrides_AllQualityValues tests all valid quality values (1, 2, 3).
//
//nolint:nolintlint,tparallel // Cannot run in parallel due to Viper global state.
func TestFlagOverrides_AllQualityValues(t *testing.T) {
	testBaseConfigContent := `
auth_token: "config_token"
quality: 1
min_quality: 0
output_path: "/config/output"
download_lyrics: false
download_speed_limit: "500KB"
log_level: "info"
track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
replace_tracks: false
replace_covers: false
replace_lyrics: false
create_folder_for_singles: false
max_folder_name_length: 100
retry_attempts_count: 3
max_download_pause: "5s"
min_retry_pause: "1s"
max_retry_pause: "3s"
max_concurrent_downloads: 1
`

	qualityTests := []struct {
		name           string
		qualityValue   int
		expectedFormat uint8
	}{
		{"quality 1 - MP3 128 Kbps", 1, 1},
		{"quality 2 - MP3 320 Kbps", 2, 2},
		{"quality 3 - FLAC 16-bit/44.1kHz", 3, 3},
	}

	for _, tt := range qualityTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")

			err := os.WriteFile(
				configPath,
				[]byte(testBaseConfigContent),
				constants.DefaultFilePermissions,
			) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Load configuration.
			cfg, err := config.LoadConfig(configPath)
			require.NoError(t, err)

			// Create a test command with flags.
			testCmd := &cobra.Command{Use: "test"}
			testCmd.Flags().IntP("quality", "q", 1, "audio quality")

			// Set quality flag.
			err = testCmd.Flags().Set("quality", string(rune(tt.qualityValue+'0')))
			require.NoError(t, err)

			// Bind flags to config.
			err = bindFlagsToConfig(testCmd.Flags(), cfg)
			require.NoError(t, err)

			// Verify quality was overridden correctly.
			assert.Equal(t, tt.expectedFormat, cfg.Quality)
		})
	}
}

// TestFlagOverrides_MinQualityValues tests min-quality flag with various values.
//
//nolint:nolintlint,tparallel // Cannot run in parallel due to Viper global state.
func TestFlagOverrides_MinQualityValues(t *testing.T) {
	testConfigContent := `
auth_token: "config_token"
quality: 3
min_quality: 0
output_path: "/config/output"
download_lyrics: false
download_speed_limit: "500KB"
log_level: "info"
track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
replace_tracks: false
replace_covers: false
replace_lyrics: false
create_folder_for_singles: false
max_folder_name_length: 100
retry_attempts_count: 3
max_download_pause: "5s"
min_retry_pause: "1s"
max_retry_pause: "3s"
max_concurrent_downloads: 1
`

	minQualityTests := []struct {
		name               string
		minQualityValue    int
		expectedMinQuality uint8
	}{
		{"min-quality 0 - no filtering", 0, 0},
		{"min-quality 1 - MP3 128 minimum", 1, 1},
		{"min-quality 2 - MP3 320 minimum", 2, 2},
		{"min-quality 3 - FLAC minimum", 3, 3},
	}

	for _, tt := range minQualityTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")

			err := os.WriteFile(
				configPath,
				[]byte(testConfigContent),
				constants.DefaultFilePermissions,
			) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Load configuration.
			cfg, err := config.LoadConfig(configPath)
			require.NoError(t, err)

			// Create a test command with flags.
			testCmd := &cobra.Command{Use: "test"}
			testCmd.Flags().IntP("min-quality", "m", 0, "minimum quality")

			// Set the min-quality flag.
			err = testCmd.Flags().Set("min-quality", strconv.Itoa(tt.minQualityValue))
			require.NoError(t, err)

			// Bind flags to config.
			err = bindFlagsToConfig(testCmd.Flags(), cfg)
			require.NoError(t, err)

			// Verify min-quality was applied.
			assert.Equal(t, tt.expectedMinQuality, cfg.MinQuality,
				"MinQuality should be set to %d", tt.expectedMinQuality)
		})
	}
}

// TestFlagOverrides_InvalidValues tests that invalid flag values are caught during validation.
//
//nolint:nolintlint,tparallel // Cannot run in parallel due to Viper global state.
func TestFlagOverrides_InvalidValues(t *testing.T) {
	invalidTests := []struct {
		name          string
		flagName      string
		flagValue     string
		expectedError string
	}{
		{
			name:          "invalid quality - too low",
			flagName:      "quality",
			flagValue:     "0",
			expectedError: "invalid quality: must be between",
		},
		{
			name:          "invalid quality - too high",
			flagName:      "quality",
			flagValue:     "4",
			expectedError: "invalid quality: must be between",
		},
		{
			name:          "invalid min-quality - too high",
			flagName:      "min-quality",
			flagValue:     "4",
			expectedError: "invalid min_quality",
		},
		{
			name:          "invalid speed limit",
			flagName:      "speed-limit",
			flagValue:     "invalid-speed",
			expectedError: "failed to parse download speed limit",
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")

			err := os.WriteFile(
				configPath,
				[]byte(testBaseConfigContent),
				constants.DefaultFilePermissions,
			) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Load configuration.
			cfg, err := config.LoadConfig(configPath)
			require.NoError(t, err)

			// Create a test command with flags.
			testCmd := &cobra.Command{Use: "test"}
			testCmd.Flags().IntP("quality", "q", 1, "audio quality")
			testCmd.Flags().IntP("min-quality", "m", 0, "minimum quality")
			testCmd.Flags().StringP("speed-limit", "s", "", "download speed limit")

			// Set the flag.
			err = testCmd.Flags().Set(tt.flagName, tt.flagValue)
			require.NoError(t, err)

			// Bind flags to config - this should fail validation.
			err = bindFlagsToConfig(testCmd.Flags(), cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestBindFlagsToConfig_UnchangedFlags tests that unchanged flags don't override config values.
//
//nolint:nolintlint,tparallel // Cannot run in parallel due to Viper global state.
func TestBindFlagsToConfig_UnchangedFlags(t *testing.T) {
	// Create temporary directory and config file.
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Use specific config content for this test.
	configContent := `
auth_token: "config_token"
quality: 2
min_quality: 0
output_path: "/config/output"
download_lyrics: true
download_speed_limit: "1MB"
log_level: "info"
track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
replace_tracks: false
replace_covers: false
replace_lyrics: false
create_folder_for_singles: false
max_folder_name_length: 100
retry_attempts_count: 3
max_download_pause: "5s"
min_retry_pause: "1s"
max_retry_pause: "3s"
max_concurrent_downloads: 1
`

	err := os.WriteFile(
		configPath,
		[]byte(configContent),
		constants.DefaultFilePermissions,
	) //nolint:gosec // It's a test file.
	require.NoError(t, err)

	// Load configuration.
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)

	// Create a test command with flags but don't set any.
	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().IntP("quality", "q", 1, "audio quality")
	testCmd.Flags().StringP("output", "o", "", "output directory")
	testCmd.Flags().BoolP("lyrics", "l", false, "include lyrics")
	testCmd.Flags().StringP("speed-limit", "s", "", "download speed limit")

	// Bind flags to config without setting any flags.
	err = bindFlagsToConfig(testCmd.Flags(), cfg)
	require.NoError(t, err)

	// Verify config values remain unchanged.
	assert.Equal(t, uint8(2), cfg.Quality)
	assert.Equal(t, "/config/output", cfg.OutputPath)
	assert.True(t, cfg.DownloadLyrics)
	assert.Equal(t, "1MB", cfg.DownloadSpeedLimit)
}

// TestBindFlagsToConfig_EmptyFlagSet tests handling of empty flag set.
func TestBindFlagsToConfig_EmptyFlagSet(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		AuthToken:              "test_token",
		Quality:                2,
		LogLevel:               "info",
		RetryAttemptsCount:     3,
		MaxDownloadPause:       "5s",
		MinRetryPause:          "1s",
		MaxRetryPause:          "3s",
		MaxConcurrentDownloads: 1,
	}

	// Create an empty flag set.
	emptyFlags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Calling with empty flag set should just validate the config.
	err := bindFlagsToConfig(emptyFlags, cfg)
	require.NoError(t, err)
}
