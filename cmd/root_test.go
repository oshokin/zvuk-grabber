package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/files"
)

// flagOverrideWant is the subset of config flags exercised by override tests.
type flagOverrideWant struct {
	// Quality is the preferred audio quality after flag binding.
	Quality uint8
	// OutputPath is the download directory after flag binding.
	OutputPath string
	// DownloadLyrics is the lyrics flag after binding.
	DownloadLyrics bool
	// DownloadSpeedLimit is the speed-limit flag after binding.
	DownloadSpeedLimit string
}

// flagOverrideCase is one CLI-flag override scenario.
type flagOverrideCase struct {
	// Name is the subtest name.
	Name string
	// Flags are cobra flag names to set.
	Flags map[string]any
	// Want is the config subset after binding.
	Want *flagOverrideWant
}

// testBaseConfigContent is the baseline YAML fixture used by root command tests.
const testBaseConfigContent = `
zvuk_auth_token: "config_token"
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
api_retry_attempts_count: 3
max_download_pause: "5s"
api_min_retry_pause: "1s"
api_max_retry_pause: "3s"
max_concurrent_downloads: 1
`

// TestFlagOverrides tests that command-line flags correctly override configuration file values.
//
//nolint:funlen,nolintlint,tparallel // It's a comprehensive integration test. Cannot run in parallel due to Viper global state.
func TestFlagOverrides(t *testing.T) {
	tests := []*flagOverrideCase{
		{
			Name:  "no flags - use config values",
			Flags: map[string]any{},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/config/output",
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "quality flag only - override quality",
			Flags: map[string]any{
				"quality": 2,
			},
			Want: &flagOverrideWant{
				Quality:            2,
				OutputPath:         "/config/output",
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "output flag only - override output path",
			Flags: map[string]any{
				"output": "/flag/output",
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/flag/output",
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "lyrics flag only - override lyrics",
			Flags: map[string]any{
				"lyrics": true,
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/config/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "speed-limit flag only - override speed limit",
			Flags: map[string]any{
				"speed-limit": "1MB",
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/config/output",
				DownloadSpeedLimit: "1MB",
			},
		},
		{
			Name: "all flags - override everything",
			Flags: map[string]any{
				"quality":     3,
				"output":      "/all/flags/output",
				"lyrics":      true,
				"speed-limit": "2MB",
			},
			Want: &flagOverrideWant{
				Quality:            3,
				OutputPath:         "/all/flags/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "2MB",
			},
		},
		{
			Name: "quality and output flags - partial override",
			Flags: map[string]any{
				"quality": 2,
				"output":  "/partial/output",
			},
			Want: &flagOverrideWant{
				Quality:            2,
				OutputPath:         "/partial/output",
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "quality and lyrics flags - partial override",
			Flags: map[string]any{
				"quality": 3,
				"lyrics":  true,
			},
			Want: &flagOverrideWant{
				Quality:            3,
				OutputPath:         "/config/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "output and speed-limit flags - partial override",
			Flags: map[string]any{
				"output":      "/speed/output",
				"speed-limit": "3MB",
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/speed/output",
				DownloadSpeedLimit: "3MB",
			},
		},
		{
			Name: "lyrics and speed-limit flags - partial override",
			Flags: map[string]any{
				"lyrics":      true,
				"speed-limit": "750KB",
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/config/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "750KB",
			},
		},
		{
			Name: "quality, output, and lyrics flags - triple override",
			Flags: map[string]any{
				"quality": 2,
				"output":  "/triple/output",
				"lyrics":  true,
			},
			Want: &flagOverrideWant{
				Quality:            2,
				OutputPath:         "/triple/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "500KB",
			},
		},
		{
			Name: "quality, output, and speed-limit flags - triple override",
			Flags: map[string]any{
				"quality":     1,
				"output":      "/speed-triple/output",
				"speed-limit": "1.5MB",
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/speed-triple/output",
				DownloadSpeedLimit: "1.5MB",
			},
		},
		{
			Name: "quality, lyrics, and speed-limit flags - triple override",
			Flags: map[string]any{
				"quality":     3,
				"lyrics":      true,
				"speed-limit": "2.5MB",
			},
			Want: &flagOverrideWant{
				Quality:            3,
				OutputPath:         "/config/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "2.5MB",
			},
		},
		{
			Name: "output, lyrics, and speed-limit flags - triple override",
			Flags: map[string]any{
				"output":      "/another-triple/output",
				"lyrics":      true,
				"speed-limit": "100KB",
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/another-triple/output",
				DownloadLyrics:     true,
				DownloadSpeedLimit: "100KB",
			},
		},
		{
			Name: "lyrics false flag - explicit false override",
			Flags: map[string]any{
				"lyrics": false,
			},
			Want: &flagOverrideWant{
				Quality:            1,
				OutputPath:         "/config/output",
				DownloadSpeedLimit: "500KB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Create temporary directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")

			err := os.WriteFile(
				configPath,
				[]byte(testBaseConfigContent),
				files.DefaultFilePermissions,
			) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Load configuration.
			cfg, err := config.LoadConfig(configPath)
			require.NoError(t, err)

			// Create a test command with flags.
			testCmd := &cobra.Command{
				Use: "test",
			}

			testCmd.Flags().IntP("quality", "q", 1, "audio quality")
			testCmd.Flags().StringP("output", "o", "", "output directory")
			testCmd.Flags().BoolP("lyrics", "l", false, "include lyrics")
			testCmd.Flags().StringP("speed-limit", "s", "", "download speed limit")

			for flagName, flagValue := range tt.Flags {
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

			err = bindFlagsToConfig(testCmd.Flags(), cfg)
			require.NoError(t, err)

			got := &flagOverrideWant{
				Quality:            cfg.Quality,
				OutputPath:         cfg.OutputPath,
				DownloadLyrics:     cfg.DownloadLyrics,
				DownloadSpeedLimit: cfg.DownloadSpeedLimit,
			}
			assert.Equal(t, tt.Want, got)
		})
	}
}

// TestFlagOverrides_AllQualityValues tests all valid quality values (1, 2, 3).
//
//nolint:nolintlint,tparallel // Cannot run in parallel due to Viper global state.
func TestFlagOverrides_AllQualityValues(t *testing.T) {
	qualityTests := []struct {
		name           string
		qualityValue   int
		expectedFormat uint8
	}{
		{"quality 1 - MP3 128 Kbps", 1, 1},
		{"quality 2 - MP3 320 Kbps", 2, 2},
		{"quality 3 - FLAC lossless", 3, 3},
	}

	for _, tt := range qualityTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")

			err := os.WriteFile(
				configPath,
				[]byte(testBaseConfigContent),
				files.DefaultFilePermissions,
			)
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
	testConfigContent := testConfigWithReplacements("quality: 1", "quality: 3")

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
				files.DefaultFilePermissions,
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
				files.DefaultFilePermissions,
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
	configContent := testConfigWithReplacements(
		"quality: 1", "quality: 2",
		"download_lyrics: false", "download_lyrics: true",
		`download_speed_limit: "500KB"`, `download_speed_limit: "1MB"`,
	)

	err := os.WriteFile(
		configPath,
		[]byte(configContent),
		files.DefaultFilePermissions,
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

	want := &flagOverrideWant{
		Quality:            2,
		OutputPath:         "/config/output",
		DownloadLyrics:     true,
		DownloadSpeedLimit: "1MB",
	}
	got := &flagOverrideWant{
		Quality:            cfg.Quality,
		OutputPath:         cfg.OutputPath,
		DownloadLyrics:     cfg.DownloadLyrics,
		DownloadSpeedLimit: cfg.DownloadSpeedLimit,
	}
	assert.Equal(t, want, got)
}

// TestBindFlagsToConfig_EmptyFlagSet tests handling of empty flag set.
func TestBindFlagsToConfig_EmptyFlagSet(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ZvukAuthToken:          "test_token",
		Quality:                2,
		LogLevel:               "info",
		APIRetryAttemptsCount:  3,
		MaxDownloadPause:       "5s",
		APIMinRetryPause:       "1s",
		APIMaxRetryPause:       "3s",
		MaxConcurrentDownloads: 1,
	}

	// Create an empty flag set.
	emptyFlags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Calling with empty flag set should just validate the config.
	err := bindFlagsToConfig(emptyFlags, cfg)
	require.NoError(t, err)
}

// testConfigWithReplacements returns the baseline test config with selected YAML substitutions.
func testConfigWithReplacements(replacements ...string) string {
	content := testBaseConfigContent
	for i := 0; i < len(replacements); i += 2 {
		content = strings.Replace(content, replacements[i], replacements[i+1], 1)
	}

	return content
}
