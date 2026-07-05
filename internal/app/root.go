package app

import (
	"context"
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

// ExecuteRootCommand is the entry point for the application.
// It accepts mixed Zvuk and Yandex Music URLs, classifies them by provider,
// and routes each group to the corresponding provider service.
func ExecuteRootCommand(ctx context.Context, cfg *config.Config, args []string) {
	urls, err := input.Flatten(args)
	if err != nil {
		logger.Fatalf(ctx, "Failed to read input URLs: %v", err)
		return
	}

	classified := input.Classify(urls)
	for _, rawURL := range classified.Unknown {
		logger.Errorf(ctx, "Unsupported URL skipped: %s", rawURL)
	}

	templateManager := media.NewTemplateManager(ctx, cfg)
	tagProcessor := media.NewTagProcessor()

	if len(classified.Zvuk) > 0 {
		executeZvukDownloads(ctx, cfg, classified.Zvuk, templateManager, tagProcessor)
	}

	if ctx.Err() == nil && len(classified.Yandex) > 0 {
		executeYandexDownloads(ctx, cfg, classified.Yandex, templateManager, tagProcessor)
	}

	if len(classified.Zvuk) == 0 && len(classified.Yandex) == 0 && len(classified.Unknown) == 0 {
		logger.Info(ctx, "No URLs to download")
	}
}

// executeZvukDownloads initializes the Zvuk client and downloads the given URLs.
func executeZvukDownloads(
	ctx context.Context,
	cfg *config.Config,
	urls []string,
	templateManager media.TemplateManager,
	tagProcessor media.TagProcessor,
) {
	if strings.TrimSpace(cfg.ZvukAuthToken) == "" {
		logger.Fatalf(ctx, "Zvuk URLs require zvuk_auth_token. Run: zvuk-grabber auth zvuk login")
		return
	}

	zvukClient, err := zvuk_client.NewClient(cfg)
	if err != nil {
		logger.Fatalf(ctx, "Failed to initialize Zvuk client: %v", err)
		return
	}

	urlProcessor := zvuk_service.NewURLProcessor()

	s := zvuk_service.NewService(cfg, zvukClient, urlProcessor, templateManager, tagProcessor)
	defer recoverAndPrintSummary(ctx, s)

	s.DownloadURLs(ctx, urls)
}

// executeYandexDownloads initializes the Yandex client and downloads the given URLs.
func executeYandexDownloads(
	ctx context.Context,
	cfg *config.Config,
	urls []string,
	templateManager media.TemplateManager,
	tagProcessor media.TagProcessor,
) {
	if strings.TrimSpace(cfg.YandexMusicToken) == "" {
		logger.Fatalf(ctx, "Yandex Music URLs require yandex_music_token. Run: zvuk-grabber auth yandex login")
		return
	}

	yandexClient := yandex_client.NewAuthorizedClient(cfg.YandexMusicToken)

	s := yandex_service.NewService(cfg, yandexClient, templateManager, tagProcessor)
	defer recoverAndPrintSummary(ctx, s)

	s.DownloadURLs(ctx, urls)
}

// recoverAndPrintSummary recovers from panics and always prints the download summary.
func recoverAndPrintSummary(ctx context.Context, s summaryPrinter) {
	if r := recover(); r != nil {
		logger.Errorf(ctx, "Panic recovered: %v", r)
	}

	s.PrintDownloadSummary(ctx)
}
