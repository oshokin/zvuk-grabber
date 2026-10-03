package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/oshokin/zvuk-grabber/internal/app"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
	"github.com/oshokin/zvuk-grabber/internal/version"
)

var (
	// configFilenameFromFlag stores the config filename provided via command-line flag.
	//
	//nolint:gochecknoglobals // It is required for configuration initialization before the application starts.
	configFilenameFromFlag string

	// appConfig stores the application configuration loaded from file and flags.
	//
	//nolint:gochecknoglobals,lll // It is initialized once during the application's startup and shared across the command execution logic.
	appConfig *config.Config

	// rootCmd is the main Cobra command for the application.
	//
	//nolint:gochecknoglobals,lll // Cobra command requires a global definition for proper command-line parsing and execution.
	rootCmd = &cobra.Command{
		Use:   "zvuk-grabber [flags] {urls}",
		Short: "Download tracks, albums and playlists from Zvuk and Yandex Music.",
		Long: `Zvuk Grabber is a CLI tool for downloading audio content from specified URLs.
It supports mixed Zvuk and Yandex Music links in one invocation.

Supported examples:
- Zvuk tracks, releases, playlists, artists, audiobooks and podcasts
- Yandex Music tracks, albums, legacy playlists and UUID playlists

The application provides flexible naming templates, quality selection, and download speed limits.`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: initConfig,
		RunE: func(cmd *cobra.Command, urls []string) error {
			// If ZVUK_GRABBER_DUMP_CONFIG is set, dump config as JSON and exit (for E2E tests).
			if os.Getenv("ZVUK_GRABBER_DUMP_CONFIG") == "1" {
				dumpConfig(appConfig)
				return nil
			}

			return app.ExecuteRootCommand(cmd.Context(), appConfig, urls)
		},
	}
)

// init registers root flags, the version command, and default run behavior.
//
//nolint:gochecknoinits // Cobra requires the init function to set up flags before the command is executed.
func init() {
	// Add version command.
	version.AttachCobraVersionCommand(rootCmd)

	rootCmd.PersistentFlags().StringVarP(
		&configFilenameFromFlag,
		"config",
		"c",
		"",
		fmt.Sprintf("path to the configuration file (default is '%s')",
			config.DefaultConfigFilename))

	rootCmdFlags := rootCmd.Flags()

	rootCmdFlags.IntP(
		"quality",
		"q",
		1,
		"audio quality: 1 = MP3, 128 Kbps, 2 = MP3, 320 Kbps, 3 = FLAC lossless.")

	rootCmdFlags.IntP(
		"min-quality",
		"m",
		0,
		"minimum acceptable quality: 1 = MP3, 128 Kbps, 2 = MP3, 320 Kbps, "+
			"3 = FLAC. Tracks below this will be skipped (0 = no filtering).")

	rootCmdFlags.StringP(
		"output",
		"o",
		"",
		"directory to save downloaded files (the path will be created if it doesn't exist).")

	rootCmdFlags.BoolP(
		"lyrics",
		"l",
		false,
		"include lyrics if available.")

	rootCmdFlags.StringP(
		"speed-limit",
		"s",
		"",
		"set per-track download limit in bytes/second, for example: 115 KiB, 500 KB, 1 MB.")

	rootCmdFlags.BoolP(
		"dry-run",
		"n",
		false,
		"preview what would be downloaded without actually downloading files.")
}

// Execute executes the root command.
func Execute() error {
	signals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM}

	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop()
	//nolint:errcheck // Console streams can reject fsync at shutdown.
	defer func() { _ = logger.Logger().Sync() }()

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	return rootCmd.ExecuteContext(ctx)
}

// initConfig loads configuration from file and binds CLI flags to it.
func initConfig(cmd *cobra.Command, _ []string) error {
	var err error

	appConfig, err = config.LoadConfig(configFilenameFromFlag)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Bind flags to config before validation.
	if err = bindFlagsToConfig(cmd.Flags(), appConfig); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	logger.SetLevel(appConfig.ParsedLogLevel)

	return nil
}

// bindFlagsToConfig applies changed CLI flags to cfg and validates the configuration.
//
//nolint:gocognit // This function handles all flag overrides, high complexity is expected.
func bindFlagsToConfig(flags *pflag.FlagSet, cfg *config.Config) error {
	var err error

	if flag := flags.Lookup("quality"); flag != nil && flag.Changed {
		var qualityValue int

		qualityValue, err = flags.GetInt("quality")
		if err != nil {
			return fmt.Errorf("failed to get quality value: %w", err)
		}

		cfg.Quality = utils.SafeIntToUint8(qualityValue)
	}

	if flag := flags.Lookup("min-quality"); flag != nil && flag.Changed {
		var minQualityValue int

		minQualityValue, err = flags.GetInt("min-quality")
		if err != nil {
			return fmt.Errorf("failed to get min-quality value: %w", err)
		}

		cfg.MinQuality = utils.SafeIntToUint8(minQualityValue)
	}

	if flag := flags.Lookup("output"); flag != nil && flag.Changed {
		cfg.OutputPath, err = flags.GetString("output")
		if err != nil {
			return fmt.Errorf("failed to get output value: %w", err)
		}
	}

	if flag := flags.Lookup("lyrics"); flag != nil && flag.Changed {
		cfg.DownloadLyrics, err = flags.GetBool("lyrics")
		if err != nil {
			return fmt.Errorf("failed to get lyrics value: %w", err)
		}
	}

	if flag := flags.Lookup("speed-limit"); flag != nil && flag.Changed {
		cfg.DownloadSpeedLimit, err = flags.GetString("speed-limit")
		if err != nil {
			return fmt.Errorf("failed to get speed limit value: %w", err)
		}
	}

	if flag := flags.Lookup("dry-run"); flag != nil && flag.Changed {
		cfg.DryRun, err = flags.GetBool("dry-run")
		if err != nil {
			return fmt.Errorf("failed to get dry-run value: %w", err)
		}
	}

	return config.ValidateConfig(cfg)
}

// dumpConfig dumps the configuration as JSON for E2E testing.
func dumpConfig(cfg *config.Config) {
	// ConfigDump is the JSON shape written by --dump-config for E2E tests.
	type ConfigDump struct {
		// Quality is the configured audio quality preset.
		Quality uint8 `json:"quality"`
		// OutputPath is the resolved download destination directory.
		OutputPath string `json:"output_path"`
		// DownloadLyrics reflects whether lyric files are enabled.
		DownloadLyrics bool `json:"download_lyrics"`
		// DownloadSpeedLimit is the configured transfer rate cap.
		DownloadSpeedLimit string `json:"download_speed_limit"`
	}

	dump := ConfigDump{
		Quality:            cfg.Quality,
		OutputPath:         cfg.OutputPath,
		DownloadLyrics:     cfg.DownloadLyrics,
		DownloadSpeedLimit: cfg.DownloadSpeedLimit,
	}

	jsonData, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		// We need to use os.Stderr here because rootCmd.ErrOrStderr() is not available in the test environment.
		fmt.Fprintf(os.Stderr, "Failed to marshal config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}
