package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testBinaryName is the name of the test binary for E2E tests.
	testBinaryName = "zvuk-grabber-test"
)

// TestMain builds the binary before running E2E tests.
func TestMain(m *testing.M) {
	// Build the binary for testing.
	//nolint:noctx // TestMain doesn't have access to context, and build is needed before tests run.
	buildCmd := exec.Command("go", "build", "-o", testBinaryName, "../.")
	if err := buildCmd.Run(); err != nil {
		os.Exit(1)
	}

	// Run tests.
	code := m.Run()

	// Cleanup.
	_ = os.Remove(testBinaryName)

	os.Exit(code)
}

// TestE2E_FlagOverrides_Format tests that --format flag overrides config.
func TestE2E_FlagOverrides_Format(t *testing.T) {
	t.Parallel()

	baseConfig := `
auth_token: "test_token_123"
download_format: 1
output_path: "/tmp/test-output"
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
`

	tests := []struct {
		name           string
		flags          []string
		expectedFormat uint8
	}{
		{
			name:           "format flag overrides to 2",
			flags:          []string{"--format", "2"},
			expectedFormat: 2,
		},
		{
			name:           "format flag overrides to 3",
			flags:          []string{"--format", "3"},
			expectedFormat: 3,
		},
		{
			name:           "no format flag uses config",
			flags:          []string{},
			expectedFormat: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")
			err := os.WriteFile(configPath, []byte(baseConfig), 0o644) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Run and get config dump.
			config := runWithConfigDump(t, configPath, tt.flags)
			require.NotNil(t, config, "Failed to get config dump")

			// Verify format was set correctly.
			assert.Equal(t, tt.expectedFormat, config.DownloadFormat,
				"Format should be %d", tt.expectedFormat)
		})
	}
}

// TestE2E_FlagOverrides_AllFlags tests all flags together.
//
//nolint:funlen // It's a comprehensive E2E test.
func TestE2E_FlagOverrides_AllFlags(t *testing.T) {
	t.Parallel()

	baseConfig := `
auth_token: "test_token_123"
download_format: 1
output_path: "/config/output"
download_lyrics: false
download_speed_limit: "500KB"
log_level: "debug"
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
`

	tests := []struct {
		name             string
		flags            []string
		expectedFormat   uint8
		expectedOutput   string
		expectedLyrics   bool
		expectedSpeedLim string
	}{
		{
			name:             "no flags - use config",
			flags:            []string{},
			expectedFormat:   1,
			expectedOutput:   "/config/output",
			expectedLyrics:   false,
			expectedSpeedLim: "500KB",
		},
		{
			name:             "format only",
			flags:            []string{"--format", "2"},
			expectedFormat:   2,
			expectedOutput:   "/config/output",
			expectedLyrics:   false,
			expectedSpeedLim: "500KB",
		},
		{
			name:             "output only",
			flags:            []string{"--output", "/flag/output"},
			expectedFormat:   1,
			expectedOutput:   "/flag/output",
			expectedLyrics:   false,
			expectedSpeedLim: "500KB",
		},
		{
			name:             "lyrics only",
			flags:            []string{"--lyrics"},
			expectedFormat:   1,
			expectedOutput:   "/config/output",
			expectedLyrics:   true,
			expectedSpeedLim: "500KB",
		},
		{
			name:             "speed-limit only",
			flags:            []string{"--speed-limit", "1MB"},
			expectedFormat:   1,
			expectedOutput:   "/config/output",
			expectedLyrics:   false,
			expectedSpeedLim: "1MB",
		},
		{
			name:             "all flags",
			flags:            []string{"--format", "3", "--output", "/all/flags", "--lyrics", "--speed-limit", "2MB"},
			expectedFormat:   3,
			expectedOutput:   "/all/flags",
			expectedLyrics:   true,
			expectedSpeedLim: "2MB",
		},
		{
			name:             "format and output",
			flags:            []string{"--format", "2", "--output", "/combo/output"},
			expectedFormat:   2,
			expectedOutput:   "/combo/output",
			expectedLyrics:   false,
			expectedSpeedLim: "500KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")
			err := os.WriteFile(configPath, []byte(baseConfig), 0o644) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Run and get config dump.
			config := runWithConfigDump(t, configPath, tt.flags)
			require.NotNil(t, config, "Failed to get config dump")

			// Verify all expected values.
			assert.Equal(t, tt.expectedFormat, config.DownloadFormat,
				"Format should be %d", tt.expectedFormat)
			assert.Equal(t, tt.expectedOutput, config.OutputPath,
				"Output path should be %s", tt.expectedOutput)
			assert.Equal(t, tt.expectedLyrics, config.DownloadLyrics,
				"Download lyrics should be %t", tt.expectedLyrics)
			assert.Equal(t, tt.expectedSpeedLim, config.DownloadSpeedLimit,
				"Speed limit should be %s", tt.expectedSpeedLim)
		})
	}
}

// TestE2E_FlagOverrides_InvalidValues tests that invalid flag values are rejected.
//
//nolint:funlen // Comprehensive E2E test with multiple test cases.
func TestE2E_FlagOverrides_InvalidValues(t *testing.T) {
	t.Parallel()

	baseConfig := `
auth_token: "test_token_123"
download_format: 1
output_path: "/tmp/test-output"
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
`

	tests := []struct {
		name             string
		flags            []string
		expectedErrorMsg string
	}{
		{
			name:             "invalid format - too low",
			flags:            []string{"--format", "0"},
			expectedErrorMsg: "invalid format",
		},
		{
			name:             "invalid format - too high",
			flags:            []string{"--format", "4"},
			expectedErrorMsg: "invalid format",
		},
		{
			name:             "invalid speed limit",
			flags:            []string{"--speed-limit", "invalid-speed"},
			expectedErrorMsg: "failed to parse download speed limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory and config file.
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test-config.yaml")
			err := os.WriteFile(configPath, []byte(baseConfig), 0o644) //nolint:gosec // It's a test file.
			require.NoError(t, err)

			// Prepare arguments.
			args := []string{
				"--config", configPath,
				"https://test-url.com/track/123",
			}
			args = append(args, tt.flags...)

			// Run the binary.
			//nolint:gosec,noctx // Test binary name is a constant, not user input. No context available in test.
			cmd := exec.Command("./"+testBinaryName, args...)
			output, err := cmd.CombinedOutput()

			// Should fail with error.
			require.Error(t, err)

			outputStr := string(output)

			// Verify error message.
			assert.Contains(t, strings.ToLower(outputStr), strings.ToLower(tt.expectedErrorMsg),
				"Expected error message about '%s' but got: %s", tt.expectedErrorMsg, outputStr)
		})
	}
}

// ConfigDump represents the config dump structure.
type ConfigDump struct {
	// DownloadFormat is the audio quality/format.
	DownloadFormat uint8 `json:"download_format"`
	// OutputPath is the directory path for downloads.
	OutputPath string `json:"output_path"`
	// DownloadLyrics indicates whether lyrics should be downloaded.
	DownloadLyrics bool `json:"download_lyrics"`
	// DownloadSpeedLimit is the speed limit for downloads.
	DownloadSpeedLimit string `json:"download_speed_limit"`
}

// runWithConfigDump runs the app with config dump enabled and parses the output.
func runWithConfigDump(t *testing.T, configPath string, flags []string) *ConfigDump {
	t.Helper()

	args := []string{
		"--config", configPath,
		"https://test-url.com/track/123",
	}
	args = append(args, flags...)

	//nolint:gosec,noctx // Test binary name is a constant, not user input. No context available in test.
	cmd := exec.Command("./"+testBinaryName, args...)

	cmd.Env = append(os.Environ(), "ZVUK_GRABBER_DUMP_CONFIG=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command failed: %v, output: %s", err, string(output))
		return nil
	}

	// Parse JSON config dump from output.
	var config ConfigDump
	if err := json.Unmarshal(output, &config); err != nil {
		t.Logf("Failed to parse config: %v, output: %s", err, string(output))
		return nil
	}

	return &config
}
