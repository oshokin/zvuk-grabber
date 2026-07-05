package app

import (
	"context"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	yandexauth "github.com/oshokin/zvuk-grabber/internal/service/yandex/auth"
	zvukauth "github.com/oshokin/zvuk-grabber/internal/service/zvuk/auth"
)

// authLoginService extracts an auth token through an interactive login flow.
type authLoginService interface {
	// LoginAndExtractToken opens a browser, waits for login, and returns the extracted token.
	LoginAndExtractToken(ctx context.Context) (string, error)
}

// ExecuteZvukAuthLoginCommand executes the Zvuk auth login command.
// It opens a browser, waits for the user to log in, extracts the token,
// and saves it to the configuration file.
func ExecuteZvukAuthLoginCommand(ctx context.Context, cfg *config.Config) {
	authService, err := zvukauth.NewService(cfg)
	if err != nil {
		logger.Fatalf(ctx, "Failed to initialize Zvuk authentication service: %v", err)
		return
	}

	executeAuthLogin(ctx, cfg, "Zvuk", "zvuk_auth_token", &cfg.ZvukAuthToken, authService)
}

// ExecuteYandexAuthLoginCommand executes the Yandex Music auth login command using go-rod.
func ExecuteYandexAuthLoginCommand(ctx context.Context, cfg *config.Config) {
	executeAuthLogin(
		ctx,
		cfg,
		"Yandex Music",
		"yandex_music_token",
		&cfg.YandexMusicToken,
		yandexauth.NewAuthService(cfg),
	)
}

// executeAuthLogin runs the shared provider login flow and persists the extracted token.
func executeAuthLogin(
	ctx context.Context,
	cfg *config.Config,
	providerName string,
	configKey string,
	tokenTarget *string,
	authService authLoginService,
) {
	logger.Infof(ctx, "Starting %s authentication process", providerName)

	token, err := authService.LoginAndExtractToken(ctx)
	if err != nil {
		logger.Fatalf(ctx, "%s authentication failed: %v", providerName, err)
		return
	}

	*tokenTarget = token
	if err = config.SaveConfigFields(cfg, map[string]string{configKey: token}); err != nil {
		logger.Fatalf(ctx, "Failed to save configuration: %v", err)
		return
	}

	logger.Infof(ctx, "%s authentication complete. Token saved to %s.", providerName, configKey)
}
