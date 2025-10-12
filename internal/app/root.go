package app

import (
	"context"

	zvuk_client "github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	zvuk_service "github.com/oshokin/zvuk-grabber/internal/service/zvuk"
)

// ExecuteRootCommand is the entry point for the application.
// It initializes the Zvuk client, sets up the necessary service components,
// and starts the download process for the provided URLs.
func ExecuteRootCommand(ctx context.Context, cfg *config.Config, urls []string) {
	zvukClient, err := zvuk_client.NewClient(cfg)
	if err != nil {
		logger.Fatalf(ctx, "Failed to initialize zvuk client: %v", err)
	}

	urlProcessor := zvuk_service.NewURLProcessor()
	templateManager := zvuk_service.NewTemplateManager(ctx, cfg)
	tagProcessor := zvuk_service.NewTagProcessor()

	s := zvuk_service.NewService(cfg, zvukClient, urlProcessor, templateManager, tagProcessor)

	// Ensure statistics are ALWAYS printed, even on panic or os.Exit bypass.
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf(ctx, "Panic recovered: %v", r)
		}

		s.PrintDownloadSummary(ctx)
	}()

	s.DownloadURLs(ctx, urls)
}
