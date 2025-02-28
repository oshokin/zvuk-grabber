package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/oshokin/zvuk-grabber/internal/app"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	//nolint:gochecknoglobals // It is required for configuration initialization before the application starts.
	configFilenameFromFlag string

	//nolint:gochecknoglobals,lll // It is initialized once during the application's startup and shared across the command execution logic.
	appConfig *config.Config

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
			if err := bindFlagsToConfig(cmd.Flags(), appConfig); err != nil {
				logger.Fatalf(cmd.Context(), "Failed to parse flags: %v", err)
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
		_ = logger.Logger().Sync()
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
		0,
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

	logger.SetLevel(appConfig.ParsedLogLevel)
}

func bindFlagsToConfig(flags *pflag.FlagSet, cfg *config.Config) error {
	if flag := flags.Lookup("format"); flag != nil && flag.Changed {
		cfg.DownloadFormat, _ = flags.GetUint8("format")
	}

	if flag := flags.Lookup("output"); flag != nil && flag.Changed {
		cfg.OutputPath, _ = flags.GetString("output")
	}

	if flag := flags.Lookup("lyrics"); flag != nil && flag.Changed {
		cfg.DownloadLyrics, _ = flags.GetBool("lyrics")
	}

	if flag := flags.Lookup("speed-limit"); flag != nil && flag.Changed {
		cfg.DownloadSpeedLimit, _ = flags.GetString("speed-limit")
	}

	return config.ValidateConfig(cfg)
}
