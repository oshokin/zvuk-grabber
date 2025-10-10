package app

import (
	"context"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/service/auth"
)

// ExecuteAuthLoginCommand executes the auth login command.
// It opens a browser, waits for the user to log in, extracts the token,
// and saves it to the configuration file.
func ExecuteAuthLoginCommand(ctx context.Context, cfg *config.Config) {
	logger.Info(ctx, "Starting authentication process")

	// Create browser authentication service.
	authService, err := auth.NewService(cfg)
	if err != nil {
		logger.Fatalf(ctx, "Failed to initialize authentication service: %v", err)
		return
	}

	// Perform login and extract token.
	token, err := authService.LoginAndExtractToken(ctx)
	if err != nil {
		logger.Fatalf(ctx, "Authentication failed: %v", err)
		return
	}

	// Update configuration with new token.
	cfg.AuthToken = token

	// Save configuration to file.
	if err = config.SaveConfig(cfg); err != nil {
		logger.Fatalf(ctx, "Failed to save configuration: %v", err)
		return
	}

	// Print success message.
	logger.Info(ctx, "Configuration updated successfully!")
	logger.Info(ctx, "Authentication complete! You can now download music.")
	logger.Info(ctx, "")
	logger.Info(ctx, "Try downloading an album:")
	logger.Info(ctx, "zvuk-grabber https://zvuk.com/release/42393651")
	logger.Info(ctx, "")
	logger.Info(ctx, "Or a playlist:")
	logger.Info(ctx, "zvuk-grabber https://zvuk.com/playlist/9037842")
}
