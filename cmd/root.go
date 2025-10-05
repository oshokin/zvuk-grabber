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
		Short: "Download tracks, albums, playlists, or an entire artist's catalog.",
		Long: `Zvuk Grabber is a CLI tool for downloading audio content from specified URLs.
It supports downloading:
- Individual tracks
- Full albums
- Playlists
- Complete catalogs of an artist

The application provides flexible naming templates, format selection, and download speed limits.`,
		Args:             cobra.MinimumNArgs(1),
		PersistentPreRun: initConfig,
		Run: func(cmd *cobra.Command, urls []string) {
			// If ZVUK_GRABBER_DUMP_CONFIG is set, dump config as JSON and exit (for E2E tests).
			if os.Getenv("ZVUK_GRABBER_DUMP_CONFIG") == "1" {
				dumpConfig(appConfig)
				return
			}

			app.ExecuteRootCommand(cmd.Context(), appConfig, urls)
		},
	}
)

// Execute executes the root command.
func Execute() {
	signals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM}
	ctx, stop := signal.NotifyContext(context.Background(), signals...)

	defer func() {
		_ = logger.Logger().Sync() //nolint:errcheck // No need to check the error here, application will exit anyway.
	}()

	defer stop()

	go func() {
		defer stop()

		err := rootCmd.ExecuteContext(ctx)
		cobra.CheckErr(err)
	}()

	<-ctx.Done()
}

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
		"format",
		"f",
		1,
		"audio format: 1 = MP3, 128 Kbps, 2 = MP3, 320 Kbps, 3 = FLAC, 16-bit/44.1kHz.")

	rootCmdFlags.StringP(
		"output",
		"o",
		"",
		"directory to save downloaded files (the path will be created if it doesn’t exist).")

	rootCmdFlags.BoolP(
		"lyrics",
		"l",
		false,
		"include lyrics if available.")

	rootCmdFlags.StringP(
		"speed-limit",
		"s",
		"",
		"set download speed limit, for example: 500 kbps, 1 mbps, 1.5 mbps.")
}

func initConfig(cmd *cobra.Command, _ []string) {
	var err error

	appConfig, err = config.LoadConfig(configFilenameFromFlag)
	if err != nil {
		logger.Fatalf(cmd.Context(), "Failed to load configuration: %v", err)
	}

	// Bind flags to config before validation.
	if err = bindFlagsToConfig(cmd.Flags(), appConfig); err != nil {
		logger.Fatalf(cmd.Context(), "Failed to parse flags: %v", err)
	}

	logger.SetLevel(appConfig.ParsedLogLevel)
}

func bindFlagsToConfig(flags *pflag.FlagSet, cfg *config.Config) error {
	var err error

	if flag := flags.Lookup("format"); flag != nil && flag.Changed {
		var formatValue int

		formatValue, err = flags.GetInt("format")
		if err != nil {
			return fmt.Errorf("failed to get format value: %w", err)
		}

		cfg.DownloadFormat = utils.SafeIntToUint8(formatValue)
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

	return config.ValidateConfig(cfg)
}

// dumpConfig dumps the configuration as JSON for E2E testing.
func dumpConfig(cfg *config.Config) {
	type ConfigDump struct {
		DownloadFormat     uint8  `json:"download_format"`
		OutputPath         string `json:"output_path"`
		DownloadLyrics     bool   `json:"download_lyrics"`
		DownloadSpeedLimit string `json:"download_speed_limit"`
	}

	dump := ConfigDump{
		DownloadFormat:     cfg.DownloadFormat,
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
