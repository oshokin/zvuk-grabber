package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	yandex_client "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/input"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/media"
	yandex_service "github.com/oshokin/zvuk-grabber/internal/service/yandex"
	zvuk_service "github.com/oshokin/zvuk-grabber/internal/service/zvuk"
)

// summaryPrinter prints a download summary after processing completes.
type summaryPrinter interface {
	// PrintDownloadSummary logs aggregated download statistics for the session.
	PrintDownloadSummary(ctx context.Context)
}

// downloadService exposes results independently from logging verbosity.
type downloadService interface {
	summaryPrinter
	// DownloadURLs downloads every URL and records per-item results.
	DownloadURLs(ctx context.Context, urls []string) error
}

// idleConnectionCloser drops HTTP keep-alives so a finished CLI run can exit.
type idleConnectionCloser interface {
	// CloseIdleConnections closes idle HTTP connections owned by the client.
	CloseIdleConnections()
}

// errUnsupportedURL is returned for URLs that are neither Zvuk nor Yandex Music.
var errUnsupportedURL = errors.New("unsupported URL")

// ExecuteRootCommand is the entry point for the application.
// It accepts mixed Zvuk and Yandex Music URLs, classifies them by provider,
// and routes each group to the corresponding provider service.
func ExecuteRootCommand(ctx context.Context, cfg *config.Config, args []string) error {
	urls, err := input.Flatten(args)
	if err != nil {
		return fmt.Errorf("failed to read input URLs: %w", err)
	}

	var failures []error

	classified := input.Classify(urls)
	for _, rawURL := range classified.Unknown {
		failures = append(failures, fmt.Errorf("%w: %s", errUnsupportedURL, rawURL))
	}

	templateManager := media.NewTemplateManager(ctx, cfg)
	tagProcessor := media.NewTagProcessor()

	if len(classified.Zvuk) > 0 {
		failures = append(failures, executeZvukDownloads(ctx, cfg, classified.Zvuk, templateManager, tagProcessor))
	}

	if ctx.Err() == nil && len(classified.Yandex) > 0 {
		failures = append(failures, executeYandexDownloads(ctx, cfg, classified.Yandex, templateManager, tagProcessor))
	}

	if len(classified.Zvuk) == 0 && len(classified.Yandex) == 0 && len(classified.Unknown) == 0 {
		logger.Info(ctx, "No URLs to download")
	}

	return errors.Join(append(failures, ctx.Err())...)
}

// executeZvukDownloads initializes the Zvuk client and downloads the given URLs.
func executeZvukDownloads(
	ctx context.Context,
	cfg *config.Config,
	urls []string,
	templateManager media.TemplateManager,
	tagProcessor media.TagProcessor,
) error {
	if strings.TrimSpace(cfg.ZvukAuthToken) == "" {
		return fmt.Errorf("%w; run: zvuk-grabber auth zvuk login", config.ErrEmptyZvukAuthToken)
	}

	zvukClient, err := zvuk_client.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Zvuk client: %w", err)
	}

	defer closeProviderIdleConnections(zvukClient)

	urlProcessor := zvuk_service.NewURLProcessor()

	s := zvuk_service.NewService(cfg, zvukClient, urlProcessor, templateManager, tagProcessor)

	return runDownloads(ctx, s, urls)
}

// executeYandexDownloads initializes the Yandex client and downloads the given URLs.
func executeYandexDownloads(
	ctx context.Context,
	cfg *config.Config,
	urls []string,
	templateManager media.TemplateManager,
	tagProcessor media.TagProcessor,
) error {
	if strings.TrimSpace(cfg.YandexMusicToken) == "" {
		return fmt.Errorf("%w; run: zvuk-grabber auth yandex login", config.ErrEmptyYandexMusicToken)
	}

	yandexClient := yandex_client.NewAuthorizedClient(cfg)
	defer yandexClient.CloseIdleConnections()

	s := yandex_service.NewService(cfg, yandexClient, templateManager, tagProcessor)

	return runDownloads(ctx, s, urls)
}

// runDownloads always prints the summary after the provider run returns.
func runDownloads(ctx context.Context, s downloadService, urls []string) error {
	defer s.PrintDownloadSummary(ctx)

	return s.DownloadURLs(ctx, urls)
}

// closeProviderIdleConnections closes idle connections when the client implements it.
func closeProviderIdleConnections(client any) {
	if closer, ok := client.(idleConnectionCloser); ok {
		closer.CloseIdleConnections()
	}
}
